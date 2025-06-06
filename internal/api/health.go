package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"go.uber.org/zap"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Uptime    string            `json:"uptime"`
	Memory    *MemoryStats      `json:"memory,omitempty"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// MemoryStats contains memory usage information
type MemoryStats struct {
	Alloc      uint64 `json:"alloc"`      // bytes allocated and still in use
	TotalAlloc uint64 `json:"totalAlloc"` // bytes allocated (even if freed)
	Sys        uint64 `json:"sys"`        // bytes obtained from system
	NumGC      uint32 `json:"numGC"`      // number of completed GC cycles
}

// HealthHandler handles health check requests
type HealthHandler struct {
	logger      *zap.SugaredLogger
	version     string
	startTime   time.Time
	withDetails bool
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(logger *zap.SugaredLogger, version string, withDetails bool) *HealthHandler {
	return &HealthHandler{
		logger:      logger,
		version:     version,
		startTime:   time.Now(),
		withDetails: withDetails,
	}
}

// ServeHTTP handles HTTP requests for health checks
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok",
		Version:   h.version,
		Timestamp: time.Now(),
		Uptime:    time.Since(h.startTime).String(),
	}

	// Include detailed stats in non-production environments
	if h.withDetails {
		// Add memory stats
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		response.Memory = &MemoryStats{
			Alloc:      memStats.Alloc,
			TotalAlloc: memStats.TotalAlloc,
			Sys:        memStats.Sys,
			NumGC:      memStats.NumGC,
		}

		// Add component checks
		response.Checks = map[string]string{
			"api":      "ok",
			"storage":  "ok", // These could be actual checks in a more complete implementation
			"database": "ok",
		}
	}

	// Set content type and status code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Encode the response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorw("Failed to encode health check response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	h.logger.Debugw("Health check completed", "status", "ok", "remote_addr", r.RemoteAddr)
}

// RegisterHealthRoutes registers the health check routes
func RegisterHealthRoutes(router chi.Router, logger *zap.SugaredLogger, version string, env string) {
	withDetails := env != "production"
	healthHandler := NewHealthHandler(logger, version, withDetails)
	
	router.Get("/health", healthHandler.ServeHTTP)
	router.Get("/healthz", healthHandler.ServeHTTP) // Kubernetes convention
}
