package api

import (
	"net/http"
)

// HealthHandler returns a simple health check handler function
// that responds with a 200 OK status and JSON {"status":"ok"}
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}
