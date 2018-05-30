package content

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	tidutils "github.com/Financial-Times/transactionid-utils-go"
	log "github.com/sirupsen/logrus"
)

type sparkDraftContentMapper struct {
	endpoint string
	client   *http.Client
}

func (mapper *sparkDraftContentMapper) GTG() error {
	return nil
}

func (mapper *sparkDraftContentMapper) Endpoint() string {
	return mapper.endpoint
}

func NewSparkDraftContentMapperService(endpoint string, httpClient *http.Client) DraftContentMapper {
	return &sparkDraftContentMapper{endpoint: endpoint, client: httpClient}
}

func (mapper *sparkDraftContentMapper) MapNativeContent(ctx context.Context, contentUUID string,
	nativeBody io.Reader, contentType string) (io.ReadCloser, error) {

	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	mapLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)

	req, err := newHttpRequest(ctx, "POST", mapper.Endpoint()+"/validate", nativeBody)

	if err != nil {
		mapLog.WithError(err).Error("Error in creating the HTTP request to the spark mapper")
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := mapper.client.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return resp.Body, err
	case http.StatusUnprocessableEntity: // content body validation/mapping has failed
		fallthrough
	case http.StatusBadRequest: // json schema validation has failed
		defer resp.Body.Close()
		responseBytes, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return nil, MapperError{resp.StatusCode,
				fmt.Sprintf("Validation has failed for uuid: %s but couldn't consume response body, error: %v",
					contentUUID, err)}
		}

		responseBody := map[string]interface{}{}
		err = json.Unmarshal(responseBytes, responseBody)

		if err != nil {
			return nil, MapperError{resp.StatusCode,
				fmt.Sprintf("Validation has failed for uuid: %s but couldn't unmarshal response body, error: %v",
					contentUUID, err)}
		}

		mapperErrorMessage := fmt.Sprintf("Content with uuid: %s, content-type: %s has failed validation/mapping with reason: %v",
			contentUUID, contentType, responseBody["error"])

		return nil, MapperError{resp.StatusCode, mapperErrorMessage}

	default:
		resp.Body.Close()
		return nil, MapperError{resp.StatusCode,
			fmt.Sprintf("spark mapper returned an unexpected HTTP status code in write operation: %v",
				resp.StatusCode)}
	}

}
