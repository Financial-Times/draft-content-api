package platform

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	status "github.com/Financial-Times/service-status-go/httphandlers"
)

type Service struct {
	endpoint   string
	httpClient *http.Client
}

func NewService(endpoint string, httpClient *http.Client) *Service {
	return &Service{endpoint, httpClient}
}

func (svc *Service) GTG() error {
	reqURI := svc.endpoint + status.GTGPath
	req, err := http.NewRequest("GET", reqURI, nil)
	if err != nil {
		return fmt.Errorf("gtg request error: %v", err.Error())
	}

	resp, err := svc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gtg call error: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsgBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.New("gtg returned a non-200 HTTP status")
		}
		return fmt.Errorf("gtg returned a non-200 HTTP status: %v - %v", resp.StatusCode, string(errMsgBody))
	}
	return nil
}

func (svc *Service) Endpoint() string {
	return svc.endpoint
}

func (svc *Service) HTTPClient() *http.Client {
	return svc.httpClient
}
