package content

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/Financial-Times/go-logger/v2"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
)

const (
	syntheticContentUUID = "4f2f97ea-b8ec-11e4-b8e6-00144feab7de"
	xPolicyHeader        = "x-policy"
)

type API struct {
	endpoint   string
	username   string
	password   string
	xPolicies  []string
	httpClient *http.Client
}

func NewContentAPI(endpoint string, username string, password string, xPolicies []string, httpClient *http.Client) *API {
	return &API{endpoint, username, password, xPolicies, httpClient}
}

func (api *API) Get(ctx context.Context, contentUUID string, log *logger.UPPLogger) (*http.Response, error) {
	apiReqURI := api.endpoint + "/" + contentUUID
	getContentLog := log.WithField("url", apiReqURI).WithField("uuid", contentUUID)
	tID, err := tidutils.GetTransactionIDFromContext(ctx)
	if err != nil {
		getContentLog.WithError(err).Warn("Transaction ID not found for request to content API")
	}
	getContentLog = getContentLog.WithField(tidutils.TransactionIDHeader, tID)

	apiReq, err := http.NewRequest("GET", apiReqURI, nil)
	if err != nil {
		getContentLog.WithError(err).Error("Error in creating the http request")
		return nil, err
	}

	for _, policy := range api.xPolicies {
		apiReq.Header.Add(xPolicyHeader, policy)
	}

	apiReq.SetBasicAuth(api.username, api.password)
	if tID != "" {
		apiReq.Header.Set(tidutils.TransactionIDHeader, tID)
	}

	getContentLog.Info("Calling Content API")
	return api.httpClient.Do(apiReq.WithContext(ctx))
}

func (api *API) GTG() error {
	apiReqURI := api.endpoint + "/" + syntheticContentUUID
	apiReq, err := http.NewRequest("GET", apiReqURI, nil)
	if err != nil {
		return fmt.Errorf("gtg request error: %v", err.Error())
	}

	apiReq.SetBasicAuth(api.username, api.password)

	apiResp, err := api.httpClient.Do(apiReq)
	if err != nil {
		return fmt.Errorf("gtg call error: %v", err.Error())
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode != http.StatusOK {
		errMsgBody, err := io.ReadAll(apiResp.Body)
		if err != nil {
			return errors.New("gtg returned a non-200 HTTP status")
		}
		return fmt.Errorf("gtg returned a non-200 HTTP status: %v - %v", apiResp.StatusCode, string(errMsgBody))
	}
	return nil
}

func (api *API) Endpoint() string {
	return api.endpoint
}
