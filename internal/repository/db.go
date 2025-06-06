package repository

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"go.uber.org/zap"
)

// DBConfig holds database connection configuration
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// NewDBConnection creates a new database connection pool
func NewDBConnection(cfg DBConfig, logger *zap.SugaredLogger) (*sql.DB, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close() // Close the connection if ping fails
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Infow("Connected to database",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Name,
		"user", cfg.User,
	)

	return db, nil
}

// NewMockDBConnection creates a mock database connection for testing
func NewMockDBConnection() (*sql.DB, error) {
	// This is a placeholder for Phase 0
	// In a real implementation, we might use an in-memory SQLite database
	// or a more sophisticated mock
	return nil, nil
}

// CreateTablesIfNotExist creates the necessary database tables if they don't exist
func CreateTablesIfNotExist(db *sql.DB, logger *zap.SugaredLogger) error {
	// Create image_types table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS image_types (
			id          SERIAL PRIMARY KEY,
			name        TEXT UNIQUE NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create image_types table: %w", err)
	}

	// Create images table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS images (
			guid            UUID PRIMARY KEY,
			type_id         INT REFERENCES image_types(id),
			owner_guid      UUID NOT NULL,
			small_url       TEXT NOT NULL,
			medium_url      TEXT NOT NULL,
			large_url       TEXT NOT NULL,
			created_at      TIMESTAMPTZ DEFAULT now(),
			updated_at      TIMESTAMPTZ DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create images table: %w", err)
	}

	// Create index on owner_guid
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_images_owner ON images(owner_guid)
	`)
	if err != nil {
		return fmt.Errorf("failed to create index on images.owner_guid: %w", err)
	}

	// Insert default image types if they don't exist
	for _, typeName := range []string{"user", "organization", "product"} {
		_, err = db.Exec(`
			INSERT INTO image_types (name)
			VALUES ($1)
			ON CONFLICT (name) DO NOTHING
		`, typeName)
		if err != nil {
			return fmt.Errorf("failed to insert image type %s: %w", typeName, err)
		}
	}

	logger.Info("Database tables created or verified")
	return nil
}

// CloseDB gracefully closes the database connection
func CloseDB(db *sql.DB, logger *zap.SugaredLogger) {
	if db != nil {
		if err := db.Close(); err != nil {
			logger.Errorw("Error closing database connection", "error", err)
		} else {
			logger.Info("Database connection closed")
		}
	}
}
