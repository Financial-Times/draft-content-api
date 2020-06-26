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
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	log "github.com/sirupsen/logrus"
)

const (
	rwURLPattern = "%s/drafts/content/%s"
)

var (
	ErrDraftNotFound                = errors.New("draft content not found in PAC")
	ErrDraftNotMappable             = errors.New("draft content is invalid for mapping")
	ErrDraftContentTypeNotSupported = errors.New("draft content-type is invalid for mapping")
)

type DraftContentRW interface {
	Read(ctx context.Context, contentUUID string) (io.ReadCloser, error)
	Write(ctx context.Context, contentUUID string, content *string, headers map[string]string) error
	GTG() error
	Endpoint() string
}

type draftContentRW struct {
	*platform.Service
	resolver DraftContentMapperResolver
}

func NewDraftContentRWService(endpoint string, resolver DraftContentMapperResolver, httpClient *http.Client) DraftContentRW {
	s := platform.NewService(endpoint, httpClient)
	return &draftContentRW{s, resolver}
}

func (rw *draftContentRW) Read(ctx context.Context, contentUUID string) (io.ReadCloser, error) {
	tid, _ := tidutils.GetTransactionIDFromContext(ctx)
	readLog := log.WithField(tidutils.TransactionIDKey, tid).WithField("uuid", contentUUID)

	// note : retrieves content from generic-Aurora-rw, the endpoint returns the same unaltered data (postman)
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
			// note aurora returns the right header for CPH (application/vnd.ft-upp-content-placeholder+json).
			contentType := resp.Header.Get(contentTypeHeader)
			mapper, resolverErr := rw.resolver.MapperForOriginIdAndContentType(contentType)

			if resolverErr != nil {
				readLog.WithError(resolverErr).Error("Unable to map content")
				return nil, resolverErr
			}

			// Note: validates content to upp-content-placeholder-validator
			mappedContent, err = mapper.MapNativeContent(ctx, contentUUID, nativeContent, contentType)

			if err != nil {
				readLog.WithError(err).Warn("Mapper error")
				switch err.(type) {
				case MapperError:
					switch err.(MapperError).MapperStatusCode() {
					case http.StatusNotFound:
						fallthrough
					case http.StatusUnsupportedMediaType:
						err = ErrDraftContentTypeNotSupported
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

	req, err := newHttpRequest(ctx, "GET", fmt.Sprintf(rwURLPattern, rw.Endpoint(), contentUUID), nil)
	if err != nil {
		readLog.WithError(err).Error("Error in creating the HTTP read request from content RW")
		return nil, err
	}

	return rw.HTTPClient().Do(req)
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
	// note : this two additions break CPH validation !
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
