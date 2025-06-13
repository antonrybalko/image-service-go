package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/antonrybalko/image-service-go/internal/domain"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// PostgresImageRepository implements ImageRepository using PostgreSQL
type PostgresImageRepository struct {
	db *sql.DB
}

// NewPostgresImageRepository creates a new PostgresImageRepository
func NewPostgresImageRepository(db *sql.DB) *PostgresImageRepository {
	return &PostgresImageRepository{
		db: db,
	}
}

// SaveImage saves a new image or updates an existing one
func (r *PostgresImageRepository) SaveImage(ctx context.Context, image *domain.Image) error {
	// Use a transaction for atomicity
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Check if the image already exists
	var exists bool
	err = tx.QueryRowContext(ctx, 
		`SELECT EXISTS(SELECT 1 FROM images WHERE guid = $1)`,
		image.GUID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	now := time.Now().UTC()

	if exists {
		// Update existing image
		_, err = tx.ExecContext(ctx, `
			UPDATE images 
			SET owner_guid = $1, 
				type_name = $2, 
				small_url = $3, 
				medium_url = $4, 
				large_url = $5, 
				updated_at = $6,
				content_type = $7,
				original_width = $8,
				original_height = $9
			WHERE guid = $10`,
			image.OwnerGUID,
			image.TypeName,
			image.SmallURL,
			image.MediumURL,
			image.LargeURL,
			now,
			image.ContentType,
			image.OriginalWidth,
			image.OriginalHeight,
			image.GUID)
	} else {
		// Insert new image
		_, err = tx.ExecContext(ctx, `
			INSERT INTO images (
				guid, owner_guid, type_name, small_url, medium_url, large_url, 
				created_at, updated_at, content_type, original_width, original_height
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			image.GUID,
			image.OwnerGUID,
			image.TypeName,
			image.SmallURL,
			image.MediumURL,
			image.LargeURL,
			image.CreatedAt,
			now,
			image.ContentType,
			image.OriginalWidth,
			image.OriginalHeight)
	}

	if err != nil {
		// Check for unique constraint violation
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("%w: %v", ErrAlreadyExists, err)
		}
		return fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	// Update the image's updated_at timestamp
	image.UpdatedAt = now

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	return nil
}

// GetImageByID retrieves an image by its GUID
func (r *PostgresImageRepository) GetImageByID(ctx context.Context, imageGUID uuid.UUID) (*domain.Image, error) {
	var image domain.Image

	err := r.db.QueryRowContext(ctx, `
		SELECT guid, owner_guid, type_name, small_url, medium_url, large_url, 
			   created_at, updated_at, content_type, original_width, original_height
		FROM images
		WHERE guid = $1`,
		imageGUID).Scan(
		&image.GUID,
		&image.OwnerGUID,
		&image.TypeName,
		&image.SmallURL,
		&image.MediumURL,
		&image.LargeURL,
		&image.CreatedAt,
		&image.UpdatedAt,
		&image.ContentType,
		&image.OriginalWidth,
		&image.OriginalHeight)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	return &image, nil
}

// GetImageByOwner retrieves an image by owner GUID and type
func (r *PostgresImageRepository) GetImageByOwner(ctx context.Context, ownerGUID uuid.UUID, typeName string) (*domain.Image, error) {
	var image domain.Image

	err := r.db.QueryRowContext(ctx, `
		SELECT guid, owner_guid, type_name, small_url, medium_url, large_url, 
			   created_at, updated_at, content_type, original_width, original_height
		FROM images
		WHERE owner_guid = $1 AND type_name = $2`,
		ownerGUID, typeName).Scan(
		&image.GUID,
		&image.OwnerGUID,
		&image.TypeName,
		&image.SmallURL,
		&image.MediumURL,
		&image.LargeURL,
		&image.CreatedAt,
		&image.UpdatedAt,
		&image.ContentType,
		&image.OriginalWidth,
		&image.OriginalHeight)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	return &image, nil
}

// DeleteImage deletes an image by its GUID
func (r *PostgresImageRepository) DeleteImage(ctx context.Context, imageGUID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM images
		WHERE guid = $1`,
		imageGUID)

	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteImageByOwner deletes an image by owner GUID and type
func (r *PostgresImageRepository) DeleteImageByOwner(ctx context.Context, ownerGUID uuid.UUID, typeName string) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM images
		WHERE owner_guid = $1 AND type_name = $2`,
		ownerGUID, typeName)

	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// ListImagesByType lists all images of a specific type
func (r *PostgresImageRepository) ListImagesByType(ctx context.Context, typeName string, limit, offset int) ([]*domain.Image, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT guid, owner_guid, type_name, small_url, medium_url, large_url, 
			   created_at, updated_at, content_type, original_width, original_height
		FROM images
		WHERE type_name = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		typeName, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	defer rows.Close()

	var images []*domain.Image

	for rows.Next() {
		var image domain.Image
		err := rows.Scan(
			&image.GUID,
			&image.OwnerGUID,
			&image.TypeName,
			&image.SmallURL,
			&image.MediumURL,
			&image.LargeURL,
			&image.CreatedAt,
			&image.UpdatedAt,
			&image.ContentType,
			&image.OriginalWidth,
			&image.OriginalHeight)

		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
		}

		images = append(images, &image)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	return images, nil
}

// CreateImagesTable creates the images table if it doesn't exist
func (r *PostgresImageRepository) CreateImagesTable(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS images (
			guid UUID PRIMARY KEY,
			owner_guid UUID NOT NULL,
			type_name TEXT NOT NULL,
			small_url TEXT NOT NULL,
			medium_url TEXT NOT NULL,
			large_url TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			content_type TEXT,
			original_width INTEGER,
			original_height INTEGER
		);
		
		CREATE INDEX IF NOT EXISTS idx_images_owner_type ON images (owner_guid, type_name);
		CREATE INDEX IF NOT EXISTS idx_images_type ON images (type_name);
	`)

	if err != nil {
		return fmt.Errorf("failed to create images table: %w", err)
	}

	return nil
}

// WithTransaction executes a function within a transaction
func (r *PostgresImageRepository) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	defer func() {
		if p := recover(); p != nil {
			// Rollback on panic
			_ = tx.Rollback()
			panic(p) // Re-throw panic after rollback
		} else if err != nil {
			// Rollback on error
			_ = tx.Rollback()
		} else {
			// Commit if no error or panic
			err = tx.Commit()
			if err != nil {
				err = fmt.Errorf("%w: %v", ErrDatabase, err)
			}
		}
	}()

	err = fn(tx)
	return err
}
