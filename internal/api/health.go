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
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			// If we can't write the response, there's not much we can do
			// The status code has already been set, so just return
			return
		}
	}
}
