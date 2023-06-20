package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Financial-Times/api-endpoint"
	"github.com/Financial-Times/draft-content-api/config"
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

//TODO: Remove logrus
//TODO: Replace mapper with validator https://github.com/Financial-Times/draft-content-api/pull/39#discussion_r782967305

const (
	defaultAppName        = "draft-content-api"
	defaultAppDescription = "PAC Draft Content"
)

func main() {
	app := cli.App(defaultAppName, defaultAppDescription)

	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "draft-content-api",
		Desc:   "System Code of the application",
		EnvVar: "APP_SYSTEM_CODE",
	})

	appName := app.String(cli.StringOpt{
		Name:   "app-name",
		Value:  defaultAppName,
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

	originIDs := app.String(cli.StringOpt{
		Name:   "origin-IDs",
		Value:  "cct|spark-lists|spark",
		Desc:   "Allowed originID header",
		EnvVar: "ORIGIN_IDS",
	})

	validatorYml := app.String(cli.StringOpt{
		Name:   "validator-yml",
		Value:  "./config.yml",
		Desc:   "Location of the Validator configuration YML file.",
		EnvVar: "VALIDATOR_YML",
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

		validatorConfig, err := config.ReadConfig(*validatorYml)
		if err != nil {
			log.WithError(err).Fatal("unable to read r/w YAML configuration")
		}

		httpClient := fthttp.NewClient(timeout, "PAC", *appSystemCode)

		content.AllowedOriginSystemIDValues = getOriginID(*originIDs)

		contentTypeMapping := buildContentTypeMapping(validatorConfig, httpClient)

		resolver := content.NewDraftContentValidatorResolver(contentTypeMapping)
		draftContentRWService := content.NewDraftContentRWService(*contentRWEndpoint, resolver, httpClient)

		content.AllowedContentTypes = getAllowedContentType(validatorConfig)

		cAPI := content.NewContentAPI(*contentEndpoint, *contentAPIKey, httpClient)

		contentHandler := content.NewHandler(cAPI, draftContentRWService, timeout)
		healthService, err := health.NewHealthService(*appSystemCode, *appName, defaultAppDescription, draftContentRWService, cAPI,
			validatorConfig, extractServices(contentTypeMapping))
		if err != nil {
			log.WithError(err).Fatal("Unable to create health service")
		}

		serveEndpoints(*port, apiYml, contentHandler, healthService)
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("App could not start, error=[%s]\n", err)
		return
	}
}

func extractServices(dcm map[string]content.DraftContentValidator) []health.ExternalService {
	result := make([]health.ExternalService, 0, len(dcm))

	for _, value := range dcm {
		result = append(result, value)
	}

	return result
}

func buildContentTypeMapping(validatorConfig *config.Config, httpClient *http.Client) map[string]content.DraftContentValidator {
	contentTypeMapping := map[string]content.DraftContentValidator{}

	for contentType, cfg := range validatorConfig.ContentTypes {
		var service content.DraftContentValidator

		switch cfg.Validator {
		case "spark":
			service = content.NewSparkDraftContentValidatorService(cfg.Endpoint, httpClient)
		default:
			log.WithField("Validator", cfg.Validator).Fatal("Unknown validator")
		}
		contentTypeMapping[contentType] = service

		log.WithField("Content-Type", contentType).WithField("Endpoint", cfg.Endpoint).WithField("Validator", cfg.Validator).Info("added validator service")
	}

	return contentTypeMapping
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

func getAllowedContentType(config *config.Config) map[string]struct{} {
	retVal := map[string]struct{}{}
	for ct := range config.ContentTypes {
		retVal[ct] = struct{}{}
	}

	return retVal
}
