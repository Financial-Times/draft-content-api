package content

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Financial-Times/go-ft-http/fthttp"
	"github.com/Financial-Times/go-logger/v2"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/google/uuid"
	"github.com/husobee/vestigo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	originIDcctTest       = "cct"
	contentTypeArticle    = "application/vnd.ft-upp-article+json"
	testBasicAuthUsername = "testUsername"
	testBasicAuthPassword = "testPassword"
	testTID               = "test_tid"
	testTimeout           = 8 * time.Second
)

type mockDraftContentRW struct {
	mock mock.Mock
}

func TestHappyRead(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	rw := &mockDraftContentRW{}
	rw.mock.On("Read", mock.Anything, contentUUID).Return(io.NopCloser(strings.NewReader(fromUppContent)), nil)

	h := NewHandler(nil, rw, testTimeout, logger.NewUPPLogger("test logger", "debug"))
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", fmt.Sprintf("http://api.ft.com/drafts/content/%s", contentUUID), nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	defer func() {
		err := resp.Body.Close()
		assert.NoError(t, err)
	}()
	body, err := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, fromUppContent, string(body))
	rw.mock.AssertExpectations(t)
}

func TestReadBackOffWhenNoDraftFoundToContentAPI(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"
	mainImageUUID := "fba9884e-0756-11e8-0074-38e932af9738"

	rw := &mockDraftContentRW{}
	rw.mock.On("Read", mock.Anything, contentUUID).Return(nil, ErrDraftNotFound)

	cAPIServerMock := newContentAPIServerMock(t, http.StatusOK, fromUppContent)
	defer cAPIServerMock.Close()
	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	cAPI := NewContentAPI(cAPIServerMock.URL, testBasicAuthUsername, testBasicAuthPassword, "", testClient)

	h := NewHandler(cAPI, rw, testTimeout, logger.NewUPPLogger("test logger", "debug"))
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", fmt.Sprintf("http://api.ft.com/drafts/content/%s", contentUUID), nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	defer func() {
		err := resp.Body.Close()
		assert.NoError(t, err)
	}()
	body, err := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, err)

	var actual map[string]interface{}
	err = json.Unmarshal(body, &actual)
	assert.NoError(t, err)

	assert.Equal(t, contentUUID, actual["uuid"])
	assert.Equal(t,
		[]interface{}{map[string]interface{}{"id": "http://api.ft.com/things/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"}},
		actual["brands"])

	actualBody, present := actual["body"]

	assert.True(t, present)
	assert.NotEmpty(t, actualBody)

	assert.Equal(t, "Article", actual["type"])
	assert.Equal(t, mainImageUUID, actual["mainImage"])
	rw.mock.AssertExpectations(t)
}

func TestReadNoBackOffForOtherErrors(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	rw := &mockDraftContentRW{}
	rw.mock.On("Read", mock.Anything, contentUUID).Return(nil, errors.New("this should never happen"))

	h := NewHandler(nil, rw, testTimeout, logger.NewUPPLogger("test logger", "debug"))
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", fmt.Sprintf("http://api.ft.com/drafts/content/%s", contentUUID), nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	defer func() {
		err := resp.Body.Close()
		assert.NoError(t, err)
	}()
	body, err := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, "{\"message\": \"Error reading draft content\"}", string(body))
	rw.mock.AssertExpectations(t)
}

func TestReadNotFoundAnywhere(t *testing.T) {
	cAPIServerMock := newContentAPIServerMock(t, http.StatusNotFound, "not found")
	defer cAPIServerMock.Close()

	rw := &mockDraftContentRW{}
	rw.mock.On("Read", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrDraftNotFound)

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	cAPI := NewContentAPI(cAPIServerMock.URL, testBasicAuthUsername, testBasicAuthPassword, "", testClient)
	h := NewHandler(cAPI, rw, testTimeout, logger.NewUPPLogger("test logger", "debug"))

	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, "{\"message\": \"Draft not found\"}", string(body))
	rw.mock.AssertExpectations(t)
}

func TestReadContentAPI504(t *testing.T) {
	cAPIServerMock := newContentAPIServerMock(t, http.StatusGatewayTimeout, "gateway time out")
	defer cAPIServerMock.Close()

	rw := &mockDraftContentRW{}
	rw.mock.On("Read", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrDraftNotFound)

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	cAPI := NewContentAPI(cAPIServerMock.URL, testBasicAuthUsername, testBasicAuthPassword, "", testClient)
	h := NewHandler(cAPI, rw, testTimeout, logger.NewUPPLogger("test logger", "debug"))
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, "{\"message\": \"Draft content request processing has timed out\"}", string(body))
	rw.mock.AssertExpectations(t)
}

func TestReadInvalidURL(t *testing.T) {
	rw := &mockDraftContentRW{}
	rw.mock.On("Read", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrDraftNotFound)
	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	cAPI := NewContentAPI(":#", testBasicAuthUsername, testBasicAuthPassword, "", testClient)
	h := NewHandler(cAPI, rw, testTimeout, logger.NewUPPLogger("test logger", "debug"))
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.NoError(t, err)
	assert.Contains(t, string(body), "missing protocol scheme")
	rw.mock.AssertExpectations(t)
}

func TestReadConnectionError(t *testing.T) {
	rw := &mockDraftContentRW{}
	rw.mock.On("Read", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrDraftNotFound)
	cAPIServerMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	cAPIServerMock.Close()

	testClient, err := fthttp.NewClient(fthttp.WithSysInfo("PAC", "awesome-service"))
	assert.NoError(t, err)
	cAPI := NewContentAPI(cAPIServerMock.URL, testBasicAuthUsername, testBasicAuthPassword, "", testClient)
	h := NewHandler(cAPI, rw, testTimeout, logger.NewUPPLogger("test logger", "debug"))
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", "http://api.ft.com/drafts/content/83a201c6-60cd-11e7-91a7-502f7ee26895", nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	_, err = io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.NoError(t, err)
	rw.mock.AssertExpectations(t)
}

func TestWriteCCTNativeContent(t *testing.T) {
	contentUUID := uuid.New().String()
	draftBody := "{\"foo\":\"bar\"}"
	headers := map[string]string{
		tidutils.TransactionIDHeader: testTID,
		originSystemIdHeader:         originIDcctTest,
		contentTypeHeader:            contentTypeArticle,
	}

	AllowedOriginSystemIDValues = map[string]struct{}{
		originIDcctTest: {},
	}

	AllowedContentTypes = map[string]struct{}{
		contentTypeArticle: {},
	}

	rw := mockDraftContentRW{}
	/* mock.AnythingOfType(...) doesn't work for interfaces: https://github.com/stretchr/testify/issues/519 */
	rw.mock.On("Write", mock.Anything, contentUUID, &draftBody, headers).Return(nil)

	h := NewHandler(nil, &rw, testTimeout, logger.NewUPPLogger("test logger", "debug"))
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, originIDcctTest)
	req.Header.Set(contentTypeHeader, contentTypeArticle)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	_, err := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, err)
	rw.mock.AssertExpectations(t)
}

func TestWriteSparkNativeContent(t *testing.T) {
	contentUUID := uuid.New().String()
	draftBody := "{\"foo\":\"bar\"}"
	headers := map[string]string{
		tidutils.TransactionIDHeader: testTID,
		originSystemIdHeader:         originIDcctTest,
		contentTypeHeader:            contentTypeArticle + "; version=1.0; charset=utf-8",
	}

	AllowedOriginSystemIDValues = map[string]struct{}{
		originIDcctTest: {},
	}

	AllowedContentTypes = map[string]struct{}{
		contentTypeArticle: {},
	}

	rw := mockDraftContentRW{}
	/* mock.AnythingOfType(...) doesn't work for interfaces: https://github.com/stretchr/testify/issues/519 */
	rw.mock.On("Write", mock.Anything, contentUUID, &draftBody, headers).Return(nil)

	h := NewHandler(nil, &rw, testTimeout, logger.NewUPPLogger("test logger", "debug"))
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, originIDcctTest)
	req.Header.Set(contentTypeHeader, "application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	_, err := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, err)
	rw.mock.AssertExpectations(t)
}

func TestWriteNativeContentInvalidUUID(t *testing.T) {
	draftBody := "{\"foo\":\"bar\"}"

	h := NewHandler(nil, nil, testTimeout, logger.NewUPPLogger("test logger", "debug"))
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", "http://api.ft.com/drafts/nativecontent/foo", strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, originIDcctTest)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()

	response := make(map[string]string)
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, response["message"], "Invalid content UUID", "error message")
}

func TestWriteNativeContentWithoutOriginSystemId(t *testing.T) {
	contentUUID := uuid.New().String()
	draftBody := "{\"foo\":\"bar\"}"

	h := NewHandler(nil, nil /*&rw*/, testTimeout, logger.NewUPPLogger("test logger", "debug"))
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()

	response := make(map[string]string)
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, response["message"], "Invalid origin system id", "error message")
}

func TestWriteNativeContentInvalidOriginSystemId(t *testing.T) {
	contentUUID := uuid.New().String()
	draftBody := "{\"foo\":\"bar\"}"

	h := NewHandler(nil, nil, testTimeout, logger.NewUPPLogger("test logger", "debug"))
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, "wordpress")
	req.Header.Set(contentTypeHeader, contentTypeArticle)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()

	response := make(map[string]string)
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, response["message"], "Invalid origin system id", "error message")
}

func TestWriteNativeContentInvalidContentType(t *testing.T) {
	contentUUID := uuid.New().String()
	draftBody := "{\"foo\":\"bar\"}"

	AllowedOriginSystemIDValues = map[string]struct{}{
		originIDcctTest: {},
	}

	AllowedContentTypes = map[string]struct{}{
		contentTypeArticle: {},
	}

	h := NewHandler(nil, nil, testTimeout, logger.NewUPPLogger("test logger", "debug"))
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, originIDcctTest)
	req.Header.Set(contentTypeHeader, "application/xml")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()

	response := make(map[string]string)
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, response["message"], "Invalid content type", "error message")
}

func TestWriteNativeContentWriteError(t *testing.T) {
	contentUUID := uuid.New().String()
	draftBody := "{\"foo\":\"bar\"}"

	AllowedOriginSystemIDValues = map[string]struct{}{
		originIDcctTest: {},
	}

	AllowedContentTypes = map[string]struct{}{
		contentTypeArticle: {},
	}

	rw := mockDraftContentRW{}
	rw.mock.On("Write", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test error from writer"))

	h := NewHandler(nil, &rw, testTimeout, logger.NewUPPLogger("test logger", "debug"))
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, originIDcctTest)
	req.Header.Set(contentTypeHeader, contentTypeArticle)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()

	response := make(map[string]string)
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, response["message"], "Error in writing draft content", "error message")
	rw.mock.AssertExpectations(t)
}

func newContentAPIServerMock(t *testing.T, status int, body string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if basicAuth := r.Header.Get(authorizationHeader); basicAuth != createBasicAuth(t) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		assert.Equal(t, testTID, r.Header.Get(tidutils.TransactionIDHeader))
		w.WriteHeader(status)
		if _, err := w.Write([]byte(body)); err != nil {
			panic(err)
		}
	}))
	return ts
}

func createBasicAuth(t *testing.T) string {
	t.Helper()
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(strings.Join([]string{testBasicAuthUsername, testBasicAuthPassword}, ":")))
}

func (m *mockDraftContentRW) Read(ctx context.Context, contentUUID string, _ *logger.UPPLogger) (io.ReadCloser, error) {
	args := m.mock.Called(ctx, contentUUID)
	var body io.ReadCloser
	o := args.Get(0)
	if o != nil {
		body = o.(io.ReadCloser)
	}
	return body, args.Error(1)
}

func (m *mockDraftContentRW) Write(ctx context.Context, contentUUID string, content *string, headers map[string]string, _ *logger.UPPLogger) error {
	args := m.mock.Called(ctx, contentUUID, content, headers)
	return args.Error(0)
}

func (m *mockDraftContentRW) GTG() error {
	return nil
}

func (m *mockDraftContentRW) Endpoint() string {
	return ""
}

const fromUppContent = `{
   "id":"http://www.ft.com/thing/83a201c6-60cd-11e7-91a7-502f7ee26895",
   "type":"http://www.ft.com/ontology/content/Article",
   "bodyXML":"<body><ft-content type=\"http://www.ft.com/ontology/content/ImageSet\" url=\"http://api.ft.com/content/fba9884e-0756-11e8-0074-38e932af9738\" data-embedded=\"true\"></ft-content><p>Britain has eliminated the deficit on its day-to-day budget, the target originally set by George Osborne when he imposed austerity on public services in 2010. </p>\n<p>The rapid improvement in the public finances over the past six months means that the former chancellor’s ambition for a <ft-content type=\"http://www.ft.com/ontology/content/Article\" url=\"http://api.ft.com/content/0e76ec54-1702-11e8-9e9c-25c814761640\" title=\"www.ft.com\">surplus on the current budget</ft-content>, which excludes capital investment, has been met, albeit two years later than planned. </p>\n<p>Paul Johnson, director of the Institute for Fiscal Studies, said the deficit reduction was “quite an achievement given how poor economic growth has been. They have stuck at it, but deficit reduction has come at the cost of an unprecedented squeeze in public spending”. </p>\n<p>That squeeze is now showing up in higher waiting times in hospitals for emergency treatment, worse performance measures in prisons, severe cuts in many local authorities and <ft-content type=\"http://www.ft.com/ontology/content/Article\" url=\"http://api.ft.com/content/e9df6f8e-1bb5-11e8-aaca-4574d7dabfb6\">lower satisfaction ratings for GP services</ft-content>.</p>\n<p>Fears of a <ft-content type=\"http://www.ft.com/ontology/content/Article\" url=\"http://api.ft.com/content/768843e8-a839-11e7-93c5-648314d2c72c\" title=\"www.ft.com\">bloodbath</ft-content> in the public finances, which the Treasury harboured as recently as October, have now been replaced by insistence from Philip Hammond’s officials that the Treasury will not respond to the windfall of revenues by easing the squeeze. </p>\n<p>They say there will be <ft-content type=\"http://www.ft.com/ontology/content/Article\" url=\"http://api.ft.com/content/82e0f642-d072-11e7-b781-794ce08b24dc\" title=\"www.ft.com\">“no spending increases, no tax changes</ft-content>” in the chancellor’s spring statement on March 13 despite pressure from Brexiters in the cabinet to raise health spending, as promised by the Vote Leave campaign in the EU referendum. </p>\n<p>Official figures show that in the 12 months ending in January, the current budget showed a £3bn surplus and it moved into the black on an annual rolling basis in November, but this came as a surprise to some senior Treasury officials. </p>\n<ft-content type=\"http://www.ft.com/ontology/content/ImageSet\" url=\"http://api.ft.com/content/b8950876-1cb9-11e8-34ac-d2ee0dae14ff\" data-embedded=\"true\"></ft-content>\n<p>Rupert Harrison, former chief of staff to Mr Osborne and now a portfolio manager at BlackRock, said: “The fact that the UK has a bit of fiscal wriggle room now as it faces uncertainty from Brexit is entirely the result of a huge and consistent focus across government on reducing the deficit over many years.</p>\n<p>“It’s easy to forget quite how big the fiscal crisis that we faced was just a few years ago.” </p>\n<p>The Treasury said: “We are making a success of reducing the deficit, which is down by more than three-quarters since 2010. But our national debt is still too high, and we must get debt falling to improve our economic resilience and reduce the burden on future generations.”</p>\n<p>When the coalition government came into office in 2010, the current budget deficit was £100bn a year, while the total deficit, including net investment, slightly exceeded £150bn at 10 per cent of national income. That is now below 2 per cent, already meeting the fiscal mandate set by Mr Hammond in 2016 to be achieved by 2020-21. </p>\n<p>No British government has run a surplus on the current budget since 2001-2002 and last had a surplus in any 12-month period since the year leading up to July 2002.</p>\n<p>The main reason for the sudden <ft-content type=\"http://www.ft.com/ontology/content/Article\" url=\"http://api.ft.com/content/8fe15508-0024-11e8-9650-9c0ad2d7c5b5\" title=\"www.ft.com\">improvement in the public finances </ft-content>has been that revenues this financial year have greatly exceeded expectations, particularly in the most important month of January when self-assessment bills for income tax and capital gains tax generally fall due. </p>\n<p>The Office for Budget Responsibility has said it will revise up its forecasts for the public finances even if it keeps its economic forecasts unchanged in the spring statement, saying the level of overall borrowing “will undershoot our November forecast by a significant margin”.</p>\n\n\n\n\n</body>",
   "title":"George Osborne austerity target is hit — 2 years late",
   "standfirst":"Improvement in public finances puts day-to-day budget into surplus",
   "byline":"Chris Giles, Economics Editor",
   "firstPublishedDate":"2018-03-01T05:00:27.000Z",
   "publishedDate":"2018-03-01T05:00:27.000Z",
   "requestUrl":"http://api.ft.com/content/3f7db634-1cac-11e8-aaca-4574d7dabfb6",
   "brands":[
      	"http://api.ft.com/things/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"
   ],
   "mainImage":{
      "id":"http://api.ft.com/content/fba9884e-0756-11e8-0074-38e932af9738"
   },
   "standout":{
      "editorsChoice":false,
      "exclusive":false,
      "scoop":false
   },
   "canBeSyndicated":"yes",
   "webUrl":"http://www.ft.com/cms/s/3f7db634-1cac-11e8-aaca-4574d7dabfb6.html"
}
`
