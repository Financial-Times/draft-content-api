package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Financial-Times/api-endpoint"
	"github.com/Financial-Times/draft-content-api/content"
	"github.com/Financial-Times/draft-content-api/health"
	"github.com/Financial-Times/go-ft-http/fthttp"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/husobee/vestigo"
	cli "github.com/jawher/mow.cli"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

const (
	appDescription = "PAC Draft Content"
	methodeDC      = "methode"
	articleDC      = "article"
	cphDC          = "CPH"
)

type draftConfig struct {
	draftContentEndpoint string
	contentType          string
}

func main() {
	app := cli.App("draft-content-api", appDescription)

	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "draft-content-api",
		Desc:   "System Code of the application",
		EnvVar: "APP_SYSTEM_CODE",
	})

	appName := app.String(cli.StringOpt{
		Name:   "app-name",
		Value:  "draft-content-api",
		Desc:   "Application name",
		EnvVar: "APP_NAME",
	})

	port := app.String(cli.StringOpt{
		Name:   "port",
		Value:  "8080",
		Desc:   "Port to listen on",
		EnvVar: "APP_PORT",
	})

	appTimeout := app.String(cli.StringOpt{
		Name:   "app-timeout",
		Value:  "8s",
		Desc:   "Draft Content API Response Timeout",
		EnvVar: "APP_TIMEOUT",
	})

	contentRWEndpoint := app.String(cli.StringOpt{
		Name:   "content-rw-endpoint",
		Value:  "http://localhost:8888",
		Desc:   "Endpoint for draft content",
		EnvVar: "DRAFT_CONTENT_RW_ENDPOINT",
	})

	contentEndpoint := app.String(cli.StringOpt{
		Name:   "content-endpoint",
		Value:  "http://test.api.ft.com/content",
		Desc:   "Endpoint to get content from CAPI",
		EnvVar: "CONTENT_ENDPOINT",
	})

	contentAPIKey := app.String(cli.StringOpt{
		Name:   "content-api-key",
		Value:  "",
		Desc:   "API key to access CAPI",
		EnvVar: "CAPI_APIKEY",
	})

	apiYml := app.String(cli.StringOpt{
		Name:   "api-yml",
		Value:  "./api.yml",
		Desc:   "Location of the API Swagger YML file.",
		EnvVar: "API_YML",
	})

	DraftConfig := app.Strings(cli.StringsOpt{
		Name:   "origin-IDs",
		Value:  []string{""},
		Desc:   "draft content url and its content type headers",
		EnvVar: "DRAFT_CONFIG",
	})

	originIDs := app.String(cli.StringOpt{
		Name:   "origin-IDs",
		Value:  "methode-web-pub|cct|spark-lists|spark",
		Desc:   "allowed originID header",
		EnvVar: "ORIGIN_IDS",
	})

	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	log.Infof("[Startup] %v is starting", *appSystemCode)

	app.Action = func() {
		log.Infof("System code: %s, App Name: %s, Port: %s, App Timeout: %sms", *appSystemCode, *appName, *port, *appTimeout)

		timeout, err := time.ParseDuration(*appTimeout)

		if err != nil {
			log.Errorf("App could not start, error=[%s]\n", err)
			return
		}

		httpClient := fthttp.NewClient(timeout, "PAC", *appSystemCode)

		// note: new OriginID can be added, modifying just value.yml, no code changes involved
		content.AllowedOriginSystemIdValues = getOriginID(*originIDs)

		drafts := getDraftContentMapper(*DraftConfig)
		mamService := content.NewDraftContentMapperService(drafts[methodeDC].draftContentEndpoint, httpClient)
		ucvService := content.NewSparkDraftContentMapperService(drafts[articleDC].draftContentEndpoint, httpClient)
		ucphvService := content.NewSparkDraftContentMapperService(drafts[cphDC].draftContentEndpoint, httpClient)

		contentTypeMapping := map[string]content.DraftContentMapper{
			drafts[methodeDC].contentType: mamService,
			drafts[articleDC].contentType: ucvService,
			drafts[cphDC].contentType:     ucphvService,
		}

		resolver := content.NewDraftContentMapperResolver(contentTypeMapping)
		draftContentRWService := content.NewDraftContentRWService(*contentRWEndpoint, resolver, httpClient)

		content.AllowedContentTypes = getAllowedContentType(drafts)

		cAPI := content.NewContentAPI(*contentEndpoint, *contentAPIKey, httpClient)

		contentHandler := content.NewHandler(cAPI, draftContentRWService, timeout)
		healthService := health.NewHealthService(*appSystemCode, *appName,
			appDescription, draftContentRWService, mamService, cAPI, ucvService, ucphvService)
		serveEndpoints(*port, apiYml, contentHandler, healthService)
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("App could not start, error=[%s]\n", err)
		return
	}
}

func serveEndpoints(port string, apiYml *string, contentHandler *content.Handler, healthService *health.Service) {

	r := vestigo.NewRouter()

	r.Get("/drafts/content/:uuid", contentHandler.ReadContent)
	r.Put("/drafts/nativecontent/:uuid", contentHandler.WriteNativeContent)

	if apiYml != nil {
		apiEndpoint, err := api.NewAPIEndpointForFile(*apiYml)
		if err != nil {
			log.WithError(err).WithField("file", apiYml).Warn("Failed to serve the API Endpoint for this service. Please validate the Swagger YML and the file location.")
		} else {
			r.Get(api.DefaultPath, apiEndpoint.ServeHTTP)
		}
	}

	var monitoringRouter http.Handler = r
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), monitoringRouter)
	monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)

	http.HandleFunc("/__health", healthService.HealthCheckHandleFunc())
	http.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(healthService.GTGChecker()))
	http.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)

	http.Handle("/", monitoringRouter)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Unable to start: %v", err)
	}
}

func getOriginID(s string) map[string]struct{} {

	retVal := make(map[string]struct{})
	originIDs := strings.Split(s, "|")
	if len(originIDs) > 0 {
		for _, oID := range originIDs {
			retVal[oID] = struct{}{}
		}
	}

	return retVal
}

func getDraftContentMapper(draftsConfigs []string) map[string]draftConfig {

	retVal := make(map[string]draftConfig, 0)
	for _, dc := range draftsConfigs {
		c := strings.Split(dc, "|")
		if len(c) != 3 {
			log.Warn("error getting draft config %s", c)
		}
		retVal[c[0]] = draftConfig{
			draftContentEndpoint: c[1],
			contentType:          c[2],
		}
	}

	return retVal
}

func getAllowedContentType(drafts map[string]draftConfig) map[string]struct{} {

	retVal := map[string]struct{}{}
	for _, d := range drafts {
		retVal[d.contentType] = struct{}{}
	}

	return retVal
}
