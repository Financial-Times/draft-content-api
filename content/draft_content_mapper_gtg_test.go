package content

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/go-ft-http/fthttp"
	"github.com/stretchr/testify/assert"
)

func TestHappyDraftContentMapperGTG(t *testing.T) {
	server := newGTGServerMock(t, http.StatusOK, "I am happy!")
	defer server.Close()

	client := NewDraftContentMapperService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	err := client.GTG()
	assert.NoError(t, err)
}

func TestUnhappyDraftContentMapperGTG(t *testing.T) {
	server := newGTGServerMock(t, http.StatusServiceUnavailable, "I not am happy!")
	defer server.Close()

	client := NewDraftContentMapperService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	err := client.GTG()
	assert.EqualError(t, err, "gtg returned a non-200 HTTP status: 503 - I not am happy!")
}

func TestDraftContentMapperGTGInvalidURL(t *testing.T) {
	client := NewDraftContentMapperService(":#", fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	err := client.GTG()
	assert.EqualError(t, err, "gtg request error: parse :: missing protocol scheme")
}

func TestDraftContentMapperGTGConnectionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	client := NewDraftContentMapperService(server.URL, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	err := client.GTG()
	assert.Error(t, err)
}
