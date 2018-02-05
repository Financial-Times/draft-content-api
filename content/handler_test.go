package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/husobee/vestigo"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const testAPIKey = "testAPIKey"
const testTID = "test_tid"

type mockDraftContentRW struct {
	mock.Mock
}

func TestHappyRead(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, contentUUID).Return(ioutil.NopCloser(strings.NewReader(aContentBody)), nil)

	h := NewHandler(nil, rw)
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", fmt.Sprintf("http://api.ft.com/drafts/content/%s", contentUUID), nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, aContentBody, string(body))
	rw.AssertExpectations(t)
}

func TestReadBackOffWhenNoDraftFoundToContentAPI(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, contentUUID).Return(nil, ErrDraftNotFound)

	cAPIServerMock := newContentAPIServerMock(t, http.StatusOK, aContentBody)
	defer cAPIServerMock.Close()
	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey)

	h := NewHandler(cAPI, rw)
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", fmt.Sprintf("http://api.ft.com/drafts/content/%s", contentUUID), nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, aContentBody, string(body))
	rw.AssertExpectations(t)
}

func TestReadNoBackOffForOtherErrors(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, contentUUID).Return(nil, errors.New("this should never happen"))

	h := NewHandler(nil, rw)
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", fmt.Sprintf("http://api.ft.com/drafts/content/%s", contentUUID), nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, "{\"message\": \"Service unavailable\"}", string(body))
	rw.AssertExpectations(t)
}

func TestReadNotFoundAnywhere(t *testing.T) {
	cAPIServerMock := newContentAPIServerMock(t, http.StatusNotFound, "not found")
	defer cAPIServerMock.Close()

	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrDraftNotFound)

	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey)
	h := NewHandler(cAPI, rw)

	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, "not found", string(body))
	rw.AssertExpectations(t)
}

func TestReadContentAPI504(t *testing.T) {
	cAPIServerMock := newContentAPIServerMock(t, http.StatusGatewayTimeout, "gateway time out")
	defer cAPIServerMock.Close()

	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrDraftNotFound)

	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey)
	h := NewHandler(cAPI, rw)
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, "{\"message\": \"Service unavailable\"}", string(body))
	rw.AssertExpectations(t)
}

func TestReadInvalidURL(t *testing.T) {
	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrDraftNotFound)
	cAPI := NewContentAPI(":#", testAPIKey)
	h := NewHandler(cAPI, rw)
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, "parse :: missing protocol scheme\n", string(body))
	rw.AssertExpectations(t)
}

func TestReadConnectionError(t *testing.T) {
	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrDraftNotFound)
	cAPIServerMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey)
	h := NewHandler(cAPI, rw)
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	_, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.NoError(t, err)
	rw.AssertExpectations(t)
}

func TestWriteNativeContent(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	draftBody := "{\"foo\":\"bar\"}"
	headers := map[string]string{
		tidutils.TransactionIDHeader: testTID,
		originSystemIdHeader:         "methode-web-pub",
	}

	rw := mockDraftContentRW{}
	/* mock.AnythingOfType(...) doesn't work for interfaces: https://github.com/stretchr/testify/issues/519 */
	rw.On("Write", mock.Anything, contentUUID, &draftBody, headers).Return(nil)

	h := NewHandler(nil, &rw)
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, "methode-web-pub")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	_, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, err)
	rw.AssertExpectations(t)
}

func TestWriteNativeContentInvalidUUID(t *testing.T) {
	draftBody := "{\"foo\":\"bar\"}"

	h := NewHandler(nil, nil)
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", "http://api.ft.com/drafts/nativecontent/foo", strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, "methode-web-pub")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()

	response := make(map[string]string)
	json.NewDecoder(resp.Body).Decode(&response)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, response["message"], "Invalid content UUID", "error message")
}

func TestWriteNativeContentWithoutOriginSystemId(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	draftBody := "{\"foo\":\"bar\"}"

	h := NewHandler(nil, nil /*&rw*/)
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()

	response := make(map[string]string)
	json.NewDecoder(resp.Body).Decode(&response)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, response["message"], "Invalid origin system id", "error message")
}

func TestWriteNativeContentInvalidOriginSystemId(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	draftBody := "{\"foo\":\"bar\"}"

	h := NewHandler(nil, nil)
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, "wordpress")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()

	response := make(map[string]string)
	json.NewDecoder(resp.Body).Decode(&response)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, response["message"], "Invalid origin system id", "error message")
}

func TestWriteNativeContentWriteError(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	draftBody := "{\"foo\":\"bar\"}"

	rw := mockDraftContentRW{}
	rw.On("Write", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test error from writer"))

	h := NewHandler(nil, &rw)
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, "methode-web-pub")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()

	response := make(map[string]string)
	json.NewDecoder(resp.Body).Decode(&response)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, response["message"], "Error in writing draft content", "error message")
	rw.AssertExpectations(t)
}

func newContentAPIServerMock(t *testing.T, status int, body string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiKey := r.Header.Get(apiKeyHeader); apiKey != testAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		assert.Equal(t, testTID, r.Header.Get(tidutils.TransactionIDHeader))
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	return ts
}

func (m *mockDraftContentRW) Read(ctx context.Context, contentUUID string) (io.ReadCloser,error) {
	args := m.Called(ctx, contentUUID)
	var body io.ReadCloser
	o := args.Get(0)
	if o != nil {
		body = o.(io.ReadCloser)
	}
	return body, args.Error(1)
}

func (m *mockDraftContentRW) Write(ctx context.Context, contentUUID string, content *string, headers map[string]string) error {
	args := m.Called(ctx, contentUUID, content, headers)
	return args.Error(0)
}

func (m *mockDraftContentRW) GTG() error {
	return nil
}

func (m *mockDraftContentRW) Endpoint() string {
	return ""
}

const aContentBody = "{" +
	"\"id\": \"http://www.ft.com/thing/83a201c6-60cd-11e7-91a7-502f7ee26895\"," +
	"\"type\": \"http://www.ft.com/ontology/content/Graphic\"," +
	"\"title\": \"Estimated range of the North Korean missile\"," +
	"\"alternativeTitles\": {}," +
	"\"alternativeStandfirsts\": {}," +
	"\"description\": \"\"," +
	"\"firstPublishedDate\": \"2017-07-04T18:14:00.000Z\"," +
	"\"publishedDate\": \"2017-07-04T18:14:00.000Z\"," +
	"\"requestUrl\": \"http://api.ft.com/content/83a201c6-60cd-11e7-91a7-502f7ee26895\"," +
	"\"binaryUrl\": \"http://com.ft.imagepublish.prod.s3.amazonaws.com/83a201c6-60cd-11e7-91a7-502f7ee26895\"," +
	"\"pixelWidth\": 600," +
	"\"pixelHeight\": 546," +
	"\"alternativeImages\": {}," +
	"\"canBeDistributed\": \"verify\"," +
	"\"canBeSyndicated\": \"verify\"" +
	"}"
