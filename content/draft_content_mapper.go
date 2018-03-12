package content

import (
	"context"
	"fmt"
	"io"
	"net/http"

	tidutils "github.com/Financial-Times/transactionid-utils-go"
	log "github.com/sirupsen/logrus"
)

type DraftContentMapper interface {
	MapNativeContent(ctx context.Context, contentUUID string, nativeBody io.Reader, contentType string) (io.ReadCloser, error)
	GTG() error
	Endpoint() string
}

type MapperError struct {
	httpStatus int
	msg        string
}

func (e MapperError) Error() string {
	return e.msg
}

func (e MapperError) MapperStatusCode() int {
	return e.httpStatus
}

type draftContentMapper struct {
	pacExternalService
}

func NewDraftContentMapperService(endpoint string, httpClient *http.Client) DraftContentMapper {
	return &draftContentMapper{pacExternalService{endpoint, httpClient}}
}

func (mapper *draftContentMapper) MapNativeContent(ctx context.Context, contentUUID string, nativeBody io.Reader, contentType string) (io.ReadCloser, error) {
	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	mapLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)

	req, err := newHttpRequest(ctx, "POST", fmt.Sprintf("%s/map?mode=suggest", mapper.endpoint), nativeBody)
	if err != nil {
		mapLog.WithError(err).Error("Error in creating the HTTP request to the mapper")
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := mapper.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return resp.Body, err
	}

	defer resp.Body.Close()
	return nil, MapperError{resp.StatusCode, fmt.Sprintf("mapper returned an unexpected HTTP status code in write operation: %v", resp.StatusCode)}
}
