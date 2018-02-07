package content

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	tidutils "github.com/Financial-Times/transactionid-utils-go"
	log "github.com/sirupsen/logrus"
)

const (
	originSystemIdHeader = "X-Origin-System-Id"
	rwURLPattern         = "%s/drafts/content/%s"
)

var (
	allowedOriginSystemIdValues = map[string]struct{}{
		"methode-web-pub": struct{}{},
	}
)

type DraftContentRW interface {
	Write(ctx context.Context, contentUUID string, content *string, headers map[string]string) error
	GTG() error
	Endpoint() string
}

type draftContentRW struct {
	pacExternalService
}

func NewDraftContentRWService(endpoint string) DraftContentRW {
	return &draftContentRW{pacExternalService{endpoint, &http.Client{}}}
}

func (rw *draftContentRW) Write(ctx context.Context, contentUUID string, content *string, headers map[string]string) error {
	tid := headers[tidutils.TransactionIDHeader]

	writeLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)

	req, err := http.NewRequest("PUT", fmt.Sprintf(rwURLPattern, rw.endpoint, contentUUID), bytes.NewBuffer([]byte(*content)))
	if err != nil {
		writeLog.WithError(err).Error("Error in creating the HTTP write request to content RW")
		return err
	}
	req.Header.Set(tidutils.TransactionIDHeader, tid)
	req.Header.Set(originSystemIdHeader, headers[originSystemIdHeader])

	resp, err := rw.httpClient.Do(req)
	if err != nil {
		writeLog.WithError(err).Error("Error making the HTTP request to content RW")
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return nil
	default:
		return fmt.Errorf("content RW returned an unexpected HTTP status code in write operation: %v", resp.StatusCode)
	}
}
