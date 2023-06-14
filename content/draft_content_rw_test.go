package content

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/go-ft-http/fthttp"
	"github.com/Financial-Times/go-logger/v2"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockValidator struct {
	expectedDraftRef     string
	expectedLastModified string
	mock                 mock.Mock
	t                    *testing.T
	log                  *logger.UPPLogger
}

const (
	testDraftRef     = "tid_draft"
	testLastModified = "2018-02-21T14:25:00Z"
	testContentType  = "application/cobol"
)

func TestReadContent(t *testing.T) {
	contentUUID := uuid.New().String()
	nativeContent := []byte("{\"foo\":\"bar\"}")
	expectedContent := []byte("{\"foo\":\"baz\"}")
	testSystemID := "foo-bar-baz"
	ctx := tidutils.TransactionAwareContext(context.TODO(), testTID)
	testLogger := logger.NewUPPLogger(testSystemID, "debug")

	rwServer := mockReadFromGenericRW(t, http.StatusOK, contentUUID, testSystemID, nativeContent, testLastModified, testDraftRef)
	defer rwServer.Close()

	validator := mockContentValidator(t, testLastModified, testDraftRef)
	validator.mock.On("Validate", mock.Anything, contentUUID, mock.Anything, contentTypeArticle, mock.Anything).Return(io.NopCloser(bytes.NewReader(expectedContent)), nil)

	resolver := NewDraftContentValidatorResolver(
		cctOnlyResolverConfig(
			validator,
		),
	)

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	rw := NewDraftContentRWService(rwServer.URL, resolver, testClient)

	body, err := rw.Read(ctx, contentUUID, testLogger)
	assert.NoError(t, err)
	defer body.Close()
	actual, err := io.ReadAll(body)
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, actual, "content")
	validator.mock.AssertExpectations(t)
}

func TestReadContentNotFound(t *testing.T) {
	contentUUID := uuid.New().String()
	testSystemID := "foo-bar-baz"
	ctx := tidutils.TransactionAwareContext(context.TODO(), testTID)
	testLogger := logger.NewUPPLogger(testSystemID, "debug")

	rwServer := mockReadFromGenericRW(t, http.StatusNotFound, contentUUID, testSystemID, []byte("{\"message\":\"not found\"}"), "", "")
	defer rwServer.Close()

	validator := mockContentValidator(t, "", "")

	resolver := NewDraftContentValidatorResolver(cctOnlyResolverConfig(validator))

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	rw := NewDraftContentRWService(rwServer.URL, resolver, testClient)

	body, err := rw.Read(ctx, contentUUID, testLogger)
	assert.Error(t, err, ErrDraftNotFound.Error())
	assert.Nil(t, body, "mapped content")
	validator.mock.AssertExpectations(t)
}

func TestReadContentError(t *testing.T) {
	contentUUID := uuid.New().String()
	testSystemID := "foo-bar-baz"
	ctx := tidutils.TransactionAwareContext(context.TODO(), testTID)
	testLogger := logger.NewUPPLogger(testSystemID, "debug")

	rwServer := mockReadFromGenericRW(t, http.StatusServiceUnavailable, contentUUID, testSystemID, []byte("{\"message\":\"service unavailable\"}"), "", "")
	defer rwServer.Close()

	validator := mockContentValidator(t, "", "")
	resolver := NewDraftContentValidatorResolver(cctOnlyResolverConfig(validator))

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	rw := NewDraftContentRWService(rwServer.URL, resolver, testClient)

	body, err := rw.Read(ctx, contentUUID, testLogger)
	assert.Error(t, err, "service unavailable", "r/w error")
	assert.Nil(t, body, "mapped content")
	validator.mock.AssertExpectations(t)
}

func TestReadContentValidatorError(t *testing.T) {
	contentUUID := uuid.New().String()
	nativeContent := []byte("{\"foo\":\"bar\"}")
	testSystemID := "foo-bar-baz"
	ctx := tidutils.TransactionAwareContext(context.TODO(), testTID)
	testLogger := logger.NewUPPLogger(testSystemID, "debug")

	rwServer := mockReadFromGenericRW(t, http.StatusOK, contentUUID, testSystemID, nativeContent, testLastModified, testDraftRef)
	defer rwServer.Close()

	validator := mockContentValidator(t, testLastModified, testDraftRef)
	validator.mock.On("Validate", mock.Anything, mock.AnythingOfType("string"), mock.Anything, contentTypeArticle).Return(nil, errors.New("test validator error"))

	resolver := NewDraftContentValidatorResolver(cctOnlyResolverConfig(validator))
	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	rw := NewDraftContentRWService(rwServer.URL, resolver, testClient)

	body, err := rw.Read(ctx, contentUUID, testLogger)
	assert.Error(t, err, "test validator error")
	assert.Nil(t, body, "mapped content")
	validator.mock.AssertExpectations(t)
}

func TestReadContentValidatorUnprocessableEntityError(t *testing.T) {
	contentUUID := uuid.New().String()
	nativeContent := []byte("{\"foo\":\"bar\"}")
	testSystemID := "foo-bar-baz"
	ctx := tidutils.TransactionAwareContext(context.TODO(), testTID)
	testLogger := logger.NewUPPLogger(testSystemID, "debug")

	rwServer := mockReadFromGenericRW(t, http.StatusOK, contentUUID, testSystemID, nativeContent, testLastModified, testDraftRef)
	defer rwServer.Close()

	validator := mockContentValidator(t, testLastModified, testDraftRef)
	validator.mock.On("Validate", mock.Anything, mock.AnythingOfType("string"), mock.Anything, contentTypeArticle).Return(nil, ValidatorError{http.StatusUnprocessableEntity, "test validator error"})
	resolver := NewDraftContentValidatorResolver(cctOnlyResolverConfig(validator))

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	rw := NewDraftContentRWService(rwServer.URL, resolver, testClient)

	body, err := rw.Read(ctx, contentUUID, testLogger)
	assert.EqualError(t, err, ErrDraftNotValid.Error())
	assert.Nil(t, body, "mapped content")
	validator.mock.AssertExpectations(t)
}

func TestWriteContent(t *testing.T) {
	contentUUID := uuid.New().String()
	content := "{\"foo\":\"bar\"}"
	testSystemID := "foo-bar-baz"
	testLogger := logger.NewUPPLogger(testSystemID, "debug")
	headers := map[string]string{
		tidutils.TransactionIDHeader: testTID,
		originSystemIdHeader:         testSystemID,
		contentTypeHeader:            testContentType,
	}

	server := mockWriteToGenericRW(t, http.StatusOK, contentUUID, testSystemID, content, testContentType)
	defer server.Close()

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	rw := NewDraftContentRWService(server.URL, nil, testClient)
	assert.NoError(t, rw.Write(context.TODO(), contentUUID, &content, headers, testLogger))
}

func TestWriteContentWriterReturnsStatusCreated(t *testing.T) {
	contentUUID := uuid.New().String()
	content := "{\"foo\":\"bar\"}"
	testSystemID := "foo-bar-baz"
	testLogger := logger.NewUPPLogger(testSystemID, "debug")
	headers := map[string]string{
		tidutils.TransactionIDHeader: testTID,
		originSystemIdHeader:         testSystemID,
		contentTypeHeader:            testContentType,
	}

	server := mockWriteToGenericRW(t, http.StatusCreated, contentUUID, testSystemID, content, testContentType)
	defer server.Close()

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	rw := NewDraftContentRWService(server.URL, nil, testClient)
	assert.NoError(t, rw.Write(context.TODO(), contentUUID, &content, headers, testLogger))
}

func TestWriteContentWriterReturnsError(t *testing.T) {
	contentUUID := uuid.New().String()
	content := "{\"foo\":\"bar\"}"
	testSystemID := "foo-bar-baz"
	testLogger := logger.NewUPPLogger(testSystemID, "debug")
	headers := map[string]string{
		tidutils.TransactionIDHeader: testTID,
		originSystemIdHeader:         testSystemID,
		contentTypeHeader:            testContentType,
	}

	server := mockWriteToGenericRW(t, http.StatusServiceUnavailable, contentUUID, testSystemID, content, testContentType)
	defer server.Close()

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	rw := NewDraftContentRWService(server.URL, nil, testClient)
	err = rw.Write(context.TODO(), contentUUID, &content, headers, testLogger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content RW returned an unexpected HTTP status code in write operation", "error message")
}

func mockReadFromGenericRW(t *testing.T, status int, contentUUID string, systemID string, body []byte, lastModified string, writeRef string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "HTTP method")
		assert.Equal(t, fmt.Sprintf("/drafts/content/%s", contentUUID), r.URL.Path)
		assert.Equal(t, testTID, r.Header.Get(tidutils.TransactionIDHeader), tidutils.TransactionIDHeader)

		w.Header().Add(originSystemIdHeader, systemID)
		w.Header().Set("Content-Type", contentTypeArticle)
		w.Header().Set("Write-Request-Id", writeRef)
		w.Header().Set("Last-Modified-RFC3339", lastModified)
		w.WriteHeader(status)
		_, err := w.Write(body)
		assert.NoError(t, err)
	}))
}

func mockWriteToGenericRW(t *testing.T, status int, contentUUID, systemID, expectedBody, contentType string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method, "HTTP method")
		assert.Equal(t, fmt.Sprintf("/drafts/content/%s", contentUUID), r.URL.Path)
		assert.Equal(t, testTID, r.Header.Get(tidutils.TransactionIDHeader), tidutils.TransactionIDHeader)
		assert.Equal(t, systemID, r.Header.Get(originSystemIdHeader), originSystemIdHeader)
		assert.Equal(t, contentType, r.Header.Get(contentTypeHeader), contentTypeHeader)

		by, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, expectedBody, string(by), "payload")

		w.WriteHeader(status)
	}))
}

func mockContentValidator(t *testing.T, lastModified string, draftRef string) *mockValidator {
	return &mockValidator{
		expectedDraftRef:     draftRef,
		expectedLastModified: lastModified,
		mock:                 mock.Mock{},
		t:                    t,
	}
}

func (m *mockValidator) Validate(ctx context.Context, contentUUID string, nativeBody io.Reader, contentType string, _ *logger.UPPLogger) (io.ReadCloser, error) {
	args := m.mock.Called(ctx, contentUUID, nativeBody, contentType)
	actualBody := make(map[string]interface{})
	err := json.NewDecoder(nativeBody).Decode(&actualBody)
	assert.NoError(m.t, err)

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

func (m *mockValidator) GTG() error {
	return nil
}

func (m *mockValidator) Endpoint() string {
	return ""
}
