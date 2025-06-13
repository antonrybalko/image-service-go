package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/antonrybalko/image-service-go/internal/domain"
	"github.com/google/uuid"
)

// Common repository errors
var (
	ErrNotFound      = errors.New("image not found")
	ErrAlreadyExists = errors.New("image already exists")
	ErrDatabase      = errors.New("database error")
)

// ImageRepository defines the operations for image metadata storage
type ImageRepository interface {
	// SaveImage saves a new image or updates an existing one
	SaveImage(ctx context.Context, image *domain.Image) error

	// GetImageByID retrieves an image by its GUID
	GetImageByID(ctx context.Context, imageGUID uuid.UUID) (*domain.Image, error)

	// GetImageByOwner retrieves an image by owner GUID and type
	GetImageByOwner(ctx context.Context, ownerGUID uuid.UUID, typeName string) (*domain.Image, error)

	// DeleteImage deletes an image by its GUID
	DeleteImage(ctx context.Context, imageGUID uuid.UUID) error

	// DeleteImageByOwner deletes an image by owner GUID and type
	DeleteImageByOwner(ctx context.Context, ownerGUID uuid.UUID, typeName string) error

	// ListImagesByType lists all images of a specific type
	ListImagesByType(ctx context.Context, typeName string, limit, offset int) ([]*domain.Image, error)
}

// MockImageRepository implements ImageRepository for testing
type MockImageRepository struct {
	mutex  sync.RWMutex
	images map[uuid.UUID]*domain.Image
	byOwner map[string]*domain.Image // key is ownerGUID + typeName
}

// NewMockImageRepository creates a new MockImageRepository
func NewMockImageRepository() *MockImageRepository {
	return &MockImageRepository{
		images: make(map[uuid.UUID]*domain.Image),
		byOwner: make(map[string]*domain.Image),
	}
}

// SaveImage saves a new image or updates an existing one
func (m *MockImageRepository) SaveImage(ctx context.Context, image *domain.Image) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Ensure image has required fields
	if image.GUID == uuid.Nil {
		return errors.New("image GUID is required")
	}
	if image.OwnerGUID == uuid.Nil {
		return errors.New("owner GUID is required")
	}
	if image.TypeName == "" {
		return errors.New("type name is required")
	}

	// Update timestamps
	now := time.Now().UTC()
	if image.CreatedAt.IsZero() {
		image.CreatedAt = now
	}
	image.UpdatedAt = now

	// Store by ID
	m.images[image.GUID] = image

	// Store by owner + type
	ownerKey := ownerTypeKey(image.OwnerGUID, image.TypeName)
	m.byOwner[ownerKey] = image

	return nil
}

// GetImageByID retrieves an image by its GUID
func (m *MockImageRepository) GetImageByID(ctx context.Context, imageGUID uuid.UUID) (*domain.Image, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	image, exists := m.images[imageGUID]
	if !exists {
		return nil, ErrNotFound
	}

	// Return a copy to prevent modification of the stored image
	imageCopy := *image
	return &imageCopy, nil
}

// GetImageByOwner retrieves an image by owner GUID and type
func (m *MockImageRepository) GetImageByOwner(ctx context.Context, ownerGUID uuid.UUID, typeName string) (*domain.Image, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	ownerKey := ownerTypeKey(ownerGUID, typeName)
	image, exists := m.byOwner[ownerKey]
	if !exists {
		return nil, ErrNotFound
	}

	// Return a copy to prevent modification of the stored image
	imageCopy := *image
	return &imageCopy, nil
}

// DeleteImage deletes an image by its GUID
func (m *MockImageRepository) DeleteImage(ctx context.Context, imageGUID uuid.UUID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	image, exists := m.images[imageGUID]
	if !exists {
		return ErrNotFound
	}

	// Remove from both maps
	delete(m.images, imageGUID)
	ownerKey := ownerTypeKey(image.OwnerGUID, image.TypeName)
	delete(m.byOwner, ownerKey)

	return nil
}

// DeleteImageByOwner deletes an image by owner GUID and type
func (m *MockImageRepository) DeleteImageByOwner(ctx context.Context, ownerGUID uuid.UUID, typeName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	ownerKey := ownerTypeKey(ownerGUID, typeName)
	image, exists := m.byOwner[ownerKey]
	if !exists {
		return ErrNotFound
	}

	// Remove from both maps
	delete(m.images, image.GUID)
	delete(m.byOwner, ownerKey)

	return nil
}

// ListImagesByType lists all images of a specific type
func (m *MockImageRepository) ListImagesByType(ctx context.Context, typeName string, limit, offset int) ([]*domain.Image, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*domain.Image
	for _, image := range m.images {
		if image.TypeName == typeName {
			// Create a copy of the image
			imageCopy := *image
			result = append(result, &imageCopy)
		}
	}

	// Apply pagination
	if offset >= len(result) {
		return []*domain.Image{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

// --- Test Helper Methods ---

// GetImageCount returns the number of images in the mock repository
func (m *MockImageRepository) GetImageCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.images)
}

// ClearImages removes all images from the mock repository
func (m *MockImageRepository) ClearImages() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.images = make(map[uuid.UUID]*domain.Image)
	m.byOwner = make(map[string]*domain.Image)
}

// Helper function to create a key for owner + type lookups
func ownerTypeKey(ownerGUID uuid.UUID, typeName string) string {
	return ownerGUID.String() + ":" + typeName
}
