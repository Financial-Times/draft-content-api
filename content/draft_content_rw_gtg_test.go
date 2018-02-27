package content

import (
	"net/http"
	"net/http/httptest"
	"testing"

	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/stretchr/testify/assert"
)

func TestHappyDraftContentRWGTG(t *testing.T) {
	server := newGTGServerMock(t, http.StatusOK, "I am happy!")
	defer server.Close()

	client := NewDraftContentRWService(server.URL, nil)
	err := client.GTG()
	assert.NoError(t, err)
}

func TestUnhappyDraftContentRWGTG(t *testing.T) {
	server := newGTGServerMock(t, http.StatusServiceUnavailable, "I not am happy!")
	defer server.Close()

	client := NewDraftContentRWService(server.URL, nil)
	err := client.GTG()
	assert.EqualError(t, err, "gtg returned a non-200 HTTP status: 503 - I not am happy!")
}

func TestDraftContentRWGTGInvalidURL(t *testing.T) {
	client := NewDraftContentRWService(":#", nil)
	err := client.GTG()
	assert.EqualError(t, err, "gtg request error: parse :: missing protocol scheme")
}

func TestDraftContentRWGTGConnectionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	client := NewDraftContentRWService(server.URL, nil)
	err := client.GTG()
	assert.Error(t, err)
}

func newGTGServerMock(t *testing.T, httpStatus int, body string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, status.GTGPath, r.URL.Path)
		w.WriteHeader(httpStatus)
		w.Write([]byte(body))
	}))
	return ts
}
