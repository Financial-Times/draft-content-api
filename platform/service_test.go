package platform

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/go-ft-http/fthttp"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/stretchr/testify/assert"
)

func TestHappyGTG(t *testing.T) {
	server := newGTGServerMock(t, http.StatusOK, "I am happy!")
	defer server.Close()
	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	svc := NewService(server.URL, testClient)
	assert.NoError(t, svc.GTG())
}

func TestUnhappyGTG(t *testing.T) {
	server := newGTGServerMock(t, http.StatusServiceUnavailable, "I not am happy!")
	defer server.Close()
	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	svc := NewService(server.URL, testClient)
	assert.EqualError(t, svc.GTG(), "gtg returned a non-200 HTTP status: 503 - I not am happy!")
}

func TestGTGInvalidURL(t *testing.T) {
	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	svc := NewService(":#", testClient)
	assert.Error(t, svc.GTG(), "Missing protocol scheme in gtg request")
}

func TestGTGConnectionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()
	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	svc := NewService(server.URL, testClient)
	assert.Error(t, svc.GTG())
}

func newGTGServerMock(t *testing.T, httpStatus int, body string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, status.GTGPath, r.URL.Path)
		w.WriteHeader(httpStatus)
		_, err := w.Write([]byte(body))
		assert.NoError(t, err)
	}))
	return ts
}

func TestEndpoint(t *testing.T) {
	testURL := "http://an-endpoint.com"
	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	svc := NewService(testURL, testClient)
	assert.Equal(t, testURL, svc.Endpoint())
}

func TestHTTPClient(t *testing.T) {
	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	svc := NewService("http://an-endpoint.com", testClient)
	assert.Equal(t, testClient, svc.HTTPClient())
}
