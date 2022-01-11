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

	svc := NewService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	err := svc.GTG()
	assert.NoError(t, err)
}

func TestUnhappyGTG(t *testing.T) {
	server := newGTGServerMock(t, http.StatusServiceUnavailable, "I not am happy!")
	defer server.Close()

	s := NewService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	err := s.GTG()
	assert.EqualError(t, err, "gtg returned a non-200 HTTP status: 503 - I not am happy!")
}

func TestGTGInvalidURL(t *testing.T) {
	svc := NewService(":#", fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	err := svc.GTG()
	assert.Error(t, err, "Missing protocol scheme in gtg request")
}

func TestGTGConnectionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	svc := NewService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	err := svc.GTG()
	assert.Error(t, err)
}

func newGTGServerMock(t *testing.T, httpStatus int, body string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, status.GTGPath, r.URL.Path)
		w.WriteHeader(httpStatus)
		_, err := w.Write([]byte(body))
		if err != nil {
			panic(err)
		}
	}))
	return ts
}

func TestEndpoint(t *testing.T) {
	testURL := "http://an-endpoint.com"
	svc := NewService(testURL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	assert.Equal(t, testURL, svc.Endpoint())
}

func TestHTTPClient(t *testing.T) {
	testClient := fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service")
	svc := NewService("http://an-endpoint.com", testClient)
	assert.Equal(t, testClient, svc.HTTPClient())
}
