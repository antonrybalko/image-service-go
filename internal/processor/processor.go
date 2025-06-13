package processor

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"sync"

	"github.com/antonrybalko/image-service-go/internal/domain"
	"golang.org/x/image/draw"
)

// ProcessorInterface defines the operations for image processing
type ProcessorInterface interface {
	// ProcessImage processes an image according to the image type configuration
	// and returns a map of size name to processed image bytes
	ProcessImage(imgData []byte, imageType *domain.ImageType) (map[string][]byte, error)
	
	// DetectImageFormat detects the image format and returns the content type
	DetectImageFormat(imgData []byte) (string, error)
	
	// GetImageDimensions returns the width and height of an image
	GetImageDimensions(imgData []byte) (width int, height int, err error)
	
	// CalculateResizeDimensions calculates new dimensions preserving aspect ratio
	CalculateResizeDimensions(origWidth, origHeight, targetWidth, targetHeight int) (newWidth, newHeight int)
}

// Processor implements ProcessorInterface using Go's standard image package
// In a real implementation, this would use govips/libvips for better performance
type Processor struct {
	// No fields needed for the basic implementation
}

// NewProcessor creates a new image processor
func NewProcessor() ProcessorInterface {
	return &Processor{}
}

// ProcessImage processes an image according to the image type configuration
func (p *Processor) ProcessImage(imgData []byte, imageType *domain.ImageType) (map[string][]byte, error) {
	if len(imgData) == 0 {
		return nil, errors.New("empty image data")
	}
	
	if imageType == nil {
		return nil, errors.New("image type configuration is required")
	}
	
	// Decode the source image
	srcImg, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	
	// Get original dimensions
	bounds := srcImg.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()
	
	result := make(map[string][]byte)
	
	// Process each size variant
	for sizeName, size := range imageType.Sizes {
		// Calculate new dimensions preserving aspect ratio
		newWidth, newHeight := p.CalculateResizeDimensions(origWidth, origHeight, size.Width, size.Height)
		
		// Create a new image with the calculated dimensions
		dstImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		
		// Resize the image using CatmullRom for high-quality resampling
		draw.CatmullRom.Scale(dstImg, dstImg.Bounds(), srcImg, srcImg.Bounds(), draw.Over, nil)
		
		// Encode the resized image
		var buf bytes.Buffer
		var encodeErr error
		
		// Always encode as JPEG for consistency, regardless of input format
		encodeErr = jpeg.Encode(&buf, dstImg, &jpeg.Options{Quality: 90})
		
		if encodeErr != nil {
			return nil, fmt.Errorf("failed to encode %s image: %w", sizeName, encodeErr)
		}
		
		result[sizeName] = buf.Bytes()
	}
	
	return result, nil
}

// DetectImageFormat detects the image format and returns the content type
func (p *Processor) DetectImageFormat(imgData []byte) (string, error) {
	if len(imgData) < 12 {
		return "", errors.New("image data too small to determine format")
	}
	
	// Check for JPEG signature
	if bytes.Equal(imgData[0:2], []byte{0xFF, 0xD8}) {
		return "image/jpeg", nil
	}
	
	// Check for PNG signature
	if bytes.Equal(imgData[0:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return "image/png", nil
	}
	
	// Check for GIF signature
	if bytes.Equal(imgData[0:6], []byte{'G', 'I', 'F', '8', '7', 'a'}) || 
	   bytes.Equal(imgData[0:6], []byte{'G', 'I', 'F', '8', '9', 'a'}) {
		return "image/gif", nil
	}
	
	// Check for WebP signature
	if bytes.Equal(imgData[0:4], []byte{'R', 'I', 'F', 'F'}) && 
	   bytes.Equal(imgData[8:12], []byte{'W', 'E', 'B', 'P'}) {
		return "image/webp", nil
	}
	
	return "", errors.New("unsupported image format")
}

// GetImageDimensions returns the width and height of an image
func (p *Processor) GetImageDimensions(imgData []byte) (width int, height int, err error) {
	reader := bytes.NewReader(imgData)
	cfg, _, err := image.DecodeConfig(reader)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image dimensions: %w", err)
	}
	
	return cfg.Width, cfg.Height, nil
}

// CalculateResizeDimensions calculates new dimensions preserving aspect ratio
func (p *Processor) CalculateResizeDimensions(origWidth, origHeight, targetWidth, targetHeight int) (newWidth, newHeight int) {
	// If both target dimensions are specified
	if targetWidth > 0 && targetHeight > 0 {
		return targetWidth, targetHeight
	}
	
	// If only target width is specified, calculate height to maintain aspect ratio
	if targetWidth > 0 && targetHeight == 0 {
		aspectRatio := float64(origHeight) / float64(origWidth)
		newHeight = int(float64(targetWidth) * aspectRatio)
		return targetWidth, newHeight
	}
	
	// If only target height is specified, calculate width to maintain aspect ratio
	if targetHeight > 0 && targetWidth == 0 {
		aspectRatio := float64(origWidth) / float64(origHeight)
		newWidth = int(float64(targetHeight) * aspectRatio)
		return newWidth, targetHeight
	}
	
	// If neither is specified, return original dimensions
	return origWidth, origHeight
}

// MockProcessor implements ProcessorInterface for testing
type MockProcessor struct {
	mutex               sync.RWMutex
	processedImages     map[string]map[string][]byte
	detectedFormats     map[string]string
	imageDimensions     map[string]struct{ width, height int }
	shouldFailProcessing bool
	shouldFailDetection  bool
}

// NewMockProcessor creates a new mock processor for testing
func NewMockProcessor() *MockProcessor {
	return &MockProcessor{
		processedImages: make(map[string]map[string][]byte),
		detectedFormats: make(map[string]string),
		imageDimensions: make(map[string]struct{ width, height int }),
	}
}

// ProcessImage mocks processing an image
func (m *MockProcessor) ProcessImage(imgData []byte, imageType *domain.ImageType) (map[string][]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.shouldFailProcessing {
		return nil, errors.New("mock processing failure")
	}
	
	// Generate a unique key for this image data
	key := fmt.Sprintf("%x", imgData[:16]) // Use first 16 bytes as key
	
	// Create mock processed images for each size
	result := make(map[string][]byte)
	for sizeName := range imageType.Sizes {
		// Mock image data for this size
		mockData := []byte(fmt.Sprintf("mock-%s-%s-data", key, sizeName))
		result[sizeName] = mockData
	}
	
	// Store the result for later verification
	m.processedImages[key] = result
	
	return result, nil
}

// DetectImageFormat mocks detecting the image format
func (m *MockProcessor) DetectImageFormat(imgData []byte) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if m.shouldFailDetection {
		return "", errors.New("mock detection failure")
	}
	
	// Generate a unique key for this image data
	key := fmt.Sprintf("%x", imgData[:16]) // Use first 16 bytes as key
	
	// Return predefined format or default to JPEG
	if format, exists := m.detectedFormats[key]; exists {
		return format, nil
	}
	
	return "image/jpeg", nil
}

// GetImageDimensions mocks getting image dimensions
func (m *MockProcessor) GetImageDimensions(imgData []byte) (width int, height int, err error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Generate a unique key for this image data
	key := fmt.Sprintf("%x", imgData[:16]) // Use first 16 bytes as key
	
	// Return predefined dimensions or default
	if dims, exists := m.imageDimensions[key]; exists {
		return dims.width, dims.height, nil
	}
	
	// Default dimensions
	return 800, 600, nil
}

// CalculateResizeDimensions calculates new dimensions preserving aspect ratio
func (m *MockProcessor) CalculateResizeDimensions(origWidth, origHeight, targetWidth, targetHeight int) (newWidth, newHeight int) {
	// Use the same logic as the real processor
	if targetWidth > 0 && targetHeight > 0 {
		return targetWidth, targetHeight
	}
	
	if targetWidth > 0 && targetHeight == 0 {
		aspectRatio := float64(origHeight) / float64(origWidth)
		newHeight = int(float64(targetWidth) * aspectRatio)
		return targetWidth, newHeight
	}
	
	if targetHeight > 0 && targetWidth == 0 {
		aspectRatio := float64(origWidth) / float64(origHeight)
		newWidth = int(float64(targetHeight) * aspectRatio)
		return newWidth, targetHeight
	}
	
	return origWidth, origHeight
}

// --- Test Helper Methods ---

// SetShouldFailProcessing configures the mock to fail processing
func (m *MockProcessor) SetShouldFailProcessing(shouldFail bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.shouldFailProcessing = shouldFail
}

// SetShouldFailDetection configures the mock to fail format detection
func (m *MockProcessor) SetShouldFailDetection(shouldFail bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.shouldFailDetection = shouldFail
}

// SetDetectedFormat sets a predefined format for an image
func (m *MockProcessor) SetDetectedFormat(imgData []byte, format string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	key := fmt.Sprintf("%x", imgData[:16])
	m.detectedFormats[key] = format
}

// SetImageDimensions sets predefined dimensions for an image
func (m *MockProcessor) SetImageDimensions(imgData []byte, width, height int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	key := fmt.Sprintf("%x", imgData[:16])
	m.imageDimensions[key] = struct{ width, height int }{width, height}
}

// GetProcessedImageCount returns the number of processed images
func (m *MockProcessor) GetProcessedImageCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.processedImages)
}

// ClearProcessedImages clears all processed images
func (m *MockProcessor) ClearProcessedImages() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.processedImages = make(map[string]map[string][]byte)
}
