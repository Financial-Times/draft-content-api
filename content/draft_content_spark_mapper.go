package content

import (
	"net/http"
	"context"
	"io"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"github.com/Financial-Times/draft-content-api/platform"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	log "github.com/sirupsen/logrus"
)

type sparkDraftContentMapper struct {
 *platform.Service
}

func NewSparkDraftContentMapperService(endpoint string, httpClient *http.Client) DraftContentMapper {
	s := platform.NewService(endpoint, httpClient)
	return &sparkDraftContentMapper{s}
}


func (mapper *sparkDraftContentMapper) MapNativeContent(ctx context.Context, contentUUID string,
	nativeBody io.Reader, contentType string) (io.ReadCloser, error) {

	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	mapLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)

	req, err := newHttpRequest(ctx, "POST", mapper.Endpoint()+"/validate", nativeBody)

	if err != nil {
		mapLog.WithError(err).Error("Error in creating the HTTP request to the UPP Validator")
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := mapper.HTTPClient().Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return resp.Body, err
	case http.StatusUnprocessableEntity: // content body validation/mapping has failed
		fallthrough
	case http.StatusUnsupportedMediaType:
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
			fmt.Sprintf("UPP Validator returned an unexpected HTTP status code in write operation: %v",
				resp.StatusCode)}
	}

}
