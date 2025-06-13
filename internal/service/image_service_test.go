package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/antonrybalko/image-service-go/internal/domain"
	"github.com/antonrybalko/image-service-go/internal/processor"
	"github.com/antonrybalko/image-service-go/internal/repository"
	"github.com/antonrybalko/image-service-go/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestService creates a new ImageService with mock dependencies for testing
func setupTestService(t *testing.T) (
	*ImageService,
	*repository.MockImageRepository,
	*storage.MockS3,
	*processor.MockProcessor,
	*domain.ImageConfig,
) {
	// Create logger
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	// Create mock repository
	mockRepo := repository.NewMockImageRepository()

	// Create mock storage
	mockStorage := storage.NewMockS3()

	// Create mock processor
	mockProcessor := processor.NewMockProcessor()

	// Create image config
	imageConfig := &domain.ImageConfig{
		Types: []domain.ImageType{
			{
				Name: "user",
				Sizes: domain.SizeSet{
					"small":  {Width: 50, Height: 50},
					"medium": {Width: 100, Height: 100},
					"large":  {Width: 800, Height: 800},
				},
			},
		},
	}

	// Create image service
	service := NewImageService(mockRepo, mockStorage, mockProcessor, imageConfig, sugar)

	return service, mockRepo, mockStorage, mockProcessor, imageConfig
}

// createTestImage creates a test image for a user
func createTestImage(userGUID uuid.UUID) *domain.Image {
	imageGUID := uuid.New()
	now := time.Now().UTC()

	return &domain.Image{
		GUID:           imageGUID,
		OwnerGUID:      userGUID,
		TypeName:       "user",
		SmallURL:       "https://cdn.example.com/images/user/" + userGUID.String() + "/" + imageGUID.String() + "/small.jpg",
		MediumURL:      "https://cdn.example.com/images/user/" + userGUID.String() + "/" + imageGUID.String() + "/medium.jpg",
		LargeURL:       "https://cdn.example.com/images/user/" + userGUID.String() + "/" + imageGUID.String() + "/large.jpg",
		CreatedAt:      now,
		UpdatedAt:      now,
		ContentType:    "image/jpeg",
		OriginalWidth:  1200,
		OriginalHeight: 800,
	}
}

// createTestImageData creates mock image data for testing
func createTestImageData() []byte {
	return []byte("mock-image-data-for-testing")
}

// TestUploadUserImage tests uploading a user image
func TestUploadUserImage(t *testing.T) {
	// Set up test service and mocks
	service, mockRepo, mockStorage, mockProcessor, _ := setupTestService(t)

	// Create test data
	ctx := context.Background()
	userGUID := uuid.New()
	imageData := createTestImageData()

	// Configure mock processor
	mockProcessor.SetDetectedFormat(imageData, "image/jpeg")
	mockProcessor.SetImageDimensions(imageData, 1200, 800)

	// Test uploading an image
	userImage, err := service.UploadUserImage(ctx, userGUID, imageData)

	// Verify results
	require.NoError(t, err)
	assert.NotNil(t, userImage)
	assert.Equal(t, userGUID, userImage.UserGUID)
	assert.NotEqual(t, uuid.Nil, userImage.ImageGUID)
	assert.NotEmpty(t, userImage.SmallURL)
	assert.NotEmpty(t, userImage.MediumURL)
	assert.NotEmpty(t, userImage.LargeURL)
	assert.False(t, userImage.UpdatedAt.IsZero())

	// Verify repository was called to save the image
	assert.Equal(t, 1, mockRepo.GetImageCount())

	// Verify storage was called to upload the image variants
	assert.True(t, mockStorage.GetObjectCount() > 0)
}

// TestGetUserImage tests retrieving a user image
func TestGetUserImage(t *testing.T) {
	// Set up test service and mocks
	service, mockRepo, _, _, _ := setupTestService(t)

	// Create test data
	ctx := context.Background()
	userGUID := uuid.New()
	testImage := createTestImage(userGUID)

	// Save the test image to the mock repository
	err := mockRepo.SaveImage(ctx, testImage)
	require.NoError(t, err)

	// Test getting the user image
	userImage, err := service.GetUserImage(ctx, userGUID)

	// Verify results
	require.NoError(t, err)
	assert.NotNil(t, userImage)
	assert.Equal(t, userGUID, userImage.UserGUID)
	assert.Equal(t, testImage.GUID, userImage.ImageGUID)
	assert.Equal(t, testImage.SmallURL, userImage.SmallURL)
	assert.Equal(t, testImage.MediumURL, userImage.MediumURL)
	assert.Equal(t, testImage.LargeURL, userImage.LargeURL)
}

// TestGetUserImage_NotFound tests retrieving a non-existent user image
func TestGetUserImage_NotFound(t *testing.T) {
	// Set up test service and mocks
	service, _, _, _, _ := setupTestService(t)

	// Create test data
	ctx := context.Background()
	userGUID := uuid.New() // Random user with no image

	// Test getting a non-existent user image
	_, err := service.GetUserImage(ctx, userGUID)

	// Verify error
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

// TestDeleteUserImage tests deleting a user image
func TestDeleteUserImage(t *testing.T) {
	// Set up test service and mocks
	service, mockRepo, mockStorage, _, _ := setupTestService(t)

	// Create test data
	ctx := context.Background()
	userGUID := uuid.New()
	testImage := createTestImage(userGUID)

	// Save the test image to the mock repository
	err := mockRepo.SaveImage(ctx, testImage)
	require.NoError(t, err)

	// Test deleting the user image
	err = service.DeleteUserImage(ctx, userGUID)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, 0, mockRepo.GetImageCount())

	// Verify storage deletion was attempted
	// Note: In the mock, we don't actually store the objects first, so we can't verify deletion directly
}

// TestDeleteUserImage_NotFound tests deleting a non-existent user image
func TestDeleteUserImage_NotFound(t *testing.T) {
	// Set up test service and mocks
	service, _, _, _, _ := setupTestService(t)

	// Create test data
	ctx := context.Background()
	userGUID := uuid.New() // Random user with no image

	// Test deleting a non-existent user image
	err := service.DeleteUserImage(ctx, userGUID)

	// Verify error
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

// TestUploadUserImage_InvalidImage tests uploading an invalid image
func TestUploadUserImage_InvalidImage(t *testing.T) {
	// Set up test service and mocks
	service, _, _, _, _ := setupTestService(t)
	service, _, _, mockProcessor, _ := setupTestService(t)
	// Create test data
	ctx := context.Background()
	userGUID := uuid.New()
	emptyData := []byte{}

	// Test uploading an empty image
	_, err := service.UploadUserImage(ctx, userGUID, emptyData)

	// Verify error
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidImage))
}

// TestUploadUserImage_ImageTooLarge tests uploading an image that exceeds the size limit
func TestUploadUserImage_ImageTooLarge(t *testing.T) {
	// Set up test service and mocks
	service, _, _, _, _ := setupTestService(t)

	// Set a very small max size for testing
	service.SetMaxImageSize(10) // 10 bytes

	// Create test data
	ctx := context.Background()
	userGUID := uuid.New()
	largeData := make([]byte, 100) // 100 bytes, exceeds the 10 byte limit

	// Test uploading a large image
	_, err := service.UploadUserImage(ctx, userGUID, largeData)

	// Verify error
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrImageTooLarge))
}

// TestUploadUserImage_UnsupportedType tests uploading an image with an unsupported format
func TestUploadUserImage_UnsupportedType(t *testing.T) {
	// Set up test service and mocks
	service, _, _, mockProcessor, _ := setupTestService(t)

	// Create test data
	ctx := context.Background()
	userGUID := uuid.New()
	imageData := createTestImageData()

	// Configure mock processor to return an unsupported format
	mockProcessor.SetDetectedFormat(imageData, "image/tiff") // Not supported

	// Test uploading an unsupported image format
	_, err := service.UploadUserImage(ctx, userGUID, imageData)

	// Verify error
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedType))
}

// TestUploadUserImage_ProcessingFailed tests when image processing fails
func TestUploadUserImage_ProcessingFailed(t *testing.T) {
	// Set up test service and mocks
	service, _, _, mockProcessor, _ := setupTestService(t)

	// Create test data
	ctx := context.Background()
	userGUID := uuid.New()
	imageData := createTestImageData()

	// Configure mock processor
	mockProcessor.SetDetectedFormat(imageData, "image/jpeg")
	mockProcessor.SetShouldFailProcessing(true)

	// Test uploading with processing failure
	_, err := service.UploadUserImage(ctx, userGUID, imageData)

	// Verify error
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrProcessingFailed))
}

// TestGetUserImageByID tests retrieving a user image by ID
func TestGetUserImageByID(t *testing.T) {
	// Set up test service and mocks
	service, mockRepo, _, _, _ := setupTestService(t)

	// Create test data
	ctx := context.Background()
	userGUID := uuid.New()
	testImage := createTestImage(userGUID)

	// Save the test image to the mock repository
	err := mockRepo.SaveImage(ctx, testImage)
	require.NoError(t, err)

	// Test getting the user image by ID
	userImage, err := service.GetUserImageByID(ctx, testImage.GUID)

	// Verify results
	require.NoError(t, err)
	assert.NotNil(t, userImage)
	assert.Equal(t, userGUID, userImage.UserGUID)
	assert.Equal(t, testImage.GUID, userImage.ImageGUID)
	assert.Equal(t, testImage.SmallURL, userImage.SmallURL)
	assert.Equal(t, testImage.MediumURL, userImage.MediumURL)
	assert.Equal(t, testImage.LargeURL, userImage.LargeURL)
}

// TestGetUserImageByID_NotFound tests retrieving a non-existent image by ID
func TestGetUserImageByID_NotFound(t *testing.T) {
	// Set up test service and mocks
	service, _, _, _, _ := setupTestService(t)

	// Create test data
	ctx := context.Background()
	randomID := uuid.New() // Random ID that doesn't exist

	// Test getting a non-existent image
	_, err := service.GetUserImageByID(ctx, randomID)

	// Verify error
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

// TestValidateImageAccess tests validating image access
func TestValidateImageAccess(t *testing.T) {
	// Set up test service and mocks
	service, mockRepo, _, _, _ := setupTestService(t)

	// Create test data
	ctx := context.Background()
	userGUID := uuid.New()
	testImage := createTestImage(userGUID)

	// Save the test image to the mock repository
	err := mockRepo.SaveImage(ctx, testImage)
	require.NoError(t, err)

	// Test validating access for the owner
	err = service.ValidateImageAccess(ctx, userGUID, testImage.GUID)
	assert.NoError(t, err)

	// Test validating access for a different user
	otherUserGUID := uuid.New()
	err = service.ValidateImageAccess(ctx, otherUserGUID, testImage.GUID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}
