package content

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/go-ft-http/fthttp"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockMapper struct {
	mock.Mock
	t                    *testing.T
	expectedDraftRef     string
	expectedLastModified string
}

const (
	testDraftRef     = "tid_draft"
	testLastModified = "2018-02-21T14:25:00Z"
)

func TestReadContent(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	nativeContent := []byte("{\"foo\":\"bar\"}")
	mappedContent := []byte("{\"foo\":\"baz\"}")
	testSystemId := "foo-bar-baz"
	ctx := tidutils.TransactionAwareContext(context.TODO(), testTID)

	rwServer := mockReadFromGenericRW(t, http.StatusOK, contentUUID, testSystemId, nativeContent, testLastModified, testDraftRef)
	defer rwServer.Close()

	mapper := mockContentMapper(t, testLastModified, testDraftRef)
	mapper.On("MapNativeContent", mock.Anything, contentUUID, mock.Anything, "application/json").Return(ioutil.NopCloser(bytes.NewReader(mappedContent)), nil)

	rw := NewDraftContentRWService(rwServer.URL, mapper, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := rw.Read(ctx, contentUUID)
	assert.NoError(t, err)
	defer body.Close()
	actual, err := ioutil.ReadAll(body)
	assert.NoError(t, err)
	assert.Equal(t, mappedContent, actual, "content")
	mapper.AssertExpectations(t)
}

func TestReadContentNotFound(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	testSystemId := "foo-bar-baz"
	ctx := tidutils.TransactionAwareContext(context.TODO(), testTID)

	rwServer := mockReadFromGenericRW(t, http.StatusNotFound, contentUUID, testSystemId, []byte("{\"message\":\"not found\"}"), "", "")
	defer rwServer.Close()

	mapper := mockContentMapper(t, "", "")

	rw := NewDraftContentRWService(rwServer.URL, mapper, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := rw.Read(ctx, contentUUID)
	assert.Error(t, err, ErrDraftNotFound.Error())
	assert.Nil(t, body, "mapped content")
	mapper.AssertExpectations(t)
}

func TestReadContentError(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	testSystemId := "foo-bar-baz"
	ctx := tidutils.TransactionAwareContext(context.TODO(), testTID)

	rwServer := mockReadFromGenericRW(t, http.StatusServiceUnavailable, contentUUID, testSystemId, []byte("{\"message\":\"service unavailable\"}"), "", "")
	defer rwServer.Close()

	mapper := mockContentMapper(t, "", "")

	rw := NewDraftContentRWService(rwServer.URL, mapper, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := rw.Read(ctx, contentUUID)
	assert.Error(t, err, "service unavailable", "r/w error")
	assert.Nil(t, body, "mapped content")
	mapper.AssertExpectations(t)
}

func TestReadContentMapperError(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	nativeContent := []byte("{\"foo\":\"bar\"}")
	testSystemId := "foo-bar-baz"
	ctx := tidutils.TransactionAwareContext(context.TODO(), testTID)

	rwServer := mockReadFromGenericRW(t, http.StatusOK, contentUUID, testSystemId, nativeContent, testLastModified, testDraftRef)
	defer rwServer.Close()

	mapper := mockContentMapper(t, testLastModified, testDraftRef)
	mapper.On("MapNativeContent", mock.Anything, mock.AnythingOfType("string"), mock.Anything, "application/json").Return(nil, errors.New("test mapper error"))

	rw := NewDraftContentRWService(rwServer.URL, mapper, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := rw.Read(ctx, contentUUID)
	assert.Error(t, err, "test mapper error")
	assert.Nil(t, body, "mapped content")
	mapper.AssertExpectations(t)
}

func TestReadContentMapperUnprocessableEntityError(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	nativeContent := []byte("{\"foo\":\"bar\"}")
	testSystemId := "foo-bar-baz"
	ctx := tidutils.TransactionAwareContext(context.TODO(), testTID)

	rwServer := mockReadFromGenericRW(t, http.StatusOK, contentUUID, testSystemId, nativeContent, testLastModified, testDraftRef)
	defer rwServer.Close()

	mapper := mockContentMapper(t, testLastModified, testDraftRef)
	mapper.On("MapNativeContent", mock.Anything, mock.AnythingOfType("string"), mock.Anything, "application/json").Return(nil, MapperError{http.StatusUnprocessableEntity, "test mapper error"})

	rw := NewDraftContentRWService(rwServer.URL, mapper, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	body, err := rw.Read(ctx, contentUUID)
	assert.EqualError(t, err, ErrDraftNotMappable.Error())
	assert.Nil(t, body, "mapped content")
	mapper.AssertExpectations(t)
}

func TestWriteContent(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	content := "{\"foo\":\"bar\"}"
	testSystemId := "foo-bar-baz"
	headers := map[string]string{
		tidutils.TransactionIDHeader: testTID,
		originSystemIdHeader:         testSystemId,
	}

	server := mockWriteToGenericRW(t, http.StatusOK, contentUUID, testSystemId, content)
	defer server.Close()

	rw := NewDraftContentRWService(server.URL, nil, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

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

	server := mockWriteToGenericRW(t, http.StatusCreated, contentUUID, testSystemId, content)
	defer server.Close()

	rw := NewDraftContentRWService(server.URL, nil, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

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

	server := mockWriteToGenericRW(t, http.StatusServiceUnavailable, contentUUID, testSystemId, content)
	defer server.Close()

	rw := NewDraftContentRWService(server.URL, nil, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	err := rw.Write(context.TODO(), contentUUID, &content, headers)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content RW returned an unexpected HTTP status code in write operation", "error message")
}

func mockReadFromGenericRW(t *testing.T, status int, contentUUID string, systemID string, body []byte, lastModified string, writeRef string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "HTTP method")
		assert.Equal(t, fmt.Sprintf("/drafts/content/%s", contentUUID), r.URL.Path)
		assert.Equal(t, testTID, r.Header.Get(tidutils.TransactionIDHeader), tidutils.TransactionIDHeader)

		w.Header().Add(originSystemIdHeader, systemID)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Write-Request-Id", writeRef)
		w.Header().Set("Last-Modified-RFC3339", lastModified)
		w.WriteHeader(status)
		w.Write(body)
	}))
}

func mockWriteToGenericRW(t *testing.T, status int, contentUUID string, systemID string, expectedBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method, "HTTP method")
		assert.Equal(t, fmt.Sprintf("/drafts/content/%s", contentUUID), r.URL.Path)
		assert.Equal(t, testTID, r.Header.Get(tidutils.TransactionIDHeader), tidutils.TransactionIDHeader)
		assert.Equal(t, systemID, r.Header.Get(originSystemIdHeader), originSystemIdHeader)
		assert.Regexp(t, `^PAC-draft-content-api/\S*\s?$`, r.Header.Get("User-Agent"), "user-agent")

		by, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, expectedBody, string(by), "payload")

		w.WriteHeader(status)
	}))
}

func mockContentMapper(t *testing.T, lastModified string, draftRef string) *mockMapper {
	return &mockMapper{mock.Mock{}, t, draftRef, lastModified}
}

func (m *mockMapper) MapNativeContent(ctx context.Context, contentUUID string, nativeBody io.Reader, contentType string) (io.ReadCloser, error) {
	args := m.Called(ctx, contentUUID, nativeBody, contentType)
	actualBody := make(map[string]interface{})
	json.NewDecoder(nativeBody).Decode(&actualBody)
	if len(m.expectedDraftRef) > 0 {
		assert.Equal(m.t, m.expectedDraftRef, actualBody["draftReference"], "draftReference")
	}
	if len(m.expectedLastModified) > 0 {
		assert.Equal(m.t, m.expectedLastModified, actualBody["lastModified"], "lastModified")
	}

	var body io.ReadCloser
	o := args.Get(0)
	if o != nil {
		body = o.(io.ReadCloser)
	}

	return body, args.Error(1)
}

func (m *mockMapper) GTG() error {
	return nil
}

func (m *mockMapper) Endpoint() string {
	return ""
}
