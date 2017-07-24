package content

import (
	"context"
	"errors"
	"fmt"
	tIDUtils "github.com/Financial-Times/transactionid-utils-go"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

const synteticContentUUID = "4f2f97ea-b8ec-11e4-b8e6-00144feab7de"

type ContentAPI struct {
	endpoint   string
	apyKey     string
	httpClient *http.Client
}

func NewContentAPI(endpoint string, apiKey string) *ContentAPI {
	return &ContentAPI{endpoint, apiKey, &http.Client{}}
}

func (api *ContentAPI) get(ctx context.Context, contentUUID string, header http.Header) (*http.Response, error) {
	apiReqURI := api.endpoint + "/" + contentUUID
	getContentLog := log.WithField("url", apiReqURI).WithField("uuid", contentUUID)
	tID, err := tIDUtils.GetTransactionIDFromContext(ctx)
	if err != nil {
		getContentLog.WithError(err).Warn("Transaction ID not found for request to content API")
	}
	getContentLog = getContentLog.WithField(tIDUtils.TransactionIDKey, tID)

	apiReq, err := http.NewRequest("GET", apiReqURI+"?apiKey="+api.apyKey, nil)

	if err != nil {
		getContentLog.WithError(err).Error("Error in creating the http request")
		return nil, err
	}

	for k, v := range header {
		if k != "Accept-Encoding" { // I decided to avoid to forward this header to avoid compression of the message body
			apiReq.Header[k] = v
		}
	}

	getContentLog.Info("Calling Content API")
	return api.httpClient.Do(apiReq)
}

func (api *ContentAPI) GTG() error {
	apiReqURI := api.endpoint + "/" + synteticContentUUID
	apiReq, err := http.NewRequest("GET", apiReqURI+"?apiKey="+api.apyKey, nil)
	if err != nil {
		return fmt.Errorf("gtg request error: %v", err.Error())
	}

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