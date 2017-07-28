package health

import (
	"context"
	"encoding/json"
	"errors"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHappyHealthCheck(t *testing.T) {
	cAPI := new(ContentAPIMock)
	cAPI.On("GTG").Return(nil)
	h := NewHealthService("", "", "", cAPI)

	req := httptest.NewRequest("GET", "/__health", nil)
	w := httptest.NewRecorder()
	h.HealthCheckHandleFunc()(w, req)

	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	hcBody := make(map[string]interface{})
	err := json.NewDecoder(resp.Body).Decode(&hcBody)
	assert.NoError(t, err)
	assert.Len(t, hcBody["checks"], 1)
	assert.True(t, hcBody["ok"].(bool))
	check := hcBody["checks"].([]interface{})[0].(map[string]interface{})
	assert.True(t, check["ok"].(bool))
	assert.Equal(t, "Content API is healthy", check["checkOutput"])
}

func TestUnhappyHealthCheck(t *testing.T) {
	cAPI := new(ContentAPIMock)
	cAPI.On("GTG").Return(errors.New("computer says no"))
	h := NewHealthService("", "", "", cAPI)

	req := httptest.NewRequest("GET", "/__health", nil)
	w := httptest.NewRecorder()
	h.HealthCheckHandleFunc()(w, req)

	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	hcBody := make(map[string]interface{})
	err := json.NewDecoder(resp.Body).Decode(&hcBody)
	assert.NoError(t, err)
	assert.Len(t, hcBody["checks"], 1)
	assert.False(t, hcBody["ok"].(bool))
	check := hcBody["checks"].([]interface{})[0].(map[string]interface{})
	assert.False(t, check["ok"].(bool))
	assert.Equal(t, "computer says no", check["checkOutput"])
}

func TestHappyGTG(t *testing.T) {
	cAPI := new(ContentAPIMock)
	cAPI.On("GTG").Return(nil)
	h := NewHealthService("", "", "", cAPI)

	req := httptest.NewRequest("GET", "/__gtg", nil)
	w := httptest.NewRecorder()
	status.NewGoodToGoHandler(h.GTG)(w, req)

	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUnhappyGTG(t *testing.T) {
	cAPI := new(ContentAPIMock)
	cAPI.On("GTG").Return(errors.New("computer says no"))
	h := NewHealthService("", "", "", cAPI)

	req := httptest.NewRequest("GET", "/__gtg", nil)
	w := httptest.NewRecorder()
	status.NewGoodToGoHandler(h.GTG)(w, req)

	resp := w.Result()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "computer says no", string(body))
}

type ContentAPIMock struct {
	mock.Mock
}

func (m ContentAPIMock) Get(ctx context.Context, contentUUID string) (*http.Response, error) {
	args := m.Called(ctx, contentUUID)
	return args.Get(0).(*http.Response), args.Error(1)
}

func (m ContentAPIMock) GTG() error {
	args := m.Called()
	return args.Error(0)
}
