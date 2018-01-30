package content

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	status "github.com/Financial-Times/service-status-go/httphandlers"
)

type pacExternalService struct {
	endpoint   string
	httpClient *http.Client
}

func (srv *pacExternalService) GTG() error {
	reqURI := srv.endpoint + status.GTGPath
	req, err := http.NewRequest("GET", reqURI, nil)
	if err != nil {
		return fmt.Errorf("gtg request error: %v", err.Error())
	}

	resp, err := srv.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gtg call error: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsgBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.New("gtg returned a non-200 HTTP status")
		}
		return fmt.Errorf("gtg returned a non-200 HTTP status: %v - %v", resp.StatusCode, string(errMsgBody))
	}
	return nil
}

func (srv *pacExternalService) Endpoint() string {
	return srv.endpoint
}
