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

func (cAPI *ContentAPI) get(ctx context.Context, contentUUID string, header http.Header) (*http.Response, error) {
	cAPIReqURI := cAPI.endpoint + "/" + contentUUID
	getContentLog := log.WithField("url", cAPIReqURI).WithField("uuid", contentUUID)
	tID, err := tIDUtils.GetTransactionIDFromContext(ctx)
	if err != nil {
		getContentLog.WithError(err).Warn("Transaction ID not found for request to content API")
	}
	getContentLog = getContentLog.WithField(tIDUtils.TransactionIDKey, tID)

	cAPIReq, err := http.NewRequest("GET", cAPIReqURI+"?apiKey="+cAPI.apyKey, nil)

	if err != nil {
		getContentLog.WithError(err).Error("Error in creating the http request")
		return nil, err
	}

	for k, v := range header {
		cAPIReq.Header[k] = v
	}

	getContentLog.Info("Calling Content API")
	return cAPI.httpClient.Do(cAPIReq)
}

func (cAPI *ContentAPI) GTG() error {
	cAPIReqURI := cAPI.endpoint + "/" + synteticContentUUID
	cAPIReq, err := http.NewRequest("GET", cAPIReqURI+"?apiKey="+cAPI.apyKey, nil)
	if err != nil {
		return fmt.Errorf("gtg request error: %v", err.Error())
	}

	cAPIResp, err := cAPI.httpClient.Do(cAPIReq)
	if err != nil {
		return fmt.Errorf("gtg call error: %v", err.Error())
	}
	defer cAPIResp.Body.Close()

	if cAPIResp.StatusCode != http.StatusOK {
		errMsgBody, err := ioutil.ReadAll(cAPIResp.Body)
		if err != nil {
			return errors.New("gtg returned a non-200 HTTP status")
		}
		return fmt.Errorf("gtg returned a non-200 HTTP status: %v - %v", cAPIResp.StatusCode, string(errMsgBody))
	}
	return nil
}
