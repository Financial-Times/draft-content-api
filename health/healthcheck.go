package health

import (
	"fmt"
	"net/http"

	health "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/service-status-go/gtg"
	"time"
)

type externalService interface {
	Endpoint() string
	GTG() error
}

type HealthService struct {
	health.HealthCheck
	uppContentAPI      externalService
	draftContentRW     externalService
	draftContentMapper externalService
}

func NewHealthService(appSystemCode string, appName string, appDescription string, draftContent externalService, mapper externalService, capi externalService) *HealthService {
	service := &HealthService{draftContentRW: draftContent, draftContentMapper: mapper, uppContentAPI: capi}
	service.SystemCode = appSystemCode
	service.Name = appName
	service.Description = appDescription
	service.Checks = []health.Check{
		service.draftContentRWCheck(),
		service.draftContentMapperCheck(),
		service.contentAPICheck(),
	}
	return service
}

func (service *HealthService) HealthCheckHandleFunc() func(w http.ResponseWriter, r *http.Request) {
	hc := health.TimedHealthCheck{
		service.HealthCheck,
		10 * time.Second,
	}

	return health.Handler(hc)
}

func (service *HealthService) draftContentRWCheck() health.Check {
	return health.Check{
		ID:               "check-draft-content-rw",
		BusinessImpact:   "Draft content cannot be provided for suggestions",
		Name:             "Check draft content RW service",
		PanicGuide:       "https://dewey.ft.com/draft-content-api.html",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("Draft content RW is not available at %v", service.draftContentRW.Endpoint()),
		Checker:          externalServiceChecker(service.draftContentRW, "Draft content RW"),
	}
}

func (service *HealthService) draftContentMapperCheck() health.Check {
	return health.Check{
		ID:               "check-draft-content-mapper",
		BusinessImpact:   "Draft content cannot be provided for suggestions",
		Name:             "Check draft content mapper service",
		PanicGuide:       "https://dewey.ft.com/draft-content-api.html",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("Draft content mapper is not available at %v", service.draftContentMapper.Endpoint()),
		Checker:          externalServiceChecker(service.draftContentMapper, "Draft content mapper"),
	}
}

func (service *HealthService) contentAPICheck() health.Check {
	return health.Check{
		ID:               "check-content-api-health",
		BusinessImpact:   "Impossible to serve content through PAC",
		Name:             "Check Content API Health",
		PanicGuide:       "https://dewey.ft.com/draft-content-api.html",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("Content API is not available at %v", service.uppContentAPI.Endpoint()),
		Checker:          externalServiceChecker(service.uppContentAPI, "Content API"),
	}
}

func externalServiceChecker(s externalService, serviceName string) func() (string, error) {
	return func() (string, error) {
		if err := s.GTG(); err != nil {
			return fmt.Sprintf("%s is not good-to-go", serviceName), err
		}
		return fmt.Sprintf("%s is good-to-go", serviceName), nil
	}
}

func (service *HealthService) GTG() gtg.Status {
	fns := []gtg.StatusChecker{}

	for _, c := range service.Checks {
		fns = append(fns, gtgCheck(c.Checker))
	}

	return gtg.FailFastParallelCheck(fns)()
}

func gtgCheck(handler func() (string, error)) func() gtg.Status {
	return func() gtg.Status {
		if _, err := handler(); err != nil {
			return gtg.Status{GoodToGo: false, Message: err.Error()}
		}
		return gtg.Status{GoodToGo: true}
	}
}
