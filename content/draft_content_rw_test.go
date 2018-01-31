package content

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestWriteContent(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	content := "{\"foo\":\"bar\"}"
	testSystemId := "foo-bar-baz"
	headers := map[string]string{
		tidutils.TransactionIDHeader: testTID,
		originSystemIdHeader:         testSystemId,
	}

	server := mockGenericRW(t, http.StatusOK, contentUUID, testSystemId, content)
	defer server.Close()

	rw := NewDraftContentRWService(server.URL)

	err := rw.Write(context.TODO(), contentUUID, &content, headers)
	assert.NoError(t, err)
}

func TestWriteContentWriterReturnsStatusCreated(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	content := "{\"foo\":\"bar\"}"
	testSystemId := "foo-bar-baz"
	headers := map[string]string{
		tidutils.TransactionIDHeader: testTID,
		originSystemIdHeader:         testSystemId,
	}

	server := mockGenericRW(t, http.StatusCreated, contentUUID, testSystemId, content)
	defer server.Close()

	rw := NewDraftContentRWService(server.URL)

	err := rw.Write(context.TODO(), contentUUID, &content, headers)
	assert.NoError(t, err)
}

func TestWriteContentWriterReturnsError(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	content := "{\"foo\":\"bar\"}"
	testSystemId := "foo-bar-baz"
	headers := map[string]string{
		tidutils.TransactionIDHeader: testTID,
		originSystemIdHeader:         testSystemId,
	}

	server := mockGenericRW(t, http.StatusServiceUnavailable, contentUUID, testSystemId, content)
	defer server.Close()

	rw := NewDraftContentRWService(server.URL)

	err := rw.Write(context.TODO(), contentUUID, &content, headers)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content RW returned an unexpected HTTP status code in write operation", "error message")
}

func mockGenericRW(t *testing.T, status int, contentUUID string, systemID string, expectedBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, fmt.Sprintf("/drafts/content/%s", contentUUID), r.URL.Path)
		assert.Equal(t, testTID, r.Header.Get(tidutils.TransactionIDHeader), tidutils.TransactionIDHeader)
		assert.Equal(t, systemID, r.Header.Get(originSystemIdHeader), originSystemIdHeader)

		by, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, expectedBody, string(by), "payload")

		w.WriteHeader(status)
	}))
}
