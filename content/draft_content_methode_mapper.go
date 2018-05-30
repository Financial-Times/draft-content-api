package content

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Financial-Times/draft-content-api/platform"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	log "github.com/sirupsen/logrus"
)

type draftContentMapper struct {
	*platform.Service
}

func NewDraftContentMapperService(endpoint string, httpClient *http.Client) DraftContentMapper {
	s := platform.NewService(endpoint, httpClient)
	return &draftContentMapper{s}
}

func (mapper *draftContentMapper) MapNativeContent(ctx context.Context, contentUUID string, nativeBody io.Reader, contentType string) (io.ReadCloser, error) {
	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	mapLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)

	req, err := newHttpRequest(ctx, "POST", fmt.Sprintf("%s/map?mode=suggest", mapper.Endpoint()), nativeBody)
	if err != nil {
		mapLog.WithError(err).Error("Error in creating the HTTP request to the mapper")
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := mapper.HTTPClient().Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return resp.Body, err
	}

	defer resp.Body.Close()
	return nil, MapperError{resp.StatusCode, fmt.Sprintf("mapper returned an unexpected HTTP status code in write operation: %v", resp.StatusCode)}
}
