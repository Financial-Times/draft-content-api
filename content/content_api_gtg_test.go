package content

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/go-ft-http/fthttp"
	"github.com/stretchr/testify/assert"
)

func TestHappyContentAPIGTG(t *testing.T) {
	cAPIServerMock := newContentAPIGTGServerMock(t, http.StatusOK, "I am happy!")
	defer cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL+"/content", testAPIKey, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	err := cAPI.GTG()
	assert.NoError(t, err)
}

func TestUnhappyContentAPIGTG(t *testing.T) {
	cAPIServerMock := newContentAPIGTGServerMock(t, http.StatusServiceUnavailable, "I not am happy!")
	defer cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL+"/content", testAPIKey, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	err := cAPI.GTG()
	assert.EqualError(t, err, "gtg returned a non-200 HTTP status: 503 - I not am happy!")
}

func TestContentAPIGTGWrongAPIKey(t *testing.T) {
	cAPIServerMock := newContentAPIGTGServerMock(t, http.StatusServiceUnavailable, "I not am happy!")
	defer cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL+"/content", "a-non-existing-key", fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	err := cAPI.GTG()
	assert.EqualError(t, err, "gtg returned a non-200 HTTP status: 401 - unauthorized")
}

func TestContentAPIGTGInvalidURL(t *testing.T) {
	cAPI := NewContentAPI(":#", testAPIKey, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	err := cAPI.GTG()
	assert.Error(t, err, "Missing protocol scheme in gtg request")
}

func TestContentAPIGTGConnectionError(t *testing.T) {
	cAPIServerMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	err := cAPI.GTG()
	assert.Error(t, err)
}

func newContentAPIGTGServerMock(t *testing.T, status int, body string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/content/"+syntheticContentUUID, r.URL.Path)
		if apiKey := r.Header.Get(apiKeyHeader); apiKey != testAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			_, err := w.Write([]byte("unauthorized"))
			assert.NoError(t, err)
			return
		}
		w.WriteHeader(status)
		_, err := w.Write([]byte(body))
		assert.NoError(t, err)
	}))
	return ts
}
