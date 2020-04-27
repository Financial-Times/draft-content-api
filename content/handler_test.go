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
	"time"

	"github.com/Financial-Times/go-ft-http/fthttp"
	tidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/husobee/vestigo"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const testAPIKey = "testAPIKey"
const testTID = "test_tid"
const testTimeout = 8 * time.Second

type mockDraftContentRW struct {
	mock.Mock
}

func TestHappyRead(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, contentUUID).Return(ioutil.NopCloser(strings.NewReader(fromMaMContent)), nil)

	h := NewHandler(nil, rw, testTimeout)
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
	assert.Equal(t, fromMaMContent, string(body))
	rw.AssertExpectations(t)
}

func TestReadBackOffWhenNoDraftFoundToContentAPI(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, contentUUID).Return(nil, ErrDraftNotFound)

	cAPIServerMock := newContentAPIServerMock(t, http.StatusOK, fromUppContent)
	defer cAPIServerMock.Close()
	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))

	h := NewHandler(cAPI, rw, testTimeout)
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

	var expected map[string]interface{}
	var actual map[string]interface{}

	json.Unmarshal([]byte(fromMaMContent), &expected)
	json.Unmarshal(body, &actual)

	// both should have the same uuid, brands, body and type fields
	// since proper transformation should already been applied.

	assert.Equal(t, expected["uuid"], actual["uuid"])
	assert.Equal(t, expected["brands"], actual["brands"])

	actualBody, present := actual["body"]

	assert.True(t, present)
	assert.NotEmpty(t, actualBody)

	assert.Equal(t, expected["type"], actual["type"])

	rw.AssertExpectations(t)
}

func TestReadNoBackOffForOtherErrors(t *testing.T) {
	contentUUID := "83a201c6-60cd-11e7-91a7-502f7ee26895"

	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, contentUUID).Return(nil, errors.New("this should never happen"))

	h := NewHandler(nil, rw, testTimeout)
	r := vestigo.NewRouter()
	r.Get("/drafts/content/:uuid", h.ReadContent)

	req := httptest.NewRequest("GET", fmt.Sprintf("http://api.ft.com/drafts/content/%s", contentUUID), nil)
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, "{\"message\": \"Error reading draft content\"}", string(body))
	rw.AssertExpectations(t)
}

func TestReadNotFoundAnywhere(t *testing.T) {
	cAPIServerMock := newContentAPIServerMock(t, http.StatusNotFound, "not found")
	defer cAPIServerMock.Close()

	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrDraftNotFound)

	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	h := NewHandler(cAPI, rw, testTimeout)

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
	assert.Equal(t, "{\"message\": \"Draft not found\"}", string(body))
	rw.AssertExpectations(t)
}

func TestReadContentAPI504(t *testing.T) {
	cAPIServerMock := newContentAPIServerMock(t, http.StatusGatewayTimeout, "gateway time out")
	defer cAPIServerMock.Close()

	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrDraftNotFound)

	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	h := NewHandler(cAPI, rw, testTimeout)
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
	assert.Equal(t, "{\"message\": \"Draft content request processing has timed out\"}", string(body))
	rw.AssertExpectations(t)
}

func TestReadInvalidURL(t *testing.T) {
	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrDraftNotFound)
	cAPI := NewContentAPI(":#", testAPIKey, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	h := NewHandler(cAPI, rw, testTimeout)
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
	assert.Contains(t, string(body), "missing protocol scheme")
	rw.AssertExpectations(t)
}

func TestReadConnectionError(t *testing.T) {
	rw := &mockDraftContentRW{}
	rw.On("Read", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrDraftNotFound)
	cAPIServerMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	cAPIServerMock.Close()

	cAPI := NewContentAPI(cAPIServerMock.URL, testAPIKey, fthttp.NewClientWithDefaultTimeout("PAC", "awesome-service"))
	h := NewHandler(cAPI, rw, testTimeout)
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

func TestWriteMethodeNativeContent(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	draftBody := "{\"foo\":\"bar\"}"
	headers := map[string]string{
		tidutils.TransactionIDHeader: testTID,
		originSystemIdHeader:         "methode-web-pub",
		contentTypeHeader:            "application/json",
	}

	rw := mockDraftContentRW{}
	/* mock.AnythingOfType(...) doesn't work for interfaces: https://github.com/stretchr/testify/issues/519 */
	rw.On("Write", mock.Anything, contentUUID, &draftBody, headers).Return(nil)

	h := NewHandler(nil, &rw, testTimeout)
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, "methode-web-pub")
	req.Header.Set(contentTypeHeader, "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()
	_, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, err)
	rw.AssertExpectations(t)
}

func TestWriteSparkNativeContent(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	draftBody := "{\"foo\":\"bar\"}"
	headers := map[string]string{
		tidutils.TransactionIDHeader: testTID,
		originSystemIdHeader:         "cct",
		contentTypeHeader:            "application/vnd.ft-upp-article+json; version=1.0; charset=utf-8",
	}

	rw := mockDraftContentRW{}
	/* mock.AnythingOfType(...) doesn't work for interfaces: https://github.com/stretchr/testify/issues/519 */
	rw.On("Write", mock.Anything, contentUUID, &draftBody, headers).Return(nil)

	h := NewHandler(nil, &rw, testTimeout)
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, "cct")
	req.Header.Set(contentTypeHeader, "application/vnd.ft-upp-article+json; version=1.0; charset=utf-8")
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

	h := NewHandler(nil, nil, testTimeout)
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

	h := NewHandler(nil, nil /*&rw*/, testTimeout)
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

	h := NewHandler(nil, nil, testTimeout)
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, "wordpress")
	req.Header.Set(contentTypeHeader, "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()

	response := make(map[string]string)
	json.NewDecoder(resp.Body).Decode(&response)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, response["message"], "Invalid origin system id", "error message")
}

func TestWriteNativeContentInvalidContentType(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	draftBody := "{\"foo\":\"bar\"}"

	h := NewHandler(nil, nil, testTimeout)
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, "cct")
	req.Header.Set(contentTypeHeader, "application/xml")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	resp := w.Result()

	response := make(map[string]string)
	json.NewDecoder(resp.Body).Decode(&response)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, response["message"], "Invalid content type", "error message")
}

func TestWriteNativeContentWriteError(t *testing.T) {
	contentUUID := uuid.NewV4().String()
	draftBody := "{\"foo\":\"bar\"}"

	rw := mockDraftContentRW{}
	rw.On("Write", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test error from writer"))

	h := NewHandler(nil, &rw, testTimeout)
	r := vestigo.NewRouter()
	r.Put("/drafts/nativecontent/:uuid", h.WriteNativeContent)

	req := httptest.NewRequest("PUT", fmt.Sprintf("http://api.ft.com/drafts/nativecontent/%s", contentUUID), strings.NewReader(draftBody))
	req.Header.Set(tidutils.TransactionIDHeader, testTID)
	req.Header.Set(originSystemIdHeader, "methode-web-pub")
	req.Header.Set(contentTypeHeader, "application/json")
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

func (m *mockDraftContentRW) Read(ctx context.Context, contentUUID string) (io.ReadCloser, error) {
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

const fromMaMContent = `{
   "uuid":"3f7db634-1cac-11e8-aaca-4574d7dabfb6",
   "title":"George Osborne austerity target is hit — 2 years late",
   "alternativeTitles":{
      "promotionalTitle":"Osborne austerity target hit — 2 years late",
      "contentPackageTitle":null
   },
   "type":"Article",
   "byline":"Chris Giles, Economics Editor",
   "brands":[
      {
         "id":"http://api.ft.com/things/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"
      }
   ],
   "identifiers":[
      {
         "authority":"http://api.ft.com/system/FTCOM-METHODE",
         "identifierValue":"3f7db634-1cac-11e8-aaca-4574d7dabfb6"
      }
   ],
   "publishedDate":"2018-03-01T05:00:27.000Z",
   "standfirst":"Improvement in public finances puts day-to-day budget into surplus",
   "body":"<body><content data-embedded=\"true\" id=\"fba9884e-0756-11e8-0074-38e932af9738\" type=\"http://www.ft.com/ontology/content/ImageSet\"></content><p>Britain has eliminated the deficit on its day-to-day budget, the target originally set by George Osborne when he imposed austerity on public services in 2010. </p>\n<p>The rapid improvement in the public finances over the past six months means that the former chancellor’s ambition for a <content id=\"0e76ec54-1702-11e8-9e9c-25c814761640\" title=\"www.ft.com\" type=\"http://www.ft.com/ontology/content/Article\">surplus on the current budget</content>, which excludes capital investment, has been met, albeit two years later than planned. </p>\n<p>Paul Johnson, director of the Institute for Fiscal Studies, said the deficit reduction was “quite an achievement given how poor economic growth has been. They have stuck at it, but deficit reduction has come at the cost of an unprecedented squeeze in public spending”. </p>\n<p>That squeeze is now showing up in higher waiting times in hospitals for emergency treatment, worse performance measures in prisons, severe cuts in many local authorities and <content id=\"e9df6f8e-1bb5-11e8-aaca-4574d7dabfb6\" type=\"http://www.ft.com/ontology/content/Article\">lower satisfaction ratings for GP services</content>.</p>\n<p>Fears of a <content id=\"768843e8-a839-11e7-93c5-648314d2c72c\" title=\"www.ft.com\" type=\"http://www.ft.com/ontology/content/Article\">bloodbath</content> in the public finances, which the Treasury harboured as recently as October, have now been replaced by insistence from Philip Hammond’s officials that the Treasury will not respond to the windfall of revenues by easing the squeeze. </p>\n<p>They say there will be <content id=\"82e0f642-d072-11e7-b781-794ce08b24dc\" title=\"www.ft.com\" type=\"http://www.ft.com/ontology/content/Article\">“no spending increases, no tax changes</content>” in the chancellor’s spring statement on March 13 despite pressure from Brexiters in the cabinet to raise health spending, as promised by the Vote Leave campaign in the EU referendum. </p>\n<p>Official figures show that in the 12 months ending in January, the current budget showed a £3bn surplus and it moved into the black on an annual rolling basis in November, but this came as a surprise to some senior Treasury officials. </p>\n<content data-embedded=\"true\" id=\"b8950876-1cb9-11e8-34ac-d2ee0dae14ff\" type=\"http://www.ft.com/ontology/content/ImageSet\"></content>\n<p>Rupert Harrison, former chief of staff to Mr Osborne and now a portfolio manager at BlackRock, said: “The fact that the UK has a bit of fiscal wriggle room now as it faces uncertainty from Brexit is entirely the result of a huge and consistent focus across government on reducing the deficit over many years.</p>\n<p>“It’s easy to forget quite how big the fiscal crisis that we faced was just a few years ago.” </p>\n<p>The Treasury said: “We are making a success of reducing the deficit, which is down by more than three-quarters since 2010. But our national debt is still too high, and we must get debt falling to improve our economic resilience and reduce the burden on future generations.”</p>\n<p>When the coalition government came into office in 2010, the current budget deficit was £100bn a year, while the total deficit, including net investment, slightly exceeded £150bn at 10 per cent of national income. That is now below 2 per cent, already meeting the fiscal mandate set by Mr Hammond in 2016 to be achieved by 2020-21. </p>\n<p>No British government has run a surplus on the current budget since 2001-2002 and last had a surplus in any 12-month period since the year leading up to July 2002.</p>\n<p>The main reason for the sudden <content id=\"8fe15508-0024-11e8-9650-9c0ad2d7c5b5\" title=\"www.ft.com\" type=\"http://www.ft.com/ontology/content/Article\">improvement in the public finances </content>has been that revenues this financial year have greatly exceeded expectations, particularly in the most important month of January when self-assessment bills for income tax and capital gains tax generally fall due. </p>\n<p>The Office for Budget Responsibility has said it will revise up its forecasts for the public finances even if it keeps its economic forecasts unchanged in the spring statement, saying the level of overall borrowing “will undershoot our November forecast by a significant margin”.</p>\n\n\n\n\n</body>",
   "description":null,
   "mediaType":null,
   "pixelWidth":null,
   "pixelHeight":null,
   "internalBinaryUrl":null,
   "externalBinaryUrl":null,
   "members":null,
   "mainImage":"fba9884e-0756-11e8-0074-38e932af9738",
   "standout":{
      "editorsChoice":false,
      "exclusive":false,
      "scoop":false
   },
   "comments":{
      "enabled":true
   },
   "copyright":null,
   "webUrl":null,
   "lastModified":"2018-03-01T11:42:33.020Z",
   "canBeSyndicated":"yes",
   "firstPublishedDate":"2018-03-01T05:00:27.000Z",
   "accessLevel":"subscribed",
   "canBeDistributed":"yes",
   "rightsGroup":null,
   "masterSource":null,
   "alternativeStandfirsts":{
      "promotionalStandfirst":null
   },
   "publishReference":"tid_nryyinjl3c"
}
`

const fromUppContent = `{
   "id":"http://www.ft.com/thing/3f7db634-1cac-11e8-aaca-4574d7dabfb6",
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
