package api

import (
	"net/http"
	"time"

	"github.com/antonrybalko/image-service-go/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// RegisterRoutes configures all routes for the application
func RegisterRoutes(r chi.Router, handler Handler, logger *zap.SugaredLogger, version string, env string) {
	// Set up middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	
	// Add CORS middleware for development
	if env != "production" {
		r.Use(middleware.AllowContentType("image/jpeg", "image/png"))
		r.Use(middleware.SetHeader("Access-Control-Allow-Origin", "*"))
		r.Use(middleware.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS"))
		r.Use(middleware.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization"))
	}

	// Register health check routes
	RegisterHealthRoutes(r, logger, version, env)

	// API v1 routes
	r.Route("/v1", func(r chi.Router) {
		// Public routes (no auth required)
		r.Get("/users/{userGuid}/image", handler.GetPublicUserImage)
		r.Get("/organizations/{orgGuid}/image", handler.GetPublicOrganizationImage)
		
		// Private routes (require authentication)
		r.Group(func(r chi.Router) {
			// Apply JWT authentication middleware
			r.Use(auth.RequireAuth)
			
			// User image routes
			r.Route("/me", func(r chi.Router) {
				// User's own image
				r.Route("/image", func(r chi.Router) {
					r.Put("/", handler.UploadUserImage)
					r.Get("/", handler.GetUserImage)
					r.Delete("/", handler.DeleteUserImage)
				})
				
				// Organization images
				r.Route("/organizations/{orgGuid}/image", func(r chi.Router) {
					r.Put("/", handler.UploadOrganizationImage)
					r.Get("/", handler.GetOrganizationImage)
					r.Delete("/", handler.DeleteOrganizationImage)
				})
			})
		})
	})

	// Not found handler
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Not found"}`))
	})

	// Method not allowed handler
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error":"Method not allowed"}`))
	})
}
