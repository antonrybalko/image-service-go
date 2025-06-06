package storage

import (
	"context"
	"fmt"
	"sync"
)

// MockS3Client implements the storage Interface for testing
type MockS3Client struct {
	mu            sync.RWMutex
	images        map[string][]byte
	contentTypes  map[string]string
	baseURL       string
	uploadCalls   int
	deleteCalls   int
	getURLCalls   int
	forceError    bool
	errorMessage  string
}

// NewMockS3Client creates a new mock S3 client for testing
func NewMockS3Client(baseURL string) *MockS3Client {
	if baseURL == "" {
		baseURL = "https://mock-s3.example.com"
	}
	
	return &MockS3Client{
		images:       make(map[string][]byte),
		contentTypes: make(map[string]string),
		baseURL:      baseURL,
	}
}

// UploadImage stores an image in memory and returns a mock URL
func (m *MockS3Client) UploadImage(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.uploadCalls++
	
	if m.forceError {
		return "", fmt.Errorf(m.errorMessage)
	}
	
	// Make a copy of the data to avoid external modifications
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	
	m.images[key] = dataCopy
	m.contentTypes[key] = contentType
	
	return m.GetImageURL(key), nil
}

// DeleteImage removes an image from memory
func (m *MockS3Client) DeleteImage(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.deleteCalls++
	
	if m.forceError {
		return fmt.Errorf(m.errorMessage)
	}
	
	if _, exists := m.images[key]; !exists {
		return fmt.Errorf("image with key %s not found", key)
	}
	
	delete(m.images, key)
	delete(m.contentTypes, key)
	
	return nil
}

// GetImageURL returns a mock URL for the given key
func (m *MockS3Client) GetImageURL(key string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.getURLCalls++
	
	return fmt.Sprintf("%s/%s", m.baseURL, key)
}

// GetImage retrieves an image from memory (helper for tests)
func (m *MockS3Client) GetImage(key string) ([]byte, string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	data, exists := m.images[key]
	if !exists {
		return nil, "", false
	}
	
	contentType := m.contentTypes[key]
	
	return data, contentType, true
}

// SetError configures the mock to return an error on operations
func (m *MockS3Client) SetError(enable bool, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.forceError = enable
	if enable {
		m.errorMessage = message
	} else {
		m.errorMessage = ""
	}
}

// GetCallCounts returns the number of calls to each method (helper for tests)
func (m *MockS3Client) GetCallCounts() (uploads, deletes, getURLs int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.uploadCalls, m.deleteCalls, m.getURLCalls
}

// Reset clears all stored images and resets call counters
func (m *MockS3Client) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.images = make(map[string][]byte)
	m.contentTypes = make(map[string]string)
	m.uploadCalls = 0
	m.deleteCalls = 0
	m.getURLCalls = 0
	m.forceError = false
	m.errorMessage = ""
}

// ImageCount returns the number of images stored (helper for tests)
func (m *MockS3Client) ImageCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return len(m.images)
}

// HasImage checks if an image with the given key exists (helper for tests)
func (m *MockS3Client) HasImage(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	_, exists := m.images[key]
	return exists
}
