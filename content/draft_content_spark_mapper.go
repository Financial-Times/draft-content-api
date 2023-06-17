package content

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Financial-Times/draft-content-api/platform"
	"github.com/Financial-Times/go-logger/v2"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
)

type sparkDraftContentValidator struct {
	service *platform.Service
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
	log *logger.UPPLogger,
) (io.ReadCloser, error) {
	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	mapLog := log.WithField(tidutils.TransactionIDHeader, tid).WithField("uuid", contentUUID)

	req, err := newHttpRequest(ctx, "POST", validator.service.Endpoint()+"/validate", nativeBody)

	if err != nil {
		mapLog.WithError(err).Error("Error in creating the HTTP request to the UPP Validator")
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := validator.service.HTTPClient().Do(req)
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

func (validator *sparkDraftContentValidator) GTG() error {
	return validator.service.GTG()
}

func (validator *sparkDraftContentValidator) Endpoint() string {
	return validator.service.Endpoint()
}
