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
)

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

	mamEndpoint := app.String(cli.StringOpt{
		Name:   "mam-endpoint",
		Value:  "http://localhost:11070",
		Desc:   "Endpoint for mapping Methode article draft content",
		EnvVar: "DRAFT_CONTENT_MAM_ENDPOINT",
	})

	ucvEndpoint := app.String(cli.StringOpt{
		Name:   "ucv-endpoint",
		Value:  "http://localhost:9876",
		Desc:   "Endpoint for mapping Spark article draft content",
		EnvVar: "DRAFT_CONTENT_UCV_ENDPOINT",
	})

	ucphvEndpoint := app.String(cli.StringOpt{
		Name:   "ucphv-endpoint",
		Value:  "http://localhost:9877",
		Desc:   "Endpoint for mapping Spark content placeholder draft content",
		EnvVar: "DRAFT_CONTENT_PLACEHOLDER_UCV_ENDPOINT",
	})

	liveBlogPostEndpoint := app.String(cli.StringOpt{
		Name:   "liveblogpost-endpoint",
		Value:  "http://localhost:9878",
		Desc:   "Endpoint for mapping Spark content placeholder draft content",
		EnvVar: "DRAFT_CONTENT_LIVE_BLOG_POST_ENDPOINT",
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

	originIDs := app.String(cli.StringOpt{
		Name:   "origin-IDs",
		Value:  "methode-web-pub|cct|spark-lists|spark",
		Desc:   "Allowed originID header",
		EnvVar: "ORIGIN_IDS",
	})

	methodeContentType := app.String(cli.StringOpt{
		Name:   "methode-content-type",
		Value:  "application/json",
		Desc:   "Methode content type header",
		EnvVar: "METHODE_CONTENT_TYPE",
	})

	sparkArticleContentType := app.String(cli.StringOpt{
		Name:   "spark-article-content-type",
		Value:  "application/vnd.ft-upp-article+json",
		Desc:   "Spark article content type header",
		EnvVar: "SPARK_ARTICLE_CONTENT_TYPE",
	})

	sparkCPHContentType := app.String(cli.StringOpt{
		Name:   "spark-CPH-content-type",
		Value:  "application/vnd.ft-upp-content-placeholder+json",
		Desc:   "Spark content placeholder type header",
		EnvVar: "SPARK_CPH_CONTENT_TYPE",
	})

	sparkLiveBlogPostContentType := app.String(cli.StringOpt{
		Name:   "spark-live-blog-post-content-type",
		Value:  "application/vnd.ft-upp-live-blog-post+json",
		Desc:   "Spark content placeholder type header",
		EnvVar: "SPARK_LIVE_BLOG_POST_CONTENT_TYPE",
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

		content.AllowedOriginSystemIDValues = getOriginID(*originIDs)

		mamService := content.NewDraftContentMapperService(*mamEndpoint, httpClient)
		ucvService := content.NewSparkDraftContentMapperService(*ucvEndpoint, httpClient)
		ucphvService := content.NewSparkDraftContentMapperService(*ucphvEndpoint, httpClient)
		lbpService := content.NewSparkDraftContentMapperService(*liveBlogPostEndpoint, httpClient)

		contentTypeMapping := map[string]content.DraftContentMapper{
			*methodeContentType:           mamService,
			*sparkArticleContentType:      ucvService,
			*sparkCPHContentType:          ucphvService,
			*sparkLiveBlogPostContentType: lbpService,
		}

		resolver := content.NewDraftContentMapperResolver(contentTypeMapping)
		draftContentRWService := content.NewDraftContentRWService(*contentRWEndpoint, resolver, httpClient)

		content.AllowedContentTypes = getAllowedContentType(*methodeContentType, *sparkArticleContentType, *sparkCPHContentType, *sparkLiveBlogPostContentType)

		cAPI := content.NewContentAPI(*contentEndpoint, *contentAPIKey, httpClient)

		contentHandler := content.NewHandler(cAPI, draftContentRWService, timeout)
		healthService := health.NewHealthService(*appSystemCode, *appName,
			appDescription, draftContentRWService, mamService, cAPI, ucvService, ucphvService, lbpService)
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

func getAllowedContentType(cts ...string) map[string]struct{} {

	retVal := map[string]struct{}{}
	for _, ct := range cts {
		retVal[ct] = struct{}{}
	}

	return retVal
}
