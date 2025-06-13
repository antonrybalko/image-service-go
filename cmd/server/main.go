package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antonrybalko/image-service-go/internal/api"
	"github.com/antonrybalko/image-service-go/internal/config"
	"github.com/antonrybalko/image-service-go/internal/processor"
	"github.com/antonrybalko/image-service-go/internal/repository"
	"github.com/antonrybalko/image-service-go/internal/service"
	"github.com/antonrybalko/image-service-go/internal/storage"
	"go.uber.org/zap"
)

func main() {
	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	var logger *zap.Logger
	if cfg.Environment == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	sugar := logger.Sugar()
	sugar.Infow("Starting image service",
		"environment", cfg.Environment,
		"port", cfg.Port,
	)

	// Load image configuration from YAML
	imageConfig, err := config.LoadImageConfig(cfg.ImageConfigPath)
	if err != nil {
		sugar.Fatalw("Failed to load image configuration",
			"error", err,
			"path", cfg.ImageConfigPath)
	}
	sugar.Infow("Loaded image configuration",
		"types", len(imageConfig.Types),
		"path", cfg.ImageConfigPath)

	// Initialize repository
	// For Phase 1, we'll use a mock repository
	var imageRepo repository.ImageRepository
	if cfg.Environment == "production" || cfg.Environment == "staging" {
		// In production, we would initialize a real PostgreSQL connection
		db, err := initializeDatabase(cfg)
		if err != nil {
			sugar.Fatalw("Failed to initialize database",
				"error", err)
		}
		defer db.Close()
		imageRepo = repository.NewPostgresImageRepository(db)
		sugar.Info("Initialized PostgreSQL repository")
	} else {
		// For development and testing, use an in-memory mock
		imageRepo = repository.NewMockImageRepository()
		sugar.Info("Initialized mock repository")
	}

	// Initialize storage client
	var storageClient storage.S3Interface
	if cfg.Environment == "test" {
		// Use mock storage for tests
		storageClient = storage.NewMockS3()
		sugar.Info("Initialized mock S3 storage")
	} else {
		// Initialize real S3 client
		s3Config := storage.S3Config{
			Region:          cfg.S3.Region,
			Bucket:          cfg.S3.Bucket,
			AccessKeyID:     cfg.S3.AccessKeyID,
			SecretAccessKey: cfg.S3.SecretAccessKey,
			Endpoint:        cfg.S3.Endpoint,
			CDNBaseURL:      cfg.S3.CDNBaseURL,
			UsePathStyle:    cfg.S3.UsePathStyle,
		}
		storageClient, err = storage.NewS3Client(s3Config)
		if err != nil {
			sugar.Fatalw("Failed to initialize S3 storage client",
				"error", err)
		}
		sugar.Infow("Initialized S3 storage client",
			"region", cfg.S3.Region,
			"bucket", cfg.S3.Bucket,
			"endpoint", cfg.S3.Endpoint)
	}

	// Initialize image processor
	imageProcessor := processor.NewProcessor()
	sugar.Info("Initialized image processor")

	// Initialize image service
	imageService := service.NewImageService(
		imageRepo,
		storageClient,
		imageProcessor,
		imageConfig,
		sugar,
	)
	sugar.Info("Initialized image service")

	// Create router with all dependencies
	router := api.NewRouter(sugar, cfg, imageService)
	sugar.Info("Initialized router")

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine so that it doesn't block
	go func() {
		sugar.Infof("Server listening on port %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-quit
	sugar.Infof("Shutting down server: %v", sig)

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		sugar.Fatalf("Server forced to shutdown: %v", err)
	}

	sugar.Info("Server exited gracefully")
}

// initializeDatabase sets up the PostgreSQL database connection
func initializeDatabase(cfg *config.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name, cfg.DB.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
