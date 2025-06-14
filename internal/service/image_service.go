package service

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/antonrybalko/image-service-go/internal/domain"
	"github.com/antonrybalko/image-service-go/internal/processor"
	"github.com/antonrybalko/image-service-go/internal/repository"
	"github.com/antonrybalko/image-service-go/internal/storage"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Common service errors
var (
	ErrInvalidImage     = errors.New("invalid image data")
	ErrImageTooLarge    = errors.New("image too large")
	ErrUnsupportedType  = errors.New("unsupported image type")
	ErrProcessingFailed = errors.New("image processing failed")
	ErrStorageFailed    = errors.New("image storage failed")
	ErrNotFound         = errors.New("image not found")
	ErrUnauthorized     = errors.New("unauthorized access to image")
)

// ImageService handles image processing, storage, and metadata management
type ImageService struct {
	repo      repository.ImageRepository
	storage   storage.S3Interface
	processor processor.ProcessorInterface
	config    *domain.ImageConfig
	logger    *zap.SugaredLogger
	maxSize   int64 // Maximum image size in bytes
}

// NewImageService creates a new image service
func NewImageService(
	repo repository.ImageRepository,
	storage storage.S3Interface,
	processor processor.ProcessorInterface,
	config *domain.ImageConfig,
	logger *zap.SugaredLogger,
) *ImageService {
	return &ImageService{
		repo:      repo,
		storage:   storage,
		processor: processor,
		config:    config,
		logger:    logger,
		maxSize:   15 * 1024 * 1024, // Default 15MB max size
	}
}

// SetMaxImageSize sets the maximum allowed image size in bytes
func (s *ImageService) SetMaxImageSize(maxBytes int64) {
	s.maxSize = maxBytes
}

// UploadUserImage processes and stores a user image
func (s *ImageService) UploadUserImage(ctx context.Context, userGUID uuid.UUID, imageData []byte) (*domain.UserImage, error) {
	// Validate image data
	if len(imageData) == 0 {
		return nil, ErrInvalidImage
	}

	// Check size limit
	if int64(len(imageData)) > s.maxSize {
		return nil, ErrImageTooLarge
	}

	// Detect image format
	contentType, err := s.processor.DetectImageFormat(imageData)
	if err != nil {
		s.logger.Errorw("Failed to detect image format",
			"error", err,
			"userGUID", userGUID)
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedType, err)
	}

	// Only allow JPEG and PNG
	if contentType != "image/jpeg" && contentType != "image/png" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, contentType)
	}

	// Get image dimensions
	width, height, err := s.processor.GetImageDimensions(imageData)
	if err != nil {
		s.logger.Errorw("Failed to get image dimensions",
			"error", err,
			"userGUID", userGUID)
		return nil, fmt.Errorf("%w: %v", ErrProcessingFailed, err)
	}

	// Get user image type configuration
	imageType, found := domain.GetImageTypeByName(s.config, "user")
	if !found {
		s.logger.Errorw("Failed to get image type configuration",
			"userGUID", userGUID)
		return nil, fmt.Errorf("image type configuration not found")
	}

	// Process image to create variants
	variants, err := s.processor.ProcessImage(imageData, imageType)
	if err != nil {
		s.logger.Errorw("Failed to process image",
			"error", err,
			"userGUID", userGUID)
		return nil, fmt.Errorf("%w: %v", ErrProcessingFailed, err)
	}

	// Generate a new image GUID
	imageGUID := uuid.New()

	// Delete any existing image for this user
	err = s.DeleteUserImage(ctx, userGUID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		s.logger.Warnw("Failed to delete existing user image",
			"error", err,
			"userGUID", userGUID)
		// Continue with upload even if deletion fails
	}

	// Create a new image record
	image := domain.NewImage(userGUID, "user")
	image.GUID = imageGUID
	image.OriginalWidth = width
	image.OriginalHeight = height
	image.ContentType = contentType

	// Upload each variant to storage
	for size, variantData := range variants {
		// Generate S3 key for this variant
		key := s.storage.GenerateUserImageKey(userGUID, imageGUID, size)

		// Upload to S3
		url, err := s.storage.Put(ctx, key, variantData, "image/jpeg")
		if err != nil {
			s.logger.Errorw("Failed to upload image variant",
				"error", err,
				"userGUID", userGUID,
				"imageGUID", imageGUID,
				"size", size)
			return nil, fmt.Errorf("%w: %v", ErrStorageFailed, err)
		}

		// Set URL in image record
		switch size {
		case "small":
			image.SmallURL = url
		case "medium":
			image.MediumURL = url
		case "large":
			image.LargeURL = url
		}
	}

	// Save image metadata to repository
	err = s.repo.SaveImage(ctx, image)
	if err != nil {
		s.logger.Errorw("Failed to save image metadata",
			"error", err,
			"userGUID", userGUID,
			"imageGUID", imageGUID)
		return nil, fmt.Errorf("failed to save image metadata: %w", err)
	}

	// Return user image view
	return image.ToUserImage(), nil
}

// GetUserImage retrieves a user's image by user GUID
func (s *ImageService) GetUserImage(ctx context.Context, userGUID uuid.UUID) (*domain.UserImage, error) {
	// Get image from repository
	image, err := s.repo.GetImageByOwner(ctx, userGUID, "user")
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		s.logger.Errorw("Failed to get user image",
			"error", err,
			"userGUID", userGUID)
		return nil, fmt.Errorf("failed to get user image: %w", err)
	}

	// Return user image view
	return image.ToUserImage(), nil
}

// GetUserImageByID retrieves a user's image by image GUID
func (s *ImageService) GetUserImageByID(ctx context.Context, imageGUID uuid.UUID) (*domain.UserImage, error) {
	// Get image from repository
	image, err := s.repo.GetImageByID(ctx, imageGUID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		s.logger.Errorw("Failed to get image by ID",
			"error", err,
			"imageGUID", imageGUID)
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	// Verify it's a user image
	if image.TypeName != "user" {
		return nil, fmt.Errorf("%w: not a user image", ErrUnauthorized)
	}

	// Return user image view
	return image.ToUserImage(), nil
}

// DeleteUserImage deletes a user's image
func (s *ImageService) DeleteUserImage(ctx context.Context, userGUID uuid.UUID) error {
	// Get the image first to get its GUID
	image, err := s.repo.GetImageByOwner(ctx, userGUID, "user")
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		s.logger.Errorw("Failed to get user image for deletion",
			"error", err,
			"userGUID", userGUID)
		return fmt.Errorf("failed to get user image for deletion: %w", err)
	}

	// Delete image variants from storage
	sizes := []string{"small", "medium", "large"}
	for _, size := range sizes {
		key := s.storage.GenerateUserImageKey(userGUID, image.GUID, size)
		err := s.storage.Delete(ctx, key)
		if err != nil {
			s.logger.Warnw("Failed to delete image variant from storage",
				"error", err,
				"userGUID", userGUID,
				"imageGUID", image.GUID,
				"size", size)
			// Continue with deletion even if one variant fails
		}
	}

	// Delete image metadata from repository
	err = s.repo.DeleteImage(ctx, image.GUID)
	if err != nil {
		s.logger.Errorw("Failed to delete image metadata",
			"error", err,
			"userGUID", userGUID,
			"imageGUID", image.GUID)
		return fmt.Errorf("failed to delete image metadata: %w", err)
	}

	return nil
}

// ValidateImageAccess checks if a user has access to an image
func (s *ImageService) ValidateImageAccess(ctx context.Context, userGUID uuid.UUID, imageGUID uuid.UUID) error {
	// Get the image
	image, err := s.repo.GetImageByID(ctx, imageGUID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to get image: %w", err)
	}

	// Check if the user owns the image
	if image.OwnerGUID != userGUID {
		return ErrUnauthorized
	}

	return nil
}

// ReadImageFromFile is a helper function to read image data from a file
func ReadImageFromFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}
