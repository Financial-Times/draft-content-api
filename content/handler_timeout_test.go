package content

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Financial-Times/go-ft-http/fthttp"
	"github.com/Financial-Times/go-logger/v2"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/husobee/vestigo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestReadTimeoutFromDraftContent(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	contentRWTestServer := newDraftContentRWTestServer(t, 300*time.Millisecond, http.StatusOK, contentTypeArticle, originIDcctTest)
	contentAPITestServer := newUppContentAPITestServer(t, 0, http.StatusOK)

	contentRWTestServer.On("EndpointCalled")

	defer contentRWTestServer.server.Close()
	defer contentAPITestServer.server.Close()

	client, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "timing-out-awesome-service"))
	assert.NoError(t, err)

	validatorService := NewSparkDraftContentValidatorService(contentAPITestServer.server.URL, client)
	resolver := NewDraftContentValidatorResolver(cctOnlyResolverConfig(validatorService))
	contentRWService := NewDraftContentRWService(contentRWTestServer.server.URL, resolver, client)
	uppAPI := NewContentAPI(contentAPITestServer.server.URL, testBasicAuthUsername, testBasicAuthPassword, nil, client)

	handler := NewHandler(uppAPI, contentRWService, 150*time.Millisecond, logger.NewUPPLogger("draft-content-api-test", "debug"))

	r := vestigo.NewRouter()

	r.Get("/drafts/content/:uuid", handler.ReadContent)

	server := httptest.NewServer(r)
	defer server.Close()

	resp, err := testRequest(server, contentUUID)
	defer func() {
		err := resp.Body.Close()
		assert.NoError(t, err)
	}()

	assert.NoError(t, err)
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)

	assert.True(t, contentAPITestServer.AssertNotCalled(t, "EndpointCalled"))

	contentRWTestServer.AssertExpectations(t)
}

func TestReadTimeoutFromUPPContent(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	contentRWTestServer := newDraftContentRWTestServer(t, 10*time.Millisecond, http.StatusNotFound, contentTypeArticle, originIDcctTest)
	contentAPITestServer := newUppContentAPITestServer(t, 300*time.Millisecond, http.StatusOK)

	contentRWTestServer.On("EndpointCalled")
	contentAPITestServer.On("EndpointCalled")

	defer contentRWTestServer.server.Close()
	defer contentAPITestServer.server.Close()

	client, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "timing-out-awesome-service"))
	assert.NoError(t, err)

	validatorService := NewSparkDraftContentValidatorService(contentAPITestServer.server.URL, client)
	resolver := NewDraftContentValidatorResolver(cctOnlyResolverConfig(validatorService))

	contentRWService := NewDraftContentRWService(contentRWTestServer.server.URL, resolver, client)
	uppAPI := NewContentAPI(contentAPITestServer.server.URL, testBasicAuthUsername, testBasicAuthPassword, nil, client)

	handler := NewHandler(uppAPI, contentRWService, 150*time.Millisecond, logger.NewUPPLogger("draft-content-api-test", "debug"))

	r := vestigo.NewRouter()

	r.Get("/drafts/content/:uuid", handler.ReadContent)

	server := httptest.NewServer(r)
	defer server.Close()

	resp, err := testRequest(server, contentUUID)
	defer func() {
		err := resp.Body.Close()
		assert.NoError(t, err)
	}()

	assert.NoError(t, err)
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
	mock.AssertExpectationsForObjects(t, contentRWTestServer, contentAPITestServer)
}

func testRequest(server *httptest.Server, contentUUID string) (*http.Response, error) {
	request, _ := http.NewRequest("GET", server.URL+"/drafts/content/"+contentUUID, nil)
	resp, err := http.DefaultClient.Do(request)
	return resp, err
}
func TestNativeWriteTimeout(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	contentRWTestServer := newDraftContentRWTestServer(t, 300*time.Millisecond, http.StatusOK, contentTypeArticle, originIDcctTest)
	contentAPITestServer := newUppContentAPITestServer(t, 0*time.Millisecond, http.StatusOK)
	AllowedOriginSystemIDValues = map[string]struct{}{
		originIDcctTest: {},
	}
	AllowedContentTypes = map[string]struct{}{
		contentTypeArticle: {},
	}

	contentRWTestServer.On("EndpointCalled")

	defer contentRWTestServer.server.Close()
	defer contentAPITestServer.server.Close()

	client, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "timing-out-awesome-service"))
	assert.NoError(t, err)

	validatorService := NewSparkDraftContentValidatorService(contentAPITestServer.server.URL, client)
	resolver := NewDraftContentValidatorResolver(cctOnlyResolverConfig(validatorService))

	contentRWService := NewDraftContentRWService(contentRWTestServer.server.URL, resolver, client)
	uppAPI := NewContentAPI(contentAPITestServer.server.URL, testBasicAuthUsername, testBasicAuthPassword, nil, client)

	handler := NewHandler(uppAPI, contentRWService, 150*time.Millisecond, logger.NewUPPLogger("draft-content-api-test", "debug"))

	r := vestigo.NewRouter()

	r.Put("/drafts/nativecontent/:uuid", handler.WriteNativeContent)

	server := httptest.NewServer(r)
	defer server.Close()

	request, _ := http.NewRequest(http.MethodPut, server.URL+"/drafts/nativecontent/"+contentUUID, nil)
	request.Header.Set(tidutils.TransactionIDHeader, testTID)
	request.Header.Set(originSystemIdHeader, originIDcctTest)
	request.Header.Set(contentTypeHeader, contentTypeArticle)

	resp, err := client.Do(request)
	defer func() {
		err := resp.Body.Close()
		assert.NoError(t, err)
	}()

	assert.NoError(t, err)
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
	contentRWTestServer.AssertExpectations(t)
}

func newDraftContentRWTestServer(t *testing.T, inducedDelay time.Duration, responseStatus int, contentType string, originID string) *mockServer {
	m := &mockServer{}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if inducedDelay > 0 {
			m.EndpointCalled()
			time.Sleep(inducedDelay)
		}

		switch r.Method {
		case http.MethodPut:
			w.WriteHeader(responseStatus)
			return
		case http.MethodGet:
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("X-Origin-System-Id", originID)
			w.WriteHeader(responseStatus)
			_, err := w.Write([]byte(fromUppContent))
			assert.NoError(t, err)
			return
		}

	}))

	return m
}

func newUppContentAPITestServer(t *testing.T, inducedDelay time.Duration, responseStatus int) *mockServer {
	m := &mockServer{}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if inducedDelay > 0 {
			m.EndpointCalled()
			time.Sleep(inducedDelay)
		}
		w.Header().Set("Content-Type", contentTypeArticle)
		w.WriteHeader(responseStatus)
		_, err := w.Write([]byte(fromUppContent))
		assert.NoError(t, err)
	}))

	return m
}

type mockServer struct {
	mock.Mock
	server *httptest.Server
}

func (m *mockServer) EndpointCalled() {
	m.Called()
}
