package content

import (
	"context"
	"net/http"
	"testing"

	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/stretchr/testify/assert"
)

func TestNewHttpRequest(t *testing.T) {
	url := "http://www.example.com/"
	ctx := tidutils.TransactionAwareContext(context.Background(), testTID)
	req, err := newHttpRequest(ctx, http.MethodGet, url, nil)

	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, req.Method, "HTTP method")
	assert.Equal(t, url, req.URL.String(), "request URL")
	assert.Equal(t, testTID, req.Header.Get(tidutils.TransactionIDHeader), tidutils.TransactionIDHeader)
}

func TestNewHttpRequestNoTID(t *testing.T) {
	url := "http://www.example.com/"
	req, err := newHttpRequest(context.Background(), http.MethodGet, url, nil)

	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, req.Method, "HTTP method")
	assert.Equal(t, url, req.URL.String(), "request URL")
	assert.Equal(t, "", req.Header.Get(tidutils.TransactionIDHeader), tidutils.TransactionIDHeader)
}

func TestNewHttpRequestInvalidUrl(t *testing.T) {
	url := ":"
	ctx := tidutils.TransactionAwareContext(context.Background(), testTID)
	_, err := newHttpRequest(ctx, http.MethodGet, url, nil)

	assert.Error(t, err)
}
