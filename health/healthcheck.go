package health

import (
	"fmt"
	"net/http"
	"time"

	health "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/service-status-go/gtg"
	log "github.com/sirupsen/logrus"
)

type externalService interface {
	Endpoint() string
	GTG() error
}

type Service struct {
	health.HealthCheck
	uppContentAPI  externalService
	draftContentRW externalService
	methodeMapper  externalService
	sparkValidator externalService
}

func NewHealthService(appSystemCode string, appName string, appDescription string,
	draftContent externalService, mam externalService, capi externalService, ucv externalService) *Service {
	service := &Service{draftContentRW: draftContent, methodeMapper: mam, uppContentAPI: capi, sparkValidator: ucv}
	service.SystemCode = appSystemCode
	service.Name = appName
	service.Description = appDescription
	service.Checks = []health.Check{
		service.draftContentRWCheck(),
		service.draftContentMethodeArticleMapperCheck(),
		service.contentAPICheck(),
		service.draftUppContentValidatorCheck(),
	}
	return service
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

func (service *Service) draftContentMethodeArticleMapperCheck() health.Check {
	return health.Check{
		ID:               "check-draft-content-mapper",
		BusinessImpact:   "Draft methode content cannot be provided for suggestions",
		Name:             "Check draft content mapper service",
		PanicGuide:       "https://runbooks.in.ft.com/draft-content-api",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("Draft content mapper is not available at %v", service.methodeMapper.Endpoint()),
		Checker:          externalServiceChecker(service.methodeMapper, "Draft content methode-article-mapper"),
	}
}

func (service *Service) draftUppContentValidatorCheck() health.Check {
	return health.Check{
		ID:               "check-draft-upp-content-validator",
		BusinessImpact:   "Draft spark content cannot be provided for suggestions",
		Name:             "Check upp-content-validator service",
		PanicGuide:       "https://runbooks.in.ft.com/draft-content-api",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("Draft upp content validator is not available at %v", service.sparkValidator.Endpoint()),
		Checker:          externalServiceChecker(service.sparkValidator, "Draft content upp-content-validator"),
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

func externalServiceChecker(s externalService, serviceName string) func() (string, error) {
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
