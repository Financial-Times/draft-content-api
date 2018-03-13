package content

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Financial-Times/go-ft-http/fthttp"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/husobee/vestigo"
	"github.com/stretchr/testify/assert"
)

func TestReadTimeoutFromDraftContent(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	contentRWTestServer := newDraftContentRWTestServer(300*time.Millisecond, http.StatusOK)
	mapperTestServer := newMethodeArticleMapperTestServer(10*time.Millisecond, http.StatusOK)
	contentAPITestServer := newUppContentAPITestServer(10*time.Millisecond, http.StatusOK)

	defer contentRWTestServer.Close()
	defer mapperTestServer.Close()
	defer contentAPITestServer.Close()

	client := fthttp.NewClientWithDefaultTimeout("PAC", "timing-out-awesome-service")

	mapperService := NewDraftContentMapperService(mapperTestServer.URL, client)
	contentRWService := NewDraftContentRWService(contentRWTestServer.URL, mapperService, client)
	uppApi := NewContentAPI(contentAPITestServer.URL, "awesomely-unique-key", client)

	handler := NewHandler(uppApi, contentRWService, 150*time.Millisecond)

	r := vestigo.NewRouter()

	r.Get("/drafts/content/:uuid", handler.ReadContent)

	server := httptest.NewServer(r)
	defer server.Close()

	resp, err := http.Get(server.URL + "/drafts/content/" + contentUUID)
	defer resp.Body.Close()

	assert.NoError(t, err)
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
}

func TestReadTimeoutFromUPPContent(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	contentRWTestServer := newDraftContentRWTestServer(10*time.Millisecond, http.StatusNotFound)
	mapperTestServer := newMethodeArticleMapperTestServer(10*time.Millisecond, http.StatusOK)
	contentAPITestServer := newUppContentAPITestServer(300*time.Millisecond, http.StatusOK)

	defer contentRWTestServer.Close()
	defer mapperTestServer.Close()
	defer contentAPITestServer.Close()

	client := fthttp.NewClientWithDefaultTimeout("PAC", "timing-out-awesome-service")

	mapperService := NewDraftContentMapperService(mapperTestServer.URL, client)
	contentRWService := NewDraftContentRWService(contentRWTestServer.URL, mapperService, client)
	uppApi := NewContentAPI(contentAPITestServer.URL, "awesomely-unique-key", client)

	handler := NewHandler(uppApi, contentRWService, 150*time.Millisecond)

	r := vestigo.NewRouter()

	r.Get("/drafts/content/:uuid", handler.ReadContent)

	server := httptest.NewServer(r)
	defer server.Close()

	resp, err := http.Get(server.URL + "/drafts/content/" + contentUUID)
	defer resp.Body.Close()

	assert.NoError(t, err)
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
}
func TestReadTimeoutFromMethodeArticleMapper(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	contentRWTestServer := newDraftContentRWTestServer(10*time.Millisecond, http.StatusOK)
	mapperTestServer := newMethodeArticleMapperTestServer(300*time.Millisecond, http.StatusOK)
	contentAPITestServer := newUppContentAPITestServer(10*time.Millisecond, http.StatusOK)

	defer contentRWTestServer.Close()
	defer mapperTestServer.Close()
	defer contentAPITestServer.Close()

	client := fthttp.NewClientWithDefaultTimeout("PAC", "timing-out-awesome-service")

	mapperService := NewDraftContentMapperService(mapperTestServer.URL, client)
	contentRWService := NewDraftContentRWService(contentRWTestServer.URL, mapperService, client)
	uppApi := NewContentAPI(contentAPITestServer.URL, "awesomely-unique-key", client)

	handler := NewHandler(uppApi, contentRWService, 150*time.Millisecond)

	r := vestigo.NewRouter()

	r.Get("/drafts/content/:uuid", handler.ReadContent)

	server := httptest.NewServer(r)
	defer server.Close()

	resp, err := http.Get(server.URL + "/drafts/content/" + contentUUID)
	defer resp.Body.Close()

	assert.NoError(t, err)
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
}
func TestNativeWriteTimeout(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	contentRWTestServer := newDraftContentRWTestServer(300*time.Millisecond, http.StatusOK)
	mapperTestServer := newMethodeArticleMapperTestServer(0*time.Millisecond, http.StatusOK)
	contentAPITestServer := newUppContentAPITestServer(0*time.Millisecond, http.StatusOK)

	defer contentRWTestServer.Close()
	defer mapperTestServer.Close()
	defer contentAPITestServer.Close()

	client := fthttp.NewClientWithDefaultTimeout("PAC", "timing-out-awesome-service")

	mapperService := NewDraftContentMapperService(mapperTestServer.URL, client)
	contentRWService := NewDraftContentRWService(contentRWTestServer.URL, mapperService, client)
	uppApi := NewContentAPI(contentAPITestServer.URL, "awesomely-unique-key", client)

	handler := NewHandler(uppApi, contentRWService, 150*time.Millisecond)

	r := vestigo.NewRouter()

	r.Put("/drafts/nativecontent/:uuid", handler.WriteNativeContent)

	server := httptest.NewServer(r)
	defer server.Close()

	request, _ := http.NewRequest(http.MethodPut, server.URL+"/drafts/nativecontent/"+contentUUID, nil)
	request.Header.Set(tidutils.TransactionIDHeader, testTID)
	request.Header.Set(originSystemIdHeader, "methode-web-pub")

	resp, err := client.Do(request)
	defer resp.Body.Close()

	assert.NoError(t, err)
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
}

func newDraftContentRWTestServer(inducedDelay time.Duration, responseStatus int) *httptest.Server {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if inducedDelay > 0 {
			time.Sleep(inducedDelay)
		}

		switch r.Method {
		case http.MethodPut:
			w.WriteHeader(responseStatus)
			return
		case http.MethodGet:
			w.WriteHeader(responseStatus)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(fromMaMContent))
			return
		}

	}))

	return server
}

func newMethodeArticleMapperTestServer(inducedDelay time.Duration, responseStatus int) *httptest.Server {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if inducedDelay > 0 {
			time.Sleep(inducedDelay)
		}

		w.WriteHeader(responseStatus)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fromMaMContent))
	}))

	return server
}

func newUppContentAPITestServer(inducedDelay time.Duration, responseStatus int) *httptest.Server {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if inducedDelay > 0 {
			time.Sleep(inducedDelay)
		}
		w.WriteHeader(responseStatus)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fromUppContent))
	}))

	return server
}
