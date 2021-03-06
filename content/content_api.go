package content

import (
	"context"
	"errors"
	"fmt"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

const apiKeyHeader = "X-Api-Key"

const syntheticContentUUID = "4f2f97ea-b8ec-11e4-b8e6-00144feab7de"

type ContentAPI interface {
	Get(ctx context.Context, contentUUID string) (*http.Response, error)
	GTG() error
	Endpoint() string
}

type contentAPI struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

func NewContentAPI(endpoint string, apiKey string, httpClient *http.Client) ContentAPI {
	return &contentAPI{endpoint, apiKey, httpClient}
}

func (api *contentAPI) Get(ctx context.Context, contentUUID string) (*http.Response, error) {
	apiReqURI := api.endpoint + "/" + contentUUID
	getContentLog := log.WithField("url", apiReqURI).WithField("uuid", contentUUID)
	tID, err := tidutils.GetTransactionIDFromContext(ctx)
	if err != nil {
		getContentLog.WithError(err).Warn("Transaction ID not found for request to content API")
	}
	getContentLog = getContentLog.WithField(tidutils.TransactionIDKey, tID)

	apiReq, err := http.NewRequest("GET", apiReqURI, nil)

	if err != nil {
		getContentLog.WithError(err).Error("Error in creating the http request")
		return nil, err
	}

	apiReq.Header.Set(apiKeyHeader, api.apiKey)
	if tID != "" {
		apiReq.Header.Set(tidutils.TransactionIDHeader, tID)
	}

	getContentLog.Info("Calling Content API")
	return api.httpClient.Do(apiReq.WithContext(ctx))
}

func (api *contentAPI) GTG() error {
	apiReqURI := api.endpoint + "/" + syntheticContentUUID
	apiReq, err := http.NewRequest("GET", apiReqURI, nil)
	if err != nil {
		return fmt.Errorf("gtg request error: %v", err.Error())
	}

	apiReq.Header.Set(apiKeyHeader, api.apiKey)

	apiResp, err := api.httpClient.Do(apiReq)
	if err != nil {
		return fmt.Errorf("gtg call error: %v", err.Error())
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode != http.StatusOK {
		errMsgBody, err := ioutil.ReadAll(apiResp.Body)
		if err != nil {
			return errors.New("gtg returned a non-200 HTTP status")
		}
		return fmt.Errorf("gtg returned a non-200 HTTP status: %v - %v", apiResp.StatusCode, string(errMsgBody))
	}
	return nil
}

func (api *contentAPI) Endpoint() string {
	return api.endpoint
}
