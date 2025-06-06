package processor

import (
	"context"
	"errors"
	"sync"

	"github.com/antonrybalko/image-service-go/internal/domain"
)

// MockProcessor implements the Processor interface for testing
type MockProcessor struct {
	mu                   sync.RWMutex
	supportedTypes       []string
	supportedContentTypes []string
	processedImages      map[string]map[string][]byte // imgType -> size -> data
	forceError           bool
	errorMessage         string
	processImageCalls    int
	getSupportedTypesCalls int
	getSupportedContentTypesCalls int
	lastProcessedType    string
	lastProcessedData    []byte
}

// NewMockProcessor creates a new mock processor with default supported types
func NewMockProcessor() *MockProcessor {
	return &MockProcessor{
		supportedTypes: []string{"user", "organization", "product"},
		supportedContentTypes: []string{"image/jpeg", "image/png"},
		processedImages: make(map[string]map[string][]byte),
	}
}

// ProcessImage returns mock processed images or an error if configured
func (m *MockProcessor) ProcessImage(ctx context.Context, imgType string, data []byte) (map[string][]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.processImageCalls++
	m.lastProcessedType = imgType
	m.lastProcessedData = make([]byte, len(data))
	copy(m.lastProcessedData, data)
	
	// Check for forced error
	if m.forceError {
		return nil, errors.New(m.errorMessage)
	}
	
	// Check if image type is supported
	supported := false
	for _, t := range m.supportedTypes {
		if t == imgType {
			supported = true
			break
		}
	}
	if !supported {
		return nil, ErrUnsupportedImageType
	}
	
	// Return configured processed images if available
	if images, exists := m.processedImages[imgType]; exists {
		return images, nil
	}
	
	// Default behavior: return a map with small, medium, large keys with the original data
	result := map[string][]byte{
		"small":  make([]byte, len(data)),
		"medium": make([]byte, len(data)),
		"large":  make([]byte, len(data)),
	}
	
	// Copy data to avoid external modifications
	copy(result["small"], data)
	copy(result["medium"], data)
	copy(result["large"], data)
	
	return result, nil
}

// GetSupportedTypes returns the configured supported types
func (m *MockProcessor) GetSupportedTypes() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.getSupportedTypesCalls++
	
	// Return a copy to avoid external modifications
	result := make([]string, len(m.supportedTypes))
	copy(result, m.supportedTypes)
	
	return result
}

// GetSupportedContentTypes returns the configured supported content types
func (m *MockProcessor) GetSupportedContentTypes() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.getSupportedContentTypesCalls++
	
	// Return a copy to avoid external modifications
	result := make([]string, len(m.supportedContentTypes))
	copy(result, m.supportedContentTypes)
	
	return result
}

// SetSupportedTypes configures the supported image types
func (m *MockProcessor) SetSupportedTypes(types []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.supportedTypes = make([]string, len(types))
	copy(m.supportedTypes, types)
}

// SetSupportedContentTypes configures the supported content types
func (m *MockProcessor) SetSupportedContentTypes(types []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.supportedContentTypes = make([]string, len(types))
	copy(m.supportedContentTypes, types)
}

// SetProcessedImages configures the processed images to return for a specific image type
func (m *MockProcessor) SetProcessedImages(imgType string, images map[string][]byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Create a deep copy of the images
	m.processedImages[imgType] = make(map[string][]byte)
	for size, data := range images {
		m.processedImages[imgType][size] = make([]byte, len(data))
		copy(m.processedImages[imgType][size], data)
	}
}

// SetError configures the mock to return an error on ProcessImage
func (m *MockProcessor) SetError(enable bool, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.forceError = enable
	if enable {
		m.errorMessage = message
	} else {
		m.errorMessage = ""
	}
}

// GetCallCounts returns the number of calls to each method
func (m *MockProcessor) GetCallCounts() (processImage, getSupportedTypes, getSupportedContentTypes int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.processImageCalls, m.getSupportedTypesCalls, m.getSupportedContentTypesCalls
}

// GetLastProcessed returns the last processed image type and data
func (m *MockProcessor) GetLastProcessed() (string, []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy of the data to avoid external modifications
	dataCopy := make([]byte, len(m.lastProcessedData))
	copy(dataCopy, m.lastProcessedData)
	
	return m.lastProcessedType, dataCopy
}

// Reset resets the mock state
func (m *MockProcessor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.processImageCalls = 0
	m.getSupportedTypesCalls = 0
	m.getSupportedContentTypesCalls = 0
	m.lastProcessedType = ""
	m.lastProcessedData = nil
	m.processedImages = make(map[string]map[string][]byte)
	m.forceError = false
	m.errorMessage = ""
}

// CreateProcessedImagesFromSizes creates a map of processed images based on domain.Size definitions
// This is useful for tests that need to match the real processor's behavior
func (m *MockProcessor) CreateProcessedImagesFromSizes(data []byte, sizes map[string]domain.Size) map[string][]byte {
	result := make(map[string][]byte)
	
	for sizeName := range sizes {
		// In a real implementation, this would resize the image
		// For the mock, we just copy the original data
		result[sizeName] = make([]byte, len(data))
		copy(result[sizeName], data)
	}
	
	return result
}
