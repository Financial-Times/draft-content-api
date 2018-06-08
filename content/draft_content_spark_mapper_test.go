package content

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Financial-Times/go-ft-http/fthttp"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestSparkMapper(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	nativeBody := "{\"foo\":\"bar\"}"
	mappedBody := "{\"foo\":\"baz\"}"
	server := mockSparkMapperHttpServer(t, http.StatusOK, nativeBody, mappedBody)

	m := NewSparkDraftContentMapperService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := m.MapNativeContent(tidutils.TransactionAwareContext(context.Background(), testTID),
		contentUUID,
		ioutil.NopCloser(strings.NewReader(nativeBody)),
		"application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.NoError(t, err)
	defer body.Close()
	actualContent, err := ioutil.ReadAll(body)
	assert.NoError(t, err)
	assert.Equal(t, mappedBody, string(actualContent), "mapped content")
}

func TestSparkMapperError(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	nativeBody := "{\"foo\":\"bar\"}"
	server := mockSparkMapperHttpServer(t, http.StatusServiceUnavailable, nativeBody, "")

	m := NewSparkDraftContentMapperService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := m.MapNativeContent(tidutils.TransactionAwareContext(context.Background(), testTID),
		contentUUID,
		ioutil.NopCloser(strings.NewReader(nativeBody)),
		"application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.Error(t, err)
	assert.Nil(t, body)
}

func TestSparkMapperClientError(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	nativeBody := "{\"foo\":\"bar\"}"
	server := mockSparkMapperHttpServer(t, http.StatusBadRequest, nativeBody, "")

	m := NewSparkDraftContentMapperService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := m.MapNativeContent(tidutils.TransactionAwareContext(context.Background(), testTID),
		contentUUID,
		ioutil.NopCloser(strings.NewReader(nativeBody)),
		"application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.Error(t, err)
	assert.Nil(t, body)
	assert.IsType(t, MapperError{}, err)
	assert.Equal(t, http.StatusBadRequest, err.(MapperError).MapperStatusCode())
}

func TestSparkMapperBadContent(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	nativeBody := "{\"foo\":\"bar\"}"
	server := mockSparkMapperHttpServer(t, http.StatusUnprocessableEntity, nativeBody, "")

	m := NewSparkDraftContentMapperService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := m.MapNativeContent(tidutils.TransactionAwareContext(context.Background(), testTID),
		contentUUID,
		ioutil.NopCloser(strings.NewReader(nativeBody)),
		"application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.Error(t, err)
	assert.Nil(t, body)
}

func mockSparkMapperHttpServer(t *testing.T, status int, expectedBody string, response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method, "HTTP method")
		assert.Equal(t, "/validate", r.URL.Path)
		assert.Equal(t, "application/vnd.ft-upp-article+json; version=1.0; charset=utf-8", r.Header.Get("Content-Type"))
		assert.Equal(t, testTID, r.Header.Get(tidutils.TransactionIDHeader), tidutils.TransactionIDHeader)

		by, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, expectedBody, string(by), "payload")

		w.WriteHeader(status)
		w.Write([]byte(response))
	}))
}
