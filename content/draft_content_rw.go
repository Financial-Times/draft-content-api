package content

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/Financial-Times/draft-content-api/platform"
	"github.com/Financial-Times/go-logger/v2"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
)

const (
	rwURLPattern = "%s/drafts/content/%s"
)

var (
	ErrDraftNotFound                = errors.New("draft content not found in PAC")
	ErrDraftNotValid                = errors.New("draft content is invalid")
	ErrDraftContentTypeNotSupported = errors.New("draft content-type is invalid")
)

type DraftContentRW interface {
	Read(ctx context.Context, contentUUID string, log *logger.UPPLogger) (io.ReadCloser, error)
	Write(ctx context.Context, contentUUID string, content *string, headers map[string]string, log *logger.UPPLogger) error
	GTG() error
	Endpoint() string
}

type draftContentRW struct {
	*platform.Service
	resolver DraftContentValidatorResolver
}

func NewDraftContentRWService(endpoint string, resolver DraftContentValidatorResolver, httpClient *http.Client) DraftContentRW {
	s := platform.NewService(endpoint, httpClient)
	return &draftContentRW{s, resolver}
}

func (rw *draftContentRW) Read(ctx context.Context, contentUUID string, log *logger.UPPLogger) (io.ReadCloser, error) {
	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	readLog := log.WithField(tidutils.TransactionIDHeader, tid).WithField("uuid", contentUUID)

	resp, err := rw.readNativeContent(ctx, contentUUID, log)
	if err != nil {
		readLog.WithError(err).Error("Error making the HTTP request to content RW")
		return nil, err
	}
	defer resp.Body.Close()
	var content io.ReadCloser
	switch resp.StatusCode {
	case http.StatusOK:
		var nativeContent io.Reader
		nativeContent, err = rw.constructNativeDocumentForValidator(ctx, resp.Body, resp.Header.Get("Last-Modified-RFC3339"), resp.Header.Get("Write-Request-Id"), log)

		if err == nil {
			contentType := resp.Header.Get(contentTypeHeader)
			validator, resolverErr := rw.resolver.ValidatorForContentType(contentType)

			if resolverErr != nil {
				readLog.WithError(resolverErr).Error("Unable to validate content")
				return nil, resolverErr
			}

			content, err = validator.Validate(ctx, contentUUID, nativeContent, contentType, log)

			if err != nil {
				readLog.WithError(err).Warn("Validator error")
				switch err.(type) {
				case ValidatorError:
					switch err.(ValidatorError).StatusCode() {
					case http.StatusNotFound:
						fallthrough
					case http.StatusUnsupportedMediaType:
						err = ErrDraftContentTypeNotSupported
					case http.StatusUnprocessableEntity:
						err = ErrDraftNotValid

					}
				}
			}
		} else {
			readLog.WithError(err).Warn("Error constructing validator input")
		}
	case http.StatusNotFound:
		err = ErrDraftNotFound
	default:
		return nil, fmt.Errorf("content RW returned an unexpected HTTP status code in read operation: %v", resp.StatusCode)
	}

	return content, err
}

func (rw *draftContentRW) readNativeContent(ctx context.Context, contentUUID string, log *logger.UPPLogger) (*http.Response, error) {
	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	readLog := log.WithField(tidutils.TransactionIDHeader, tid).WithField("uuid", contentUUID)

	req, err := newHttpRequest(ctx, "GET", fmt.Sprintf(rwURLPattern, rw.Endpoint(), contentUUID), nil)
	if err != nil {
		readLog.WithError(err).Error("Error in creating the HTTP read request from content RW")
		return nil, err
	}

	return rw.HTTPClient().Do(req)
}

func (rw *draftContentRW) constructNativeDocumentForValidator(ctx context.Context, rawNativeBody io.Reader, lastModified string, writeRef string, log *logger.UPPLogger) (io.Reader, error) {
	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	readLog := log.WithField(tidutils.TransactionIDHeader, tid)

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

func (rw *draftContentRW) Write(ctx context.Context, contentUUID string, content *string, headers map[string]string, log *logger.UPPLogger) error {
	tid := headers[tidutils.TransactionIDHeader]

	writeLog := log.WithField(tidutils.TransactionIDHeader, tid).WithField("uuid", contentUUID)

	req, err := newHttpRequest(ctx, "PUT", fmt.Sprintf(rwURLPattern, rw.Endpoint(), contentUUID), bytes.NewBuffer([]byte(*content)))
	if err != nil {
		writeLog.WithError(err).Error("Error in creating the HTTP write request to content RW")
		return err
	}
	req.Header.Set(tidutils.TransactionIDHeader, tid)
	req.Header.Set(originSystemIdHeader, headers[originSystemIdHeader])
	req.Header.Set(contentTypeHeader, headers[contentTypeHeader])

	resp, err := rw.HTTPClient().Do(req)
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
