package content

import (
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/husobee/vestigo"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testAPIKey = "testAPIKey"
const testTID = "test_tid"

func TestHappyContentAPI(t *testing.T) {
	cAPIServerMock := newContentAPIServerMock(t, http.StatusOK, aContentBody)
	defer cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey)
	h := NewHandler(cAPI)
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ServeHTTP)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, aContentBody, string(body))
}

func TestContentAPI404(t *testing.T) {
	cAPIServerMock := newContentAPIServerMock(t, http.StatusNotFound, "not found")
	defer cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey)
	h := NewHandler(cAPI)
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ServeHTTP)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, "not found", string(body))
}

func TestContentAPI504(t *testing.T) {
	cAPIServerMock := newContentAPIServerMock(t, http.StatusGatewayTimeout, "gateway time out")
	defer cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey)
	h := NewHandler(cAPI)
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ServeHTTP)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, "{\"message\": \"Service unavailable\"}", string(body))
}

func TestInvalidURL(t *testing.T) {
	cAPI := NewContentAPI(":#", testAPIKey)
	h := NewHandler(cAPI)
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ServeHTTP)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, "parse :: missing protocol scheme\n", string(body))
}

func TestConnectionError(t *testing.T) {
	cAPIServerMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey)
	h := NewHandler(cAPI)
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ServeHTTP)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	_, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.NoError(t, err)
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
