package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// HealthResponse represents the expected JSON response from the health endpoint
type HealthResponse struct {
	Status string `json:"status"`
}

func TestHealthHandler(t *testing.T) {
	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := HealthHandler()

	// Call the handler directly with the request and response recorder
	handler.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	// Check the content type
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"), "handler returned wrong content type")

	// Parse the response body
	var resp HealthResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err, "failed to parse response body")

	// Check the response body
	assert.Equal(t, "ok", resp.Status, "handler returned wrong status in body")
}
