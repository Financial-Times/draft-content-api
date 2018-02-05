package content

import (
	"fmt"
	"io"
	"net/http"

	tidutils "github.com/Financial-Times/transactionid-utils-go"
	log "github.com/sirupsen/logrus"
)

type DraftContentMapper interface {
	MapNativeContent(tid string, contentUUID string, nativeBody io.Reader) (io.ReadCloser, error)
	GTG() error
	Endpoint() string
}

type draftContentMapper struct {
	pacExternalService
}

func NewDraftContentMapperService(endpoint string) DraftContentMapper {
	return &draftContentMapper{pacExternalService{endpoint, &http.Client{}}}
}

func (mapper *draftContentMapper) MapNativeContent(tid string, contentUUID string, nativeBody io.Reader) (io.ReadCloser, error) {
	mapLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/map", mapper.endpoint), nativeBody)
	if err != nil {
		mapLog.WithError(err).Error("Error in creating the HTTP request to the mapper")
		return nil, err
	}
	req.Header.Set(tidutils.TransactionIDHeader, tid)

	resp, err := mapper.httpClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mapper returned an unexpected HTTP status code in write operation: %v", resp.StatusCode)
	}

	return resp.Body, err
}
