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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMapper(t *testing.T) {
	contentUUID := uuid.New().String()
	nativeBody := "{\"foo\":\"bar\"}"
	mappedBody := "{\"foo\":\"baz\"}"
	server := mockMapperHttpServer(t, http.StatusOK, nativeBody, mappedBody)

	m := NewDraftContentMapperService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := m.MapNativeContent(tidutils.TransactionAwareContext(context.Background(), testTID), contentUUID, ioutil.NopCloser(strings.NewReader(nativeBody)), contentType)

	assert.NoError(t, err)
	defer body.Close()
	actualContent, err := ioutil.ReadAll(body)
	assert.NoError(t, err)
	assert.Equal(t, mappedBody, string(actualContent), "mapped content")
}

func TestMapperError(t *testing.T) {
	contentUUID := uuid.New().String()
	nativeBody := "{\"foo\":\"bar\"}"
	server := mockMapperHttpServer(t, http.StatusServiceUnavailable, nativeBody, "")

	m := NewDraftContentMapperService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := m.MapNativeContent(tidutils.TransactionAwareContext(context.Background(), testTID), contentUUID, ioutil.NopCloser(strings.NewReader(nativeBody)), contentType)

	assert.Error(t, err)
	assert.Nil(t, body)
}

func TestMapperClientError(t *testing.T) {
	contentUUID := uuid.New().String()
	nativeBody := "{\"foo\":\"bar\"}"
	server := mockMapperHttpServer(t, http.StatusUnprocessableEntity, nativeBody, "")

	m := NewDraftContentMapperService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := m.MapNativeContent(tidutils.TransactionAwareContext(context.Background(), testTID), contentUUID, ioutil.NopCloser(strings.NewReader(nativeBody)), contentType)

	assert.Error(t, err)
	assert.Nil(t, body)
	assert.IsType(t, MapperError{}, err)
	assert.Equal(t, http.StatusUnprocessableEntity, err.(MapperError).MapperStatusCode())
}

func TestMapperBadContent(t *testing.T) {
	contentUUID := uuid.New().String()
	nativeBody := "{\"foo\":\"bar\"}"
	server := mockMapperHttpServer(t, http.StatusUnprocessableEntity, nativeBody, "")

	m := NewDraftContentMapperService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := m.MapNativeContent(tidutils.TransactionAwareContext(context.Background(), testTID), contentUUID, ioutil.NopCloser(strings.NewReader(nativeBody)), contentType)

	assert.Error(t, err)
	assert.Nil(t, body)
}

func mockMapperHttpServer(t *testing.T, status int, expectedBody string, response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method, "HTTP method")
		assert.Equal(t, "/map", r.URL.Path)
		assert.Equal(t, "suggest", r.URL.Query().Get("mode"))
		assert.Equal(t, contentType, r.Header.Get("Content-Type"))
		assert.Equal(t, testTID, r.Header.Get(tidutils.TransactionIDHeader), tidutils.TransactionIDHeader)

		by, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, expectedBody, string(by), "payload")

		w.WriteHeader(status)
		w.Write([]byte(response))
	}))
}
