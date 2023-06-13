package content

import (
	"context"
	"io"
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

func TestSparkValidator(t *testing.T) {
	contentUUID := uuid.New().String()
	nativeBody := "{\"foo\":\"bar\"}"
	expectedBody := "{\"foo\":\"baz\"}"
	server := mockSparkValidatorHTTPServer(t, http.StatusOK, nativeBody, expectedBody)

	m := NewSparkDraftContentValidatorService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := m.Validate(tidutils.TransactionAwareContext(context.Background(), testTID),
		contentUUID,
		io.NopCloser(strings.NewReader(nativeBody)),
		"application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.NoError(t, err)
	defer body.Close()
	actualContent, err := io.ReadAll(body)
	assert.NoError(t, err)
	assert.Equal(t, expectedBody, string(actualContent), "mapped content")
}

func TestSparkValidatorError(t *testing.T) {
	contentUUID := uuid.New().String()
	nativeBody := "{\"foo\":\"bar2\"}"
	server := mockSparkValidatorHTTPServer(t, http.StatusServiceUnavailable, nativeBody, "")

	m := NewSparkDraftContentValidatorService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := m.Validate(tidutils.TransactionAwareContext(context.Background(), testTID),
		contentUUID,
		io.NopCloser(strings.NewReader(nativeBody)),
		"application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.Error(t, err)
	assert.Nil(t, body)
}

func TestSparkValidatorClientError(t *testing.T) {
	contentUUID := uuid.New().String()
	nativeBody := "{\"foo\":\"bar\"}"
	server := mockSparkValidatorHTTPServer(t, http.StatusBadRequest, nativeBody, "")

	m := NewSparkDraftContentValidatorService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := m.Validate(tidutils.TransactionAwareContext(context.Background(), testTID),
		contentUUID,
		ioutil.NopCloser(strings.NewReader(nativeBody)),
		"application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.Error(t, err)
	assert.Nil(t, body)
	assert.IsType(t, ValidatorError{}, err)
	assert.Equal(t, http.StatusBadRequest, err.(ValidatorError).StatusCode())
}

func TestSparkValidatorBadContent(t *testing.T) {
	contentUUID := uuid.New().String()
	nativeBody := "{\"foo\":\"bar\"}"
	server := mockSparkValidatorHTTPServer(t, http.StatusUnprocessableEntity, nativeBody, "")

	m := NewSparkDraftContentValidatorService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := m.Validate(tidutils.TransactionAwareContext(context.Background(), testTID),
		contentUUID,
		ioutil.NopCloser(strings.NewReader(nativeBody)),
		"application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")

	assert.Error(t, err)
	assert.Nil(t, body)
}

func mockSparkValidatorHTTPServer(t *testing.T, status int, expectedBody string, response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method, "HTTP method")
		assert.Equal(t, "/validate", r.URL.Path)
		assert.Equal(t, "application/vnd.ft-upp-article+json; version=1.0; charset=utf-8", r.Header.Get("Content-Type"))
		assert.Equal(t, testTID, r.Header.Get(tidutils.TransactionIDHeader), tidutils.TransactionIDHeader)

		by, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, expectedBody, string(by), "payload")

		w.WriteHeader(status)
		_, err = w.Write([]byte(response))
		assert.NoError(t, err)
	}))
}
