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
	MapNativeContent(ctx context.Context, contentUUID string, nativeBody io.Reader) (io.ReadCloser, error)
	GTG() error
	Endpoint() string
}

type draftContentMapper struct {
	pacExternalService
}

func NewDraftContentMapperService(endpoint string) DraftContentMapper {
	return &draftContentMapper{pacExternalService{endpoint, &http.Client{}}}
}

func (mapper *draftContentMapper) MapNativeContent(ctx context.Context, contentUUID string, nativeBody io.Reader) (io.ReadCloser, error) {
	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	mapLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)

	req, err := newHttpRequest(ctx, "POST", fmt.Sprintf("%s/map", mapper.endpoint), nativeBody)
	if err != nil {
		mapLog.WithError(err).Error("Error in creating the HTTP request to the mapper")
		return nil, err
	}

	resp, err := mapper.httpClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mapper returned an unexpected HTTP status code in write operation: %v", resp.StatusCode)
	}

	return resp.Body, err
}
