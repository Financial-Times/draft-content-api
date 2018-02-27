package content

import (
	"bytes"
	"context"
	"encoding/json"
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
	ErrDraftNotFound = errors.New("draft content not found in PAC")
	ErrDraftNotMappable = errors.New("draft content is invalid for mapping")

	allowedOriginSystemIdValues = map[string]struct{}{
		"methode-web-pub": {},
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
	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	readLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)

	resp, err := rw.readNativeContent(ctx, contentUUID)
	if err != nil {
		readLog.WithError(err).Error("Error making the HTTP request to content RW")
		return nil, err
	}
	defer resp.Body.Close()
	var mappedContent io.ReadCloser
	switch resp.StatusCode {
	case http.StatusOK:
		var nativeContent io.Reader
		nativeContent, err = rw.constructNativeDocumentForMapper(ctx, resp.Body, resp.Header.Get("Last-Modified-RFC3339"), resp.Header.Get("Write-Request-Id"))
		if err == nil {
			mappedContent, err = rw.mapper.MapNativeContent(ctx, contentUUID, nativeContent, resp.Header.Get("Content-Type"))
			if err != nil {
				readLog.WithError(err).Warn("Mapper error")
				switch err.(type) {
				case MapperError:
					switch err.(MapperError).MapperStatusCode() {
					case http.StatusNotFound:
						fallthrough

					case http.StatusUnprocessableEntity:
						err = ErrDraftNotMappable

					}
				}
			}
		} else {
			readLog.WithError(err).Warn("Error constructing mapper input")
		}
	case http.StatusNotFound:
		err = ErrDraftNotFound
	default:
		return nil, fmt.Errorf("content RW returned an unexpected HTTP status code in read operation: %v", resp.StatusCode)
	}

	return mappedContent, err
}

func (rw *draftContentRW) readNativeContent(ctx context.Context, contentUUID string) (*http.Response, error) {
	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	readLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)

	req, err := newHttpRequest(ctx, "GET", fmt.Sprintf(rwURLPattern, rw.endpoint, contentUUID), nil)
	if err != nil {
		readLog.WithError(err).Error("Error in creating the HTTP read request from content RW")
		return nil, err
	}

	return rw.httpClient.Do(req)
}

func (rw *draftContentRW) constructNativeDocumentForMapper(ctx context.Context, rawNativeBody io.Reader, lastModified string, writeRef string) (io.Reader, error) {
	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	readLog := log.WithField(tidutils.TransactionIDKey, tid)

	rawNativeDoc := make(map[string]interface{})
	err := json.NewDecoder(rawNativeBody).Decode(&rawNativeDoc)
	if err != nil {
		readLog.WithError(err).Error("unable to unmarshal native content")
		return nil, err
	}

	rawNativeDoc["lastModified"] = lastModified
	rawNativeDoc["draftReference"] = writeRef

	nativeDoc, err := json.Marshal(&rawNativeDoc)
	if err != nil {
		readLog.WithError(err).Error("unable to marshal native content")
		return nil, err
	}

	return bytes.NewReader(nativeDoc), nil
}

func (rw *draftContentRW) Write(ctx context.Context, contentUUID string, content *string, headers map[string]string) error {
	tid := headers[tidutils.TransactionIDHeader]

	writeLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)

	req, err := newHttpRequest(ctx, "PUT", fmt.Sprintf(rwURLPattern, rw.endpoint, contentUUID), bytes.NewBuffer([]byte(*content)))
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
