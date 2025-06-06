package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/antonrybalko/image-service-go/internal/domain"
)

// Common errors
var (
	ErrNotFound      = errors.New("image not found")
	ErrAlreadyExists = errors.New("image already exists")
	ErrForcedError   = errors.New("forced error for testing")
)

// MockImageRepository is an in-memory implementation of the ImageRepository interface
// for testing purposes
type MockImageRepository struct {
	mu             sync.RWMutex
	userImages     map[string]*domain.Image
	orgImages      map[string]*domain.Image
	forceError     bool
	errorMessage   string
	saveUserCalls  int
	getUserCalls   int
	deleteUserCalls int
	saveOrgCalls   int
	getOrgCalls    int
	deleteOrgCalls int
	getPubUserCalls int
	getPubOrgCalls  int
}

// NewMockImageRepository creates a new mock repository
func NewMockImageRepository() *MockImageRepository {
	return &MockImageRepository{
		userImages: make(map[string]*domain.Image),
		orgImages:  make(map[string]*domain.Image),
	}
}

// SaveUserImage stores a user image in memory
func (m *MockImageRepository) SaveUserImage(
	ctx context.Context,
	userID, imageID string,
	smallURL, mediumURL, largeURL string,
) (*domain.Image, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.saveUserCalls++
	
	if m.forceError {
		return nil, errors.New(m.errorMessage)
	}
	
	now := time.Now()
	image := &domain.Image{
		GUID:      imageID,
		TypeID:    1, // User type ID
		TypeName:  "user",
		OwnerGUID: userID,
		SmallURL:  smallURL,
		MediumURL: mediumURL,
		LargeURL:  largeURL,
		CreatedAt: now,
		UpdatedAt: now,
	}
	
	m.userImages[userID] = image
	return image, nil
}

// GetUserImage retrieves a user image from memory
func (m *MockImageRepository) GetUserImage(ctx context.Context, userID string) (*domain.Image, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	m.getUserCalls++
	
	if m.forceError {
		return nil, errors.New(m.errorMessage)
	}
	
	image, exists := m.userImages[userID]
	if !exists {
		return nil, nil // Not found, but not an error
	}
	
	return image, nil
}

// DeleteUserImage removes a user image from memory
func (m *MockImageRepository) DeleteUserImage(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.deleteUserCalls++
	
	if m.forceError {
		return errors.New(m.errorMessage)
	}
	
	if _, exists := m.userImages[userID]; !exists {
		return ErrNotFound
	}
	
	delete(m.userImages, userID)
	return nil
}

// SaveOrganizationImage stores an organization image in memory
func (m *MockImageRepository) SaveOrganizationImage(
	ctx context.Context,
	orgID, imageID string,
	smallURL, mediumURL, largeURL string,
) (*domain.Image, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.saveOrgCalls++
	
	if m.forceError {
		return nil, errors.New(m.errorMessage)
	}
	
	now := time.Now()
	image := &domain.Image{
		GUID:      imageID,
		TypeID:    2, // Organization type ID
		TypeName:  "organization",
		OwnerGUID: orgID,
		SmallURL:  smallURL,
		MediumURL: mediumURL,
		LargeURL:  largeURL,
		CreatedAt: now,
		UpdatedAt: now,
	}
	
	m.orgImages[orgID] = image
	return image, nil
}

// GetOrganizationImage retrieves an organization image from memory
func (m *MockImageRepository) GetOrganizationImage(ctx context.Context, orgID string) (*domain.Image, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	m.getOrgCalls++
	
	if m.forceError {
		return nil, errors.New(m.errorMessage)
	}
	
	image, exists := m.orgImages[orgID]
	if !exists {
		return nil, nil // Not found, but not an error
	}
	
	return image, nil
}

// DeleteOrganizationImage removes an organization image from memory
func (m *MockImageRepository) DeleteOrganizationImage(ctx context.Context, orgID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.deleteOrgCalls++
	
	if m.forceError {
		return errors.New(m.errorMessage)
	}
	
	if _, exists := m.orgImages[orgID]; !exists {
		return ErrNotFound
	}
	
	delete(m.orgImages, orgID)
	return nil
}

// GetPublicUserImage retrieves a public user image from memory
func (m *MockImageRepository) GetPublicUserImage(ctx context.Context, userID string) (*domain.Image, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	m.getPubUserCalls++
	
	if m.forceError {
		return nil, errors.New(m.errorMessage)
	}
	
	image, exists := m.userImages[userID]
	if !exists {
		return nil, nil // Not found, but not an error
	}
	
	return image, nil
}

// GetPublicOrganizationImage retrieves a public organization image from memory
func (m *MockImageRepository) GetPublicOrganizationImage(ctx context.Context, orgID string) (*domain.Image, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	m.getPubOrgCalls++
	
	if m.forceError {
		return nil, errors.New(m.errorMessage)
	}
	
	image, exists := m.orgImages[orgID]
	if !exists {
		return nil, nil // Not found, but not an error
	}
	
	return image, nil
}

// SetError configures the mock to return an error on operations
func (m *MockImageRepository) SetError(enable bool, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.forceError = enable
	if enable {
		m.errorMessage = message
	} else {
		m.errorMessage = ""
	}
}

// Reset clears all stored images and resets call counters
func (m *MockImageRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.userImages = make(map[string]*domain.Image)
	m.orgImages = make(map[string]*domain.Image)
	m.saveUserCalls = 0
	m.getUserCalls = 0
	m.deleteUserCalls = 0
	m.saveOrgCalls = 0
	m.getOrgCalls = 0
	m.deleteOrgCalls = 0
	m.getPubUserCalls = 0
	m.getPubOrgCalls = 0
	m.forceError = false
	m.errorMessage = ""
}

// GetCallCounts returns the number of calls to each method
func (m *MockImageRepository) GetCallCounts() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return map[string]int{
		"SaveUserImage":             m.saveUserCalls,
		"GetUserImage":              m.getUserCalls,
		"DeleteUserImage":           m.deleteUserCalls,
		"SaveOrganizationImage":     m.saveOrgCalls,
		"GetOrganizationImage":      m.getOrgCalls,
		"DeleteOrganizationImage":   m.deleteOrgCalls,
		"GetPublicUserImage":        m.getPubUserCalls,
		"GetPublicOrganizationImage": m.getPubOrgCalls,
	}
}

// HasUserImage checks if a user image exists
func (m *MockImageRepository) HasUserImage(userID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	_, exists := m.userImages[userID]
	return exists
}

// HasOrganizationImage checks if an organization image exists
func (m *MockImageRepository) HasOrganizationImage(orgID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	_, exists := m.orgImages[orgID]
	return exists
}

// GetUserImageCount returns the number of user images
func (m *MockImageRepository) GetUserImageCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return len(m.userImages)
}

// GetOrganizationImageCount returns the number of organization images
func (m *MockImageRepository) GetOrganizationImageCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return len(m.orgImages)
}

// AddUserImage adds a user image directly (for test setup)
func (m *MockImageRepository) AddUserImage(image *domain.Image) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.userImages[image.OwnerGUID] = image
}

// AddOrganizationImage adds an organization image directly (for test setup)
func (m *MockImageRepository) AddOrganizationImage(image *domain.Image) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.orgImages[image.OwnerGUID] = image
}
