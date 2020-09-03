package health

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHappyHealthCheck(t *testing.T) {
	draftContentRW := mockHealthyExternalService()
	draftContentMapper := mockHealthyExternalService()
	cAPI := mockHealthyExternalService()
	ucv := mockHealthyExternalService()
	draftContentPlaceholderValidator := mockHealthyExternalService()
	liveBlogPostValidator := mockHealthyExternalService()

	h := NewHealthService("", "", "", draftContentRW, draftContentMapper, cAPI, ucv, draftContentPlaceholderValidator, liveBlogPostValidator)

	req := httptest.NewRequest("GET", "/__health", nil)
	w := httptest.NewRecorder()
	h.HealthCheckHandleFunc()(w, req)

	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	hcBody := make(map[string]interface{})
	err := json.NewDecoder(resp.Body).Decode(&hcBody)
	assert.NoError(t, err)
	assert.Len(t, hcBody["checks"], 6)
	assert.True(t, hcBody["ok"].(bool))

	checks := hcBody["checks"].([]interface{})
	for _, c := range checks {
		check := c.(map[string]interface{})
		assert.True(t, check["ok"].(bool))

		if check["id"] == "check-content-api-health" {
			assert.Equal(t, "Content API is good-to-go", check["checkOutput"])
			assert.Equal(t, "Content API is not available at http://cool.api.ft.com/content", check["technicalSummary"])
		}
	}

	cAPI.AssertExpectations(t)
}

func TestUnhappyHealthCheck(t *testing.T) {
	draftContentRW := mockHealthyExternalService()
	draftContentMapper := mockHealthyExternalService()
	draftContentValidator := mockHealthyExternalService()
	draftContentPlaceholderValidator := mockHealthyExternalService()
	liveBlogPostValidator := mockHealthyExternalService()

	cAPI := new(ExternalServiceMock)
	cAPI.On("GTG").Return(errors.New("computer says no"))
	cAPI.On("Endpoint").Return("http://cool.api.ft.com/content")

	h := NewHealthService("", "", "", draftContentRW, draftContentMapper, cAPI, draftContentValidator, draftContentPlaceholderValidator, liveBlogPostValidator)

	req := httptest.NewRequest("GET", "/__health", nil)
	w := httptest.NewRecorder()
	h.HealthCheckHandleFunc()(w, req)

	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	hcBody := make(map[string]interface{})
	err := json.NewDecoder(resp.Body).Decode(&hcBody)
	assert.NoError(t, err)
	assert.Len(t, hcBody["checks"], 6)
	assert.False(t, hcBody["ok"].(bool))

	checks := hcBody["checks"].([]interface{})
	for _, c := range checks {
		check := c.(map[string]interface{})
		if check["id"] == "check-content-api-health" {
			assert.False(t, check["ok"].(bool))
			assert.Equal(t, "computer says no", check["checkOutput"])
			assert.Equal(t, "Content API is not available at http://cool.api.ft.com/content", check["technicalSummary"])
			break
		}
	}

	cAPI.AssertExpectations(t)
}

func TestHappyGTG(t *testing.T) {
	draftContentRW := mockHealthyExternalService()
	draftContentMapper := mockHealthyExternalService()
	draftContentValidator := mockHealthyExternalService()
	draftContentPlaceholderValidator := mockHealthyExternalService()
	liveBlogPostValidator := mockHealthyExternalService()

	cAPI := mockHealthyExternalService()

	h := NewHealthService("", "", "", draftContentRW, draftContentMapper, cAPI, draftContentValidator, draftContentPlaceholderValidator, liveBlogPostValidator)

	req := httptest.NewRequest("GET", "/__gtg", nil)
	w := httptest.NewRecorder()
	status.NewGoodToGoHandler(h.GTGChecker())(w, req)

	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cAPI.AssertExpectations(t)
}

func TestUnhappyGTG(t *testing.T) {
	draftContentRW := mockHealthyExternalService()
	draftContentMapper := mockHealthyExternalService()
	draftContentValidator := mockHealthyExternalService()
	draftContentPlaceholderValidator := mockHealthyExternalService()
	liveBlogPostValidator := mockHealthyExternalService()

	cAPI := new(ExternalServiceMock)
	cAPI.On("GTG").Return(errors.New("computer says no"))
	cAPI.On("Endpoint").Return("http://cool.api.ft.com/content")
	h := NewHealthService("", "", "", draftContentRW, draftContentMapper, cAPI, draftContentValidator, draftContentPlaceholderValidator, liveBlogPostValidator)

	req := httptest.NewRequest("GET", "/__gtg", nil)
	w := httptest.NewRecorder()
	status.NewGoodToGoHandler(h.GTGChecker())(w, req)

	resp := w.Result()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "computer says no", string(body))

	cAPI.AssertExpectations(t)
}

type ExternalServiceMock struct {
	mock.Mock
}

func (m *ExternalServiceMock) GTG() error {
	args := m.Called()
	return args.Error(0)
}

func (m *ExternalServiceMock) Endpoint() string {
	args := m.Called()
	return args.String(0)
}

func mockHealthyExternalService() *ExternalServiceMock {
	srv := new(ExternalServiceMock)
	srv.On("GTG").Return(nil)
	srv.On("Endpoint").Return("http://cool.api.ft.com/content")

	return srv
}
