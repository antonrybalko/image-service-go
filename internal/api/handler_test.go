package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/antonrybalko/image-service-go/internal/auth"
	"github.com/antonrybalko/image-service-go/internal/domain"
	"github.com/antonrybalko/image-service-go/internal/processor"
	"github.com/antonrybalko/image-service-go/internal/repository"
	"github.com/antonrybalko/image-service-go/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUploadUserImage(t *testing.T) {
	// Create test dependencies
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()
	mockStorage := storage.NewMockS3Client("https://test-cdn.example.com")
	mockProcessor := processor.NewMockProcessor()
	mockRepo := repository.NewMockImageRepository()

	// Create handler with mocks
	handler := NewHandler(mockProcessor, mockStorage, mockRepo, sugar)

	// Setup test data
	userID := "test-user-123"
	testImage := []byte("fake-image-data")
	validContentType := "image/jpeg"

	// Setup mock processor to return different sized images
	smallImage := []byte("small-image")
	mediumImage := []byte("medium-image")
	largeImage := []byte("large-image")
	mockProcessor.SetProcessedImages("user", map[string][]byte{
		"small":  smallImage,
		"medium": mediumImage,
		"large":  largeImage,
	})

	t.Run("SuccessfulUpload", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		mockProcessor.SetProcessedImages("user", map[string][]byte{
			"small":  smallImage,
			"medium": mediumImage,
			"large":  largeImage,
		})

		// Create request
		req := httptest.NewRequest(http.MethodPut, "/v1/me/image", bytes.NewReader(testImage))
		req.Header.Set("Content-Type", validContentType)
		
		// Add user ID to context
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UploadUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusOK, rr.Code)
		
		// Verify response body
		var response domain.UserImageResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, userID, response.UserGUID)
		assert.NotEmpty(t, response.ImageGUID)
		assert.Contains(t, response.SmallURL, "small.jpg")
		assert.Contains(t, response.MediumURL, "medium.jpg")
		assert.Contains(t, response.LargeURL, "large.jpg")
		
		// Verify mock calls
		assert.True(t, mockRepo.HasUserImage(userID))
		processImageCalls, _, _ := mockProcessor.GetCallCounts()
		assert.Equal(t, 1, processImageCalls)
		uploads, _, _ := mockStorage.GetCallCounts()
		assert.Equal(t, 3, uploads) // One upload for each size
	})

	t.Run("InvalidContentType", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Create request with invalid content type
		req := httptest.NewRequest(http.MethodPut, "/v1/me/image", bytes.NewReader(testImage))
		req.Header.Set("Content-Type", "text/plain") // Invalid content type
		
		// Add user ID to context
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UploadUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		
		// Verify error message
		var errorResponse map[string]string
		err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
		require.NoError(t, err)
		assert.Contains(t, errorResponse["error"], "invalid content type")
		
		// Verify no storage or repository calls were made
		assert.False(t, mockRepo.HasUserImage(userID))
		uploads, _, _ := mockStorage.GetCallCounts()
		assert.Equal(t, 0, uploads)
	})

	t.Run("EmptyImage", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Create request with empty body
		req := httptest.NewRequest(http.MethodPut, "/v1/me/image", bytes.NewReader([]byte{}))
		req.Header.Set("Content-Type", validContentType)
		
		// Add user ID to context
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UploadUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		
		// Verify no storage or repository calls were made
		assert.False(t, mockRepo.HasUserImage(userID))
		uploads, _, _ := mockStorage.GetCallCounts()
		assert.Equal(t, 0, uploads)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Create request without user ID in context
		req := httptest.NewRequest(http.MethodPut, "/v1/me/image", bytes.NewReader(testImage))
		req.Header.Set("Content-Type", validContentType)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UploadUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		
		// Verify no storage or repository calls were made
		assert.False(t, mockRepo.HasUserImage(userID))
		uploads, _, _ := mockStorage.GetCallCounts()
		assert.Equal(t, 0, uploads)
	})

	t.Run("ProcessorError", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Configure processor to return an error
		mockProcessor.SetError(true, "processor test error")
		
		// Create request
		req := httptest.NewRequest(http.MethodPut, "/v1/me/image", bytes.NewReader(testImage))
		req.Header.Set("Content-Type", validContentType)
		
		// Add user ID to context
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UploadUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		
		// Verify no storage or repository calls were made
		assert.False(t, mockRepo.HasUserImage(userID))
		uploads, _, _ := mockStorage.GetCallCounts()
		assert.Equal(t, 0, uploads)
	})

	t.Run("StorageError", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		mockProcessor.SetProcessedImages("user", map[string][]byte{
			"small":  smallImage,
			"medium": mediumImage,
			"large":  largeImage,
		})
		
		// Configure storage to return an error
		mockStorage.SetError(true, "storage test error")
		
		// Create request
		req := httptest.NewRequest(http.MethodPut, "/v1/me/image", bytes.NewReader(testImage))
		req.Header.Set("Content-Type", validContentType)
		
		// Add user ID to context
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UploadUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		
		// Verify no repository calls were made
		assert.False(t, mockRepo.HasUserImage(userID))
	})

	t.Run("RepositoryError", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		mockProcessor.SetProcessedImages("user", map[string][]byte{
			"small":  smallImage,
			"medium": mediumImage,
			"large":  largeImage,
		})
		
		// Configure repository to return an error
		mockRepo.SetError(true, "repository test error")
		
		// Create request
		req := httptest.NewRequest(http.MethodPut, "/v1/me/image", bytes.NewReader(testImage))
		req.Header.Set("Content-Type", validContentType)
		
		// Add user ID to context
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UploadUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		
		// Verify storage calls were made but repository failed
		uploads, _, _ := mockStorage.GetCallCounts()
		assert.Equal(t, 3, uploads) // One upload for each size
		assert.False(t, mockRepo.HasUserImage(userID))
	})
}

func TestGetUserImage(t *testing.T) {
	// Create test dependencies
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()
	mockStorage := storage.NewMockS3Client("https://test-cdn.example.com")
	mockProcessor := processor.NewMockProcessor()
	mockRepo := repository.NewMockImageRepository()

	// Create handler with mocks
	handler := NewHandler(mockProcessor, mockStorage, mockRepo, sugar)

	// Setup test data
	userID := "test-user-123"
	imageID := uuid.New().String()
	now := time.Now()
	
	// Create a test image in the repository
	testImage := &domain.Image{
		GUID:      imageID,
		TypeID:    1,
		TypeName:  "user",
		OwnerGUID: userID,
		SmallURL:  "https://test-cdn.example.com/images/user/" + userID + "/" + imageID + "/small.jpg",
		MediumURL: "https://test-cdn.example.com/images/user/" + userID + "/" + imageID + "/medium.jpg",
		LargeURL:  "https://test-cdn.example.com/images/user/" + userID + "/" + imageID + "/large.jpg",
		CreatedAt: now,
		UpdatedAt: now,
	}

	t.Run("SuccessfulRetrieval", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Add image to repository
		mockRepo.AddUserImage(testImage)
		
		// Create request
		req := httptest.NewRequest(http.MethodGet, "/v1/me/image", nil)
		
		// Add user ID to context
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusOK, rr.Code)
		
		// Verify response body
		var response domain.UserImageResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, userID, response.UserGUID)
		assert.Equal(t, imageID, response.ImageGUID)
		assert.Equal(t, testImage.SmallURL, response.SmallURL)
		assert.Equal(t, testImage.MediumURL, response.MediumURL)
		assert.Equal(t, testImage.LargeURL, response.LargeURL)
	})

	t.Run("ImageNotFound", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Create request
		req := httptest.NewRequest(http.MethodGet, "/v1/me/image", nil)
		
		// Add user ID to context
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusNotFound, rr.Code)
		
		// Verify error message
		var errorResponse map[string]string
		err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
		require.NoError(t, err)
		assert.Contains(t, errorResponse["error"], "not found")
	})

	t.Run("Unauthorized", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Add image to repository
		mockRepo.AddUserImage(testImage)
		
		// Create request without user ID in context
		req := httptest.NewRequest(http.MethodGet, "/v1/me/image", nil)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("RepositoryError", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Configure repository to return an error
		mockRepo.SetError(true, "repository test error")
		
		// Create request
		req := httptest.NewRequest(http.MethodGet, "/v1/me/image", nil)
		
		// Add user ID to context
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestDeleteUserImage(t *testing.T) {
	// Create test dependencies
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()
	mockStorage := storage.NewMockS3Client("https://test-cdn.example.com")
	mockProcessor := processor.NewMockProcessor()
	mockRepo := repository.NewMockImageRepository()

	// Create handler with mocks
	handler := NewHandler(mockProcessor, mockStorage, mockRepo, sugar)

	// Setup test data
	userID := "test-user-123"
	imageID := uuid.New().String()
	now := time.Now()
	
	// Create a test image in the repository
	testImage := &domain.Image{
		GUID:      imageID,
		TypeID:    1,
		TypeName:  "user",
		OwnerGUID: userID,
		SmallURL:  "https://test-cdn.example.com/images/user/" + userID + "/" + imageID + "/small.jpg",
		MediumURL: "https://test-cdn.example.com/images/user/" + userID + "/" + imageID + "/medium.jpg",
		LargeURL:  "https://test-cdn.example.com/images/user/" + userID + "/" + imageID + "/large.jpg",
		CreatedAt: now,
		UpdatedAt: now,
	}

	t.Run("SuccessfulDeletion", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Add image to repository
		mockRepo.AddUserImage(testImage)
		
		// Create request
		req := httptest.NewRequest(http.MethodDelete, "/v1/me/image", nil)
		
		// Add user ID to context
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.DeleteUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusOK, rr.Code)
		
		// Verify image was deleted from repository
		assert.False(t, mockRepo.HasUserImage(userID))
		
		// Verify storage delete calls
		_, deletes, _ := mockStorage.GetCallCounts()
		assert.Equal(t, 3, deletes) // One delete for each size
	})

	t.Run("ImageNotFound", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Create request
		req := httptest.NewRequest(http.MethodDelete, "/v1/me/image", nil)
		
		// Add user ID to context
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.DeleteUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Add image to repository
		mockRepo.AddUserImage(testImage)
		
		// Create request without user ID in context
		req := httptest.NewRequest(http.MethodDelete, "/v1/me/image", nil)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.DeleteUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		
		// Verify image was not deleted
		assert.True(t, mockRepo.HasUserImage(userID))
	})

	t.Run("StorageError", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Add image to repository
		mockRepo.AddUserImage(testImage)
		
		// Configure storage to return an error
		mockStorage.SetError(true, "storage test error")
		
		// Create request
		req := httptest.NewRequest(http.MethodDelete, "/v1/me/image", nil)
		
		// Add user ID to context
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.DeleteUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		
		// Verify image was not deleted from repository
		assert.True(t, mockRepo.HasUserImage(userID))
	})

	t.Run("RepositoryError", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Add image to repository
		mockRepo.AddUserImage(testImage)
		
		// Configure repository delete to return an error
		mockRepo.SetError(true, "repository test error")
		
		// Create request
		req := httptest.NewRequest(http.MethodDelete, "/v1/me/image", nil)
		
		// Add user ID to context
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.DeleteUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestGetPublicUserImage(t *testing.T) {
	// Create test dependencies
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()
	mockStorage := storage.NewMockS3Client("https://test-cdn.example.com")
	mockProcessor := processor.NewMockProcessor()
	mockRepo := repository.NewMockImageRepository()

	// Create handler with mocks
	handler := NewHandler(mockProcessor, mockStorage, mockRepo, sugar)

	// Setup test data
	userID := "test-user-123"
	imageID := uuid.New().String()
	now := time.Now()
	
	// Create a test image in the repository
	testImage := &domain.Image{
		GUID:      imageID,
		TypeID:    1,
		TypeName:  "user",
		OwnerGUID: userID,
		SmallURL:  "https://test-cdn.example.com/images/user/" + userID + "/" + imageID + "/small.jpg",
		MediumURL: "https://test-cdn.example.com/images/user/" + userID + "/" + imageID + "/medium.jpg",
		LargeURL:  "https://test-cdn.example.com/images/user/" + userID + "/" + imageID + "/large.jpg",
		CreatedAt: now,
		UpdatedAt: now,
	}

	t.Run("SuccessfulRetrieval", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Add image to repository
		mockRepo.AddUserImage(testImage)
		
		// Create request with URL parameter
		req := httptest.NewRequest(http.MethodGet, "/v1/users/"+userID+"/image", nil)
		
		// Setup chi router context with URL parameters
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("userGuid", userID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetPublicUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusOK, rr.Code)
		
		// Verify response body
		var response domain.UserImageResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, userID, response.UserGUID)
		assert.Equal(t, imageID, response.ImageGUID)
		assert.Equal(t, testImage.SmallURL, response.SmallURL)
		assert.Equal(t, testImage.MediumURL, response.MediumURL)
		assert.Equal(t, testImage.LargeURL, response.LargeURL)
	})

	t.Run("ImageNotFound", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Create request with URL parameter
		req := httptest.NewRequest(http.MethodGet, "/v1/users/"+userID+"/image", nil)
		
		// Setup chi router context with URL parameters
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("userGuid", userID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetPublicUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("MissingUserID", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Create request without URL parameter
		req := httptest.NewRequest(http.MethodGet, "/v1/users/image", nil)
		
		// Setup chi router context without URL parameters
		rctx := chi.NewRouteContext()
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetPublicUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("RepositoryError", func(t *testing.T) {
		// Reset mocks
		mockStorage.Reset()
		mockProcessor.Reset()
		mockRepo.Reset()
		
		// Configure repository to return an error
		mockRepo.SetError(true, "repository test error")
		
		// Create request with URL parameter
		req := httptest.NewRequest(http.MethodGet, "/v1/users/"+userID+"/image", nil)
		
		// Setup chi router context with URL parameters
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("userGuid", userID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetPublicUserImage(rr, req)

		// Check response
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

// Helper function to read the entire response body
func readBody(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
