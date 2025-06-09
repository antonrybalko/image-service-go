package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/antonrybalko/image-service-go/internal/auth"
	"github.com/antonrybalko/image-service-go/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Router holds the HTTP router and its dependencies
type Router struct {
	router *chi.Mux
	logger *zap.SugaredLogger
	config *config.Config
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// NewRouter creates and configures a new router
func NewRouter(logger *zap.SugaredLogger, cfg *config.Config) *Router {
	r := &Router{
		router: chi.NewRouter(),
		logger: logger,
		config: cfg,
	}

	// Set up common middleware
	r.router.Use(middleware.RequestID)
	r.router.Use(middleware.RealIP)
	r.router.Use(middleware.Logger)
	r.router.Use(middleware.Recoverer)
	r.router.Use(middleware.Timeout(60 * time.Second))
	r.router.Use(middleware.AllowContentType("application/json", "image/jpeg", "image/png"))
	r.router.Use(middleware.SetHeader("Content-Type", "application/json"))

	// Set up routes
	r.setupRoutes()

	return r
}

// Handler returns the HTTP handler for the router
func (r *Router) Handler() http.Handler {
	return r.router
}

// setupRoutes configures all routes for the API
func (r *Router) setupRoutes() {
	// Public health check endpoint
	r.router.Get("/health", HealthHandler())

	// API v1 routes
	r.router.Route("/v1", func(v1 r chi.Router) {
		// Public routes
		v1.Get("/users/{userGuid}/image", r.handleGetUserImage())

		// Protected routes - require authentication
		v1.Group(func(auth r chi.Router) {
			// Apply JWT middleware to all routes in this group
			auth.Use(r.jwtAuth())

			// Current user routes
			auth.Route("/me", func(me r chi.Router) {
				me.Put("/image", r.handleUploadUserImage())
				me.Get("/image", r.handleGetCurrentUserImage())
				me.Delete("/image", r.handleDeleteUserImage())
			})
		})
	})
}

// jwtAuth creates a JWT authentication middleware
func (r *Router) jwtAuth() func(http.Handler) http.Handler {
	jwtConfig := auth.JWTConfig{
		PublicKeyURL: r.config.JWT.PublicKeyURL,
		Secret:       r.config.JWT.Secret,
		Algorithm:    r.config.JWT.Algorithm,
	}
	return auth.JWTMiddleware(jwtConfig)
}

// handleUploadUserImage handles PUT /v1/me/image
func (r *Router) handleUploadUserImage() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Get user ID from context (set by JWT middleware)
		userID, ok := auth.GetUserIDFromContext(req.Context())
		if !ok {
			r.writeError(w, http.StatusUnauthorized, "Unauthorized", "Invalid or missing authentication")
			return
		}

		// For Phase 0, just return a placeholder response
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Upload user image endpoint not yet implemented",
			"userID":  userID,
		})
	}
}

// handleGetCurrentUserImage handles GET /v1/me/image
func (r *Router) handleGetCurrentUserImage() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Get user ID from context (set by JWT middleware)
		userID, ok := auth.GetUserIDFromContext(req.Context())
		if !ok {
			r.writeError(w, http.StatusUnauthorized, "Unauthorized", "Invalid or missing authentication")
			return
		}

		// For Phase 0, just return a placeholder response
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Get current user image endpoint not yet implemented",
			"userID":  userID,
		})
	}
}

// handleDeleteUserImage handles DELETE /v1/me/image
func (r *Router) handleDeleteUserImage() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Get user ID from context (set by JWT middleware)
		userID, ok := auth.GetUserIDFromContext(req.Context())
		if !ok {
			r.writeError(w, http.StatusUnauthorized, "Unauthorized", "Invalid or missing authentication")
			return
		}

		// For Phase 0, just return a placeholder response
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Delete user image endpoint not yet implemented",
			"userID":  userID,
		})
	}
}

// handleGetUserImage handles GET /v1/users/{userGuid}/image
func (r *Router) handleGetUserImage() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Extract user GUID from URL path
		userGuid := chi.URLParam(req, "userGuid")
		if userGuid == "" {
			r.writeError(w, http.StatusBadRequest, "BadRequest", "User GUID is required")
			return
		}

		// For Phase 0, just return a placeholder response
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{
			"message":  "Get user image endpoint not yet implemented",
			"userGuid": userGuid,
		})
	}
}

// writeError writes a standardized error response
func (r *Router) writeError(w http.ResponseWriter, status int, errType, message string) {
	resp := ErrorResponse{
		Error:   errType,
		Message: message,
		Code:    status,
	}

	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		r.logger.Errorw("Failed to write error response", 
			"status", status,
			"error", errType,
			"message", message,
			"encoding_error", err,
		)
	}
}

// writeJSON writes a JSON response with the given status code
func (r *Router) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		r.logger.Errorw("Failed to write JSON response",
			"status", status,
			"error", err,
		)
	}
}

// logError logs an error with request details
func (r *Router) logError(req *http.Request, err error, message string, keysAndValues ...interface{}) {
	args := []interface{}{
		"error", err,
		"path", req.URL.Path,
		"method", req.Method,
		"remote_addr", req.RemoteAddr,
	}
	args = append(args, keysAndValues...)
	r.logger.Errorw(message, args...)
}

// logInfo logs information with request details
func (r *Router) logInfo(req *http.Request, message string, keysAndValues ...interface{}) {
	args := []interface{}{
		"path", req.URL.Path,
		"method", req.Method,
		"remote_addr", req.RemoteAddr,
	}
	args = append(args, keysAndValues...)
	r.logger.Infow(message, args...)
}
