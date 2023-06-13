package content

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Financial-Times/draft-content-api/platform"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	log "github.com/sirupsen/logrus"
)

type sparkDraftContentValidator struct {
	*platform.Service
}

func NewSparkDraftContentValidatorService(endpoint string, httpClient *http.Client) DraftContentValidator {
	s := platform.NewService(endpoint, httpClient)
	return &sparkDraftContentValidator{s}
}

func (validator *sparkDraftContentValidator) Validate(
	ctx context.Context,
	contentUUID string,
	nativeBody io.Reader,
	contentType string,
) (io.ReadCloser, error) {

	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	mapLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)

	req, err := newHttpRequest(ctx, "POST", validator.Endpoint()+"/validate", nativeBody)

	if err != nil {
		mapLog.WithError(err).Error("Error in creating the HTTP request to the UPP Validator")
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := validator.HTTPClient().Do(req)
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
		responseBytes, err := io.ReadAll(resp.Body)

		if err != nil {
			return nil, ValidatorError{resp.StatusCode,
				fmt.Sprintf(
					"Validation has failed for uuid: %s but couldn't consume response body, error: %v",
					contentUUID,
					err,
				),
			}
		}

		responseBody := make(map[string]interface{})
		err = json.Unmarshal(responseBytes, &responseBody)

		if err != nil {
			return nil, ValidatorError{resp.StatusCode,
				fmt.Sprintf(
					"Validation has failed for uuid: %s but couldn't unmarshal response body, error: %v",
					contentUUID,
					err,
				),
			}
		}

		errorMessage := fmt.Sprintf(
			"Content with uuid: %s, content-type: %s has failed validation/mapping with reason: %v",
			contentUUID,
			contentType,
			responseBody["error"],
		)

		return nil, ValidatorError{resp.StatusCode, errorMessage}

	default:
		resp.Body.Close()
		return nil, ValidatorError{resp.StatusCode,
			fmt.Sprintf(
				"UPP Validator returned an unexpected HTTP status code in write operation: %v",
				resp.StatusCode,
			),
		}
	}

}
