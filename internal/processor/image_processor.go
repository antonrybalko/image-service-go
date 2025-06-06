package processor

import (
	"context"
	"errors"
	"fmt"

	"github.com/antonrybalko/image-service-go/internal/domain"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrUnsupportedImageType     = errors.New("unsupported image type")
	ErrInvalidImageData         = errors.New("invalid image data")
	ErrProcessingFailed         = errors.New("image processing failed")
	ErrUnsupportedContentType   = errors.New("unsupported content type")
)

// Processor defines the interface for image processing operations
type Processor interface {
	// ProcessImage processes an image of the given type and returns a map of processed images
	// The map keys are the size names (small, medium, large) and the values are the processed image data
	ProcessImage(ctx context.Context, imgType string, data []byte) (map[string][]byte, error)
	
	// GetSupportedTypes returns a list of supported image types (user, organization, product)
	GetSupportedTypes() []string
	
	// GetSupportedContentTypes returns a list of supported content types (image/jpeg, image/png)
	GetSupportedContentTypes() []string
}

// ImageProcessor implements the Processor interface
type ImageProcessor struct {
	config  map[string]domain.ImageType
	logger  *zap.SugaredLogger
}

// New creates a new ImageProcessor
func New(imageConfig *domain.ImageConfig, logger *zap.SugaredLogger) *ImageProcessor {
	// Create a map for faster lookups
	configMap := make(map[string]domain.ImageType)
	for _, imgType := range imageConfig.Images {
		configMap[imgType.Name] = imgType
	}
	
	return &ImageProcessor{
		config: configMap,
		logger: logger,
	}
}

// ProcessImage processes an image according to the configured sizes for the given type
// This is a placeholder implementation that will be replaced with govips later
func (p *ImageProcessor) ProcessImage(ctx context.Context, imgType string, data []byte) (map[string][]byte, error) {
	// Validate image type
	imageType, exists := p.config[imgType]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedImageType, imgType)
	}
	
	// Validate image data
	if len(data) == 0 {
		return nil, ErrInvalidImageData
	}
	
	// TODO: Replace with actual image processing using govips
	// This is just a placeholder that returns the original data for each size
	
	p.logger.Debugw("Processing image", 
		"type", imgType, 
		"dataSize", len(data),
		"sizes", getSizeNames(imageType.Sizes),
	)
	
	result := make(map[string][]byte)
	
	// For each configured size, create a "processed" image
	// In the real implementation, this would resize the image using govips
	for sizeName := range imageType.Sizes {
		// Just copy the data for now - this will be replaced with actual resizing
		result[sizeName] = make([]byte, len(data))
		copy(result[sizeName], data)
	}
	
	return result, nil
}

// GetSupportedTypes returns the list of supported image types
func (p *ImageProcessor) GetSupportedTypes() []string {
	types := make([]string, 0, len(p.config))
	for typeName := range p.config {
		types = append(types, typeName)
	}
	return types
}

// GetSupportedContentTypes returns the list of supported content types
func (p *ImageProcessor) GetSupportedContentTypes() []string {
	// For now, support JPEG and PNG
	return []string{"image/jpeg", "image/png"}
}

// Helper functions

// getSizeNames returns a slice of size names for logging
func getSizeNames(sizes map[string]domain.Size) []string {
	names := make([]string, 0, len(sizes))
	for name := range sizes {
		names = append(names, name)
	}
	return names
}

// ValidateContentType checks if a content type is supported
func (p *ImageProcessor) ValidateContentType(contentType string) bool {
	for _, supported := range p.GetSupportedContentTypes() {
		if contentType == supported {
			return true
		}
	}
	return false
}

// GetImageType returns the image type configuration for the given type name
func (p *ImageProcessor) GetImageType(typeName string) (domain.ImageType, bool) {
	imageType, exists := p.config[typeName]
	return imageType, exists
}
