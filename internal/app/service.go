package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antonrybalko/image-service-go/internal/api"
	"github.com/antonrybalko/image-service-go/internal/auth"
	"github.com/antonrybalko/image-service-go/internal/config"
	"github.com/antonrybalko/image-service-go/internal/domain"
	"github.com/antonrybalko/image-service-go/internal/processor"
	"github.com/antonrybalko/image-service-go/internal/repository"
	"github.com/antonrybalko/image-service-go/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Version represents the application version
const Version = "0.1.0"

// Service represents the application service
type Service struct {
	config     *config.Config
	logger     *zap.Logger
	sugar      *zap.SugaredLogger
	router     chi.Router
	server     *http.Server
	db         *repository.PostgresRepository
	storage    storage.Interface
	processor  processor.Processor
	repository repository.ImageRepository
	authMW     *auth.JWTMiddleware
	imageTypes *domain.ImageConfig
}

// NewService creates a new application service
func NewService() (*Service, error) {
	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	var logger *zap.Logger
	if cfg.Environment == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	sugar := logger.Sugar()

	// Load image configuration
	imageTypes, err := config.LoadImageConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load image configuration: %w", err)
	}

	// Initialize storage
	var storageClient storage.Interface
	if cfg.Environment != "test" {
		// Use real S3 client
		storageClient, err = storage.NewS3Client(storage.Config{
			Region:          cfg.S3.Region,
			Bucket:          cfg.S3.Bucket,
			AccessKeyID:     cfg.S3.AccessKeyID,
			SecretAccessKey: cfg.S3.SecretAccessKey,
			Endpoint:        cfg.S3.Endpoint,
			CDNBaseURL:      cfg.S3.CDNBaseURL,
			UsePathStyle:    cfg.S3.UsePathStyle,
		}, sugar)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize storage: %w", err)
		}
	} else {
		// Use mock storage for testing
		storageClient = storage.NewMockS3Client("https://test-cdn.example.com")
	}

	// Initialize image processor
	imageProcessor := processor.New(imageTypes, sugar)

	// Initialize database and repository
	var repo repository.ImageRepository
	if cfg.Environment != "test" {
		// Connect to database
		dbConn, err := repository.NewDBConnection(repository.DBConfig{
			Host:     cfg.DB.Host,
			Port:     cfg.DB.Port,
			User:     cfg.DB.User,
			Password: cfg.DB.Password,
			Name:     cfg.DB.Name,
			SSLMode:  cfg.DB.SSLMode,
		}, sugar)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}

		// Create tables if they don't exist
		if err := repository.CreateTablesIfNotExist(dbConn, sugar); err != nil {
			return nil, fmt.Errorf("failed to create database tables: %w", err)
		}

		// Initialize repository
		postgresRepo := repository.NewPostgresRepository(dbConn, sugar)
		repo = postgresRepo
	} else {
		// Use mock repository for testing
		repo = repository.NewMockImageRepository()
	}

	// Initialize JWT middleware
	jwtMW := auth.NewJWTMiddleware(auth.Config{
		PublicKeyURL: cfg.JWT.PublicKeyURL,
		Secret:       cfg.JWT.Secret,
		Algorithm:    cfg.JWT.Algorithm,
	}, sugar)

	// Initialize router
	router := chi.NewRouter()

	// Initialize API handler
	handler := api.NewHandler(imageProcessor, storageClient, repo, sugar)

	// Register routes
	api.RegisterRoutes(router, handler, sugar, Version, cfg.Environment)

	// Initialize HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Service{
		config:     cfg,
		logger:     logger,
		sugar:      sugar,
		router:     router,
		server:     server,
		storage:    storageClient,
		processor:  imageProcessor,
		repository: repo,
		authMW:     jwtMW,
		imageTypes: imageTypes,
	}, nil
}

// Start starts the service
func (s *Service) Start() error {
	// Log service startup
	s.sugar.Infow("Starting image service",
		"version", Version,
		"environment", s.config.Environment,
		"port", s.config.Port,
	)

	// Start server in a goroutine
	go func() {
		s.sugar.Infof("Server listening on port %d", s.config.Port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.sugar.Fatalf("Server failed: %v", err)
		}
	}()

	return nil
}

// WaitForShutdown waits for a shutdown signal and gracefully shuts down the server
func (s *Service) WaitForShutdown() {
	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-quit
	s.sugar.Infof("Shutting down server: %v", sig)

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.server.Shutdown(ctx); err != nil {
		s.sugar.Fatalf("Server forced to shutdown: %v", err)
	}

	s.sugar.Info("Server exited gracefully")
}

// Cleanup performs cleanup tasks
func (s *Service) Cleanup() {
	// Sync logger
	if err := s.logger.Sync(); err != nil {
		fmt.Printf("Failed to sync logger: %v\n", err)
	}

	// Add any other cleanup tasks here
	s.sugar.Info("Cleanup completed")
}
