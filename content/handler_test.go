package content

import (
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testAPIKey = "testAPIKey"

var testHeaders = http.Header{
	"Pippo":   []string{"pluto"},
	"Cartman": []string{"Kyle"},
}

func TestHappyContentAPI(t *testing.T) {
	cAPIServerMock := newContentAPIServerMock(t, http.StatusOK, aContentBody)
	defer cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey)
	h := NewHandler(cAPI)
	r := mux.NewRouter()
	r.Handle("/drafts/content/{uuid}", h)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header = testHeaders
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
	r := mux.NewRouter()
	r.Handle("/drafts/content/{uuid}", h)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header = testHeaders
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, "not found", string(body))
}

func TestInvalidURL(t *testing.T) {
	cAPI := NewContentAPI(":#", testAPIKey)
	h := NewHandler(cAPI)
	r := mux.NewRouter()
	r.Handle("/drafts/content/{uuid}", h)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header = testHeaders
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, "parse :: missing protocol scheme\n", string(body))
}

func TestConnectionError(t *testing.T) {
	cAPI := NewContentAPI("http://an-endpoint-that-does-not-exist.com", testAPIKey)
	h := NewHandler(cAPI)
	r := mux.NewRouter()
	r.Handle("/drafts/content/{uuid}", h)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header = testHeaders
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	_, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.NoError(t, err)
}

func newContentAPIServerMock(t *testing.T, status int, body string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range testHeaders {
			assert.Equal(t, v, r.Header[k])
		}

		if values := r.URL.Query(); len(values["apiKey"]) != 1 || values["apiKey"][0] != testAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
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
