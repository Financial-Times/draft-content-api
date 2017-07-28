package content

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHappyContentAPIGTG(t *testing.T) {
	cAPIServerMock := newContentAPIGTGServerMock(t, http.StatusOK, "I am happy!")
	defer cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL+"/content", testAPIKey)
	err := cAPI.GTG()
	assert.NoError(t, err)
}

func TestUnhappyContentAPIGTG(t *testing.T) {
	cAPIServerMock := newContentAPIGTGServerMock(t, http.StatusServiceUnavailable, "I not am happy!")
	defer cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL+"/content", testAPIKey)
	err := cAPI.GTG()
	assert.EqualError(t, err, "gtg returned a non-200 HTTP status: 503 - I not am happy!")
}

func TestContentAPIGTGWrongAPIKey(t *testing.T) {
	cAPIServerMock := newContentAPIGTGServerMock(t, http.StatusServiceUnavailable, "I not am happy!")
	defer cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL+"/content", "a-non-existing-key")
	err := cAPI.GTG()
	assert.EqualError(t, err, "gtg returned a non-200 HTTP status: 401 - unauthorized")
}

func TestContentAPIGTGInvalidURL(t *testing.T) {
	cAPI := NewContentAPI(":#", testAPIKey)
	err := cAPI.GTG()
	assert.EqualError(t, err, "gtg request error: parse :: missing protocol scheme")
}

func TestContentAPIGTGConnectionError(t *testing.T) {
	cAPI := NewContentAPI("http://a-url-that-does-not-exist.com", testAPIKey)
	err := cAPI.GTG()
	assert.Error(t, err)
}

func newContentAPIGTGServerMock(t *testing.T, status int, body string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/content/"+syntheticContentUUID, r.URL.Path)
		if values := r.URL.Query(); len(values["apiKey"]) != 1 || values["apiKey"][0] != testAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	return ts
}
