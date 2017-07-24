package main

import (
	"github.com/Financial-Times/draft-content-api/content"
	"github.com/Financial-Times/draft-content-api/health"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/husobee/vestigo"
	"github.com/jawher/mow.cli"
	log "github.com/sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

const appDescription = "UPP Golang Microservice Template short description - please amend"

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

	log.SetLevel(log.InfoLevel)
	log.Infof("[Startup] draft-content-api is starting ")

	app.Action = func() {
		log.Infof("System code: %s, App Name: %s, Port: %s", *appSystemCode, *appName, *port)

		cAPI := content.NewContentAPI(*contentEndpoint, *contentAPIKey)
		contentHandler := content.NewHandler(cAPI)
		healthService := health.NewHealthService(*appSystemCode, *appName, appDescription, cAPI)
		go func() {
			serveEndpoints(*port, contentHandler, healthService)
		}()

		waitForSignal()
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("App could not start, error=[%s]\n", err)
		return
	}
}

func serveEndpoints(port string, contentHandler *content.Handler, healthService *health.HealthService) {

	r := vestigo.NewRouter()

	r.Get("/drafts/content/:uuid", contentHandler.ServeHTTP)
	r.Get("/__health", healthService.HealthCheckHandleFunc())
	r.Get(status.GTGPath, status.NewGoodToGoHandler(healthService.GTG))
	r.Get(status.BuildInfoPath, status.BuildInfoHandler)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Unable to start: %v", err)
	}
}

func waitForSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
