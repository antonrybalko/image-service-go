package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// MockS3 implements S3Interface for testing purposes
type MockS3 struct {
	objects     map[string][]byte
	contentType map[string]string
	urls        map[string]string
	bucket      string
	region      string
	cdnBaseURL  string
	mutex       sync.RWMutex
}

// NewMockS3 creates a new mock S3 client for testing
func NewMockS3() *MockS3 {
	return &MockS3{
		objects:     make(map[string][]byte),
		contentType: make(map[string]string),
		urls:        make(map[string]string),
		bucket:      "test-bucket",
		region:      "us-east-1",
		cdnBaseURL:  "https://cdn.example.com",
	}
}

// Put mocks uploading an object to S3
func (m *MockS3) Put(ctx context.Context, key string, body []byte, contentType string) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Store the object in memory
	m.objects[key] = body
	m.contentType[key] = contentType
	
	// Generate and store URL
	url := m.GetURL(key)
	m.urls[key] = url
	
	return url, nil
}

// Get mocks retrieving an object from S3
func (m *MockS3) Get(ctx context.Context, key string) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	data, exists := m.objects[key]
	if !exists {
		return nil, fmt.Errorf("object not found: %s", key)
	}
	
	return data, nil
}

// Delete mocks removing an object from S3
func (m *MockS3) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if _, exists := m.objects[key]; !exists {
		return fmt.Errorf("object not found: %s", key)
	}
	
	delete(m.objects, key)
	delete(m.contentType, key)
	delete(m.urls, key)
	
	return nil
}

// GenerateUserImageKey generates a consistent key for user images
func (m *MockS3) GenerateUserImageKey(userGUID uuid.UUID, imageGUID uuid.UUID, size string) string {
	return fmt.Sprintf("images/user/%s/%s/%s.jpg", userGUID.String(), imageGUID.String(), size)
}

// GenerateOrganizationImageKey generates a consistent key for organization images
func (m *MockS3) GenerateOrganizationImageKey(orgGUID uuid.UUID, imageGUID uuid.UUID, size string) string {
	return fmt.Sprintf("images/organization/%s/%s/%s.jpg", orgGUID.String(), imageGUID.String(), size)
}

// GetURL returns the URL for an object
func (m *MockS3) GetURL(key string) string {
	if m.cdnBaseURL != "" {
		return fmt.Sprintf("%s/%s", m.cdnBaseURL, key)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", m.bucket, m.region, key)
}

// --- Test Helper Methods ---

// HasObject checks if an object exists in the mock storage
func (m *MockS3) HasObject(key string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	_, exists := m.objects[key]
	return exists
}

// GetObjectCount returns the number of objects in the mock storage
func (m *MockS3) GetObjectCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return len(m.objects)
}

// ClearObjects removes all objects from the mock storage
func (m *MockS3) ClearObjects() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.objects = make(map[string][]byte)
	m.contentType = make(map[string]string)
	m.urls = make(map[string]string)
}

// GetContentType returns the content type for a key
func (m *MockS3) GetContentType(key string) (string, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	ct, exists := m.contentType[key]
	return ct, exists
}

// GetStoredURL returns the stored URL for a key
func (m *MockS3) GetStoredURL(key string) (string, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	url, exists := m.urls[key]
	return url, exists
}

// SetBucket sets the bucket name for testing
func (m *MockS3) SetBucket(bucket string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.bucket = bucket
}

// SetRegion sets the region for testing
func (m *MockS3) SetRegion(region string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.region = region
}

// SetCDNBaseURL sets the CDN base URL for testing
func (m *MockS3) SetCDNBaseURL(url string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.cdnBaseURL = url
}

// GetAllKeys returns all keys in the mock storage
func (m *MockS3) GetAllKeys() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	keys := make([]string, 0, len(m.objects))
	for k := range m.objects {
		keys = append(keys, k)
	}
	
	return keys
}
