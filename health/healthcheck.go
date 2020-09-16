package health

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Financial-Times/draft-content-api/config"
	health "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/service-status-go/gtg"
	log "github.com/sirupsen/logrus"
)

type ExternalService interface {
	Endpoint() string
	GTG() error
}

type Service struct {
	health.HealthCheck
	uppContentAPI  ExternalService
	draftContentRW ExternalService
}

func NewHealthService(appSystemCode string, appName string, appDescription string,
	draftContent ExternalService, capi ExternalService, hcConfig *config.Config, services []ExternalService) (*Service, error) {
	service := &Service{
		draftContentRW: draftContent,
		uppContentAPI:  capi,
	}
	service.SystemCode = appSystemCode
	service.Name = appName
	service.Description = appDescription
	service.Checks = []health.Check{
		service.draftContentRWCheck(),
		service.contentAPICheck(),
	}

	for endpoint, cfg := range hcConfig.HealthChecks {
		externalService, err := findService(endpoint, services)
		if err != nil {
			return nil, err
		}

		c := health.Check{
			ID:               cfg.ID,
			BusinessImpact:   cfg.BusinessImpact,
			Name:             cfg.Name,
			PanicGuide:       cfg.PanicGuide,
			Severity:         cfg.Severity,
			TechnicalSummary: fmt.Sprintf(cfg.TechnicalSummary, endpoint),
			Checker:          externalServiceChecker(externalService, cfg.CheckerName),
		}
		service.Checks = append(service.Checks, c)
	}

	return service, nil
}

func findService(endpoint string, services []ExternalService) (ExternalService, error) {
	for _, s := range services {
		if s.Endpoint() == endpoint {
			return s, nil
		}
	}

	return nil, fmt.Errorf("unable to find service with endpoint %v", endpoint)
}

func (service *Service) HealthCheckHandleFunc() func(w http.ResponseWriter, r *http.Request) {
	hc := health.TimedHealthCheck{
		HealthCheck: service.HealthCheck,
		Timeout:     10 * time.Second,
	}

	return health.Handler(hc)
}

func (service *Service) draftContentRWCheck() health.Check {
	return health.Check{
		ID:               "check-draft-content-rw",
		BusinessImpact:   "Draft content cannot be provided for suggestions",
		Name:             "Check draft content RW service",
		PanicGuide:       "https://runbooks.in.ft.com/draft-content-api",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("Draft content RW is not available at %v", service.draftContentRW.Endpoint()),
		Checker:          externalServiceChecker(service.draftContentRW, "Draft content RW"),
	}
}

func (service *Service) contentAPICheck() health.Check {
	return health.Check{
		ID:               "check-content-api-health",
		BusinessImpact:   "Impossible to serve content through PAC",
		Name:             "Check Content API Health",
		PanicGuide:       "https://runbooks.in.ft.com/draft-content-api",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("Content API is not available at %v", service.uppContentAPI.Endpoint()),
		Checker:          externalServiceChecker(service.uppContentAPI, "Content API"),
	}
}

func externalServiceChecker(s ExternalService, serviceName string) func() (string, error) {
	return func() (string, error) {
		if err := s.GTG(); err != nil {
			log.WithField("url", s.Endpoint()).WithError(err).Error("External service healthcehck failed")
			return fmt.Sprintf("%s is not good-to-go", serviceName), err
		}
		return fmt.Sprintf("%s is good-to-go", serviceName), nil
	}
}

func (service *Service) GTGChecker() gtg.StatusChecker {
	var fns []gtg.StatusChecker

	for _, c := range service.Checks {
		fns = append(fns, gtgCheck(c.Checker))
	}

	return gtg.FailFastParallelCheck(fns)
}

func gtgCheck(handler func() (string, error)) func() gtg.Status {
	return func() gtg.Status {
		if _, err := handler(); err != nil {
			return gtg.Status{GoodToGo: false, Message: err.Error()}
		}
		return gtg.Status{GoodToGo: true}
	}
}
