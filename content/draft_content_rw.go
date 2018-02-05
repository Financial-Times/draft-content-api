package content

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	tidutils "github.com/Financial-Times/transactionid-utils-go"
	log "github.com/sirupsen/logrus"
)

const (
	originSystemIdHeader = "X-Origin-System-Id"
	rwURLPattern         = "%s/drafts/content/%s"
)

var (
	ErrDraftNotFound            = errors.New("draft content not found in PAC")

	allowedOriginSystemIdValues = map[string]struct{}{
		"methode-web-pub": struct{}{},
	}
)

type DraftContentRW interface {
	Read(ctx context.Context, contentUUID string) (io.ReadCloser, error)
	Write(ctx context.Context, contentUUID string, content *string, headers map[string]string) error
	GTG() error
	Endpoint() string
}

type draftContentRW struct {
	pacExternalService
	mapper DraftContentMapper
}

func NewDraftContentRWService(endpoint string, mapper DraftContentMapper) DraftContentRW {
	return &draftContentRW{pacExternalService{endpoint, &http.Client{}}, mapper}
}

func (rw *draftContentRW) Read(ctx context.Context, contentUUID string) (io.ReadCloser, error) {
	tid, err := tidutils.GetTransactionIDFromContext(ctx)
	if err != nil {
		log.Warn("context contains no transaction id")
	}

	readLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)
	resp, err := rw.readNativeContent(tid, contentUUID)
	if err != nil {
		readLog.WithError(err).Error("Error making the HTTP request to content RW")
		return nil, err
	}
	defer resp.Body.Close()
	var mappedContent io.ReadCloser
	switch resp.StatusCode {
	case http.StatusOK:
		mappedContent, err = rw.mapper.MapNativeContent(tid, contentUUID, resp.Body)
		if err != nil {
			readLog.WithError(err).Warn("Mapper error")
		}
	case http.StatusNotFound:
		err = ErrDraftNotFound
	default:
		return nil, fmt.Errorf("content RW returned an unexpected HTTP status code in read operation: %v", resp.StatusCode)
	}

	return mappedContent, err
}

func (rw *draftContentRW) readNativeContent(tid string, contentUUID string) (*http.Response, error) {
	readLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)

	req, err := http.NewRequest("GET", fmt.Sprintf(rwURLPattern, rw.endpoint, contentUUID), nil)
	if err != nil {
		readLog.WithError(err).Error("Error in creating the HTTP read request from content RW")
		return nil, err
	}
	req.Header.Set(tidutils.TransactionIDHeader, tid)

	return rw.httpClient.Do(req)
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
