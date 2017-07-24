package health

import (
	"github.com/Financial-Times/draft-content-api/content"
	health "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/service-status-go/gtg"
	"net/http"
)

type HealthService struct {
	health.HealthCheck
	contentAPI content.ContentAPI
}

func NewHealthService(appSystemCode string, appName string, appDescription string, api content.ContentAPI) *HealthService {
	service := &HealthService{contentAPI: api}
	service.SystemCode = appSystemCode
	service.Name = appName
	service.Description = appDescription
	service.Checks = []health.Check{
		service.contentAPICheck(),
	}
	return service
}

func (service *HealthService) HealthCheckHandleFunc() func(w http.ResponseWriter, r *http.Request) {
	return health.Handler(service)
}

func (service *HealthService) contentAPICheck() health.Check {
	return health.Check{
		ID:               "check-content-api-health",
		BusinessImpact:   "Impossible to serve content through PAC",
		Name:             "Check Content API Health",
		PanicGuide:       "https://dewey.ft.com/draft-content-api.html",
		Severity:         1,
		TechnicalSummary: "Content API is not available",
		Checker:          service.contentAPIChecker,
	}
}

func (service *HealthService) contentAPIChecker() (string, error) {
	if err := service.contentAPI.GTG(); err != nil {
		return "Content API is not healthy", err
	}
	return "Content API is healthy", nil

}

func (service *HealthService) GTG() gtg.Status {
	for _, check := range service.Checks {
		if _, err := check.Checker(); err != nil {
			return gtg.Status{GoodToGo: false, Message: err.Error()}
		}
	}
	return gtg.Status{GoodToGo: true}
}
