package api

import (
	"net/http"
	"time"

	"github.com/antonrybalko/image-service-go/internal/auth"
	"github.com/antonrybalko/image-service-go/internal/config"
	"github.com/antonrybalko/image-service-go/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Router holds the HTTP router and its dependencies
type Router struct {
	router       *chi.Mux
	logger       *zap.SugaredLogger
	config       *config.Config
	imageService *service.ImageService
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// NewRouter creates and configures a new router
func NewRouter(logger *zap.SugaredLogger, cfg *config.Config, imageService *service.ImageService) *Router {
	r := &Router{
		router:       chi.NewRouter(),
		logger:       logger,
		config:       cfg,
		imageService: imageService,
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
	// Create user image handlers
	userImageHandlers := NewUserImageHandlers(r.imageService)

	// Public health check endpoint
	r.router.Get("/health", HealthHandler())

	// API v1 routes
	r.router.Route("/v1", func(v1 chi.Router) {
		// Public routes
		v1.Get("/users/{userGuid}/image", userImageHandlers.GetUserImage())

		// Protected routes - require authentication
		v1.Group(func(auth chi.Router) {
			// Apply JWT middleware to all routes in this group
			auth.Use(r.jwtAuth())

			// Current user routes
			auth.Route("/me", func(me chi.Router) {
				me.Put("/image", userImageHandlers.UploadUserImage())
				me.Get("/image", userImageHandlers.GetCurrentUserImage())
				me.Delete("/image", userImageHandlers.DeleteUserImage())
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
