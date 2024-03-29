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

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	cAPI := NewContentAPI(cAPIServerMock.URL+"/content", testBasicAuthUsername, testBasicAuthPassword, nil, testClient)
	assert.NoError(t, cAPI.GTG())
}

func TestUnhappyContentAPIGTG(t *testing.T) {
	cAPIServerMock := newContentAPIGTGServerMock(t, http.StatusServiceUnavailable, "I not am happy!")
	defer cAPIServerMock.Close()

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	cAPI := NewContentAPI(cAPIServerMock.URL+"/content", testBasicAuthUsername, testBasicAuthPassword, nil, testClient)
	assert.EqualError(t, cAPI.GTG(), "gtg returned a non-200 HTTP status: 503 - I not am happy!")
}

func TestContentAPIGTGWrongAPIKey(t *testing.T) {
	cAPIServerMock := newContentAPIGTGServerMock(t, http.StatusServiceUnavailable, "I not am happy!")
	defer cAPIServerMock.Close()

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	cAPI := NewContentAPI(cAPIServerMock.URL+"/content", "a-non-existing-username", "a-non-existing-password", nil, testClient)
	assert.EqualError(t, cAPI.GTG(), "gtg returned a non-200 HTTP status: 401 - unauthorized")
}

func TestContentAPIGTGInvalidURL(t *testing.T) {
	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	cAPI := NewContentAPI(":#", testBasicAuthUsername, testBasicAuthPassword, nil, testClient)
	assert.Error(t, cAPI.GTG(), "Missing protocol scheme in gtg request")
}

func TestContentAPIGTGConnectionError(t *testing.T) {
	cAPIServerMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	cAPIServerMock.Close()

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	cAPI := NewContentAPI(cAPIServerMock.URL, testBasicAuthUsername, testBasicAuthPassword, nil, testClient)
	assert.Error(t, cAPI.GTG())
}

func newContentAPIGTGServerMock(t *testing.T, status int, body string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/content/"+syntheticContentUUID, r.URL.Path)
		username, password, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_, err := w.Write([]byte("unauthorized"))
			assert.NoError(t, err)
			return
		}

		if username != testBasicAuthUsername || password != testBasicAuthPassword {
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
