package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/antonrybalko/image-service-go/internal/auth"
	"github.com/antonrybalko/image-service-go/internal/domain"
	"github.com/antonrybalko/image-service-go/internal/processor"
	"github.com/antonrybalko/image-service-go/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrInvalidContentType = errors.New("invalid content type")
	ErrNoImageProvided    = errors.New("no image provided")
	ErrImageTooLarge      = errors.New("image too large")
	ErrImageNotFound      = errors.New("image not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidImageFormat = errors.New("invalid image format")
	ErrProcessingFailed   = errors.New("image processing failed")
	ErrStorageFailed      = errors.New("storage operation failed")
)

const (
	// MaxImageSize is the maximum allowed size for uploaded images (15MB)
	MaxImageSize = 15 * 1024 * 1024
)

// Handler defines the interface for the API handler
type Handler interface {
	// User image operations
	UploadUserImage(w http.ResponseWriter, r *http.Request)
	GetUserImage(w http.ResponseWriter, r *http.Request)
	DeleteUserImage(w http.ResponseWriter, r *http.Request)
	
	// Organization image operations
	UploadOrganizationImage(w http.ResponseWriter, r *http.Request)
	GetOrganizationImage(w http.ResponseWriter, r *http.Request)
	DeleteOrganizationImage(w http.ResponseWriter, r *http.Request)
	
	// Public endpoints
	GetPublicUserImage(w http.ResponseWriter, r *http.Request)
	GetPublicOrganizationImage(w http.ResponseWriter, r *http.Request)
	
	// Register routes
	RegisterRoutes(r chi.Router)
}

// ImageProcessor defines the interface for image processing operations
type ImageProcessor interface {
	ProcessImage(ctx context.Context, imgType string, data []byte) (map[string][]byte, error)
	GetSupportedTypes() []string
	GetSupportedContentTypes() []string
}

// ImageStorage defines the interface for image storage operations
type ImageStorage interface {
	UploadImage(ctx context.Context, key string, data []byte, contentType string) (string, error)
	DeleteImage(ctx context.Context, key string) error
	GetImageURL(key string) string
}

// ImageRepository defines the interface for image metadata persistence
type ImageRepository interface {
	SaveUserImage(ctx context.Context, userID, imageID string, smallURL, mediumURL, largeURL string) (*domain.Image, error)
	GetUserImage(ctx context.Context, userID string) (*domain.Image, error)
	DeleteUserImage(ctx context.Context, userID string) error
	
	SaveOrganizationImage(ctx context.Context, orgID, imageID string, smallURL, mediumURL, largeURL string) (*domain.Image, error)
	GetOrganizationImage(ctx context.Context, orgID string) (*domain.Image, error)
	DeleteOrganizationImage(ctx context.Context, orgID string) error
	
	GetPublicUserImage(ctx context.Context, userID string) (*domain.Image, error)
	GetPublicOrganizationImage(ctx context.Context, orgID string) (*domain.Image, error)
}

// handlerImpl implements the Handler interface
type handlerImpl struct {
	processor  ImageProcessor
	storage    ImageStorage
	repository ImageRepository
	logger     *zap.SugaredLogger
}

// NewHandler creates a new API handler
func NewHandler(
	processor ImageProcessor,
	storage ImageStorage,
	repository ImageRepository,
	logger *zap.SugaredLogger,
) Handler {
	return &handlerImpl{
		processor:  processor,
		storage:    storage,
		repository: repository,
		logger:     logger,
	}
}

// RegisterRoutes registers all API routes
func (h *handlerImpl) RegisterRoutes(r chi.Router) {
	// Private routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth)
		
		// User image routes
		r.Put("/v1/me/image", h.UploadUserImage)
		r.Get("/v1/me/image", h.GetUserImage)
		r.Delete("/v1/me/image", h.DeleteUserImage)
		
		// Organization image routes
		r.Put("/v1/me/organizations/{orgGuid}/image", h.UploadOrganizationImage)
		r.Get("/v1/me/organizations/{orgGuid}/image", h.GetOrganizationImage)
		r.Delete("/v1/me/organizations/{orgGuid}/image", h.DeleteOrganizationImage)
	})
	
	// Public routes
	r.Get("/v1/users/{userGuid}/image", h.GetPublicUserImage)
	r.Get("/v1/organizations/{orgGuid}/image", h.GetPublicOrganizationImage)
}

// UploadUserImage handles user image uploads
func (h *handlerImpl) UploadUserImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from context
	userID, ok := auth.GetUserID(ctx)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}
	
	// Validate content type
	contentType := r.Header.Get("Content-Type")
	if !h.isValidContentType(contentType) {
		h.respondWithError(w, http.StatusBadRequest, ErrInvalidContentType)
		return
	}
	
	// Read image data with size limit
	imageData, err := h.readImageData(r)
	if err != nil {
		if errors.Is(err, ErrImageTooLarge) {
			h.respondWithError(w, http.StatusRequestEntityTooLarge, err)
		} else {
			h.respondWithError(w, http.StatusBadRequest, err)
		}
		return
	}
	
	// Generate a new image ID
	imageID := uuid.New().String()
	
	// Process the image (resize to different sizes)
	processedImages, err := h.processor.ProcessImage(ctx, "user", imageData)
	if err != nil {
		h.logger.Errorw("Failed to process image", "userID", userID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, ErrProcessingFailed)
		return
	}
	
	// Upload images to storage
	smallURL, mediumURL, largeURL, err := h.uploadProcessedImages(ctx, "user", userID, imageID, processedImages)
	if err != nil {
		h.logger.Errorw("Failed to upload processed images", "userID", userID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, ErrStorageFailed)
		return
	}
	
	// Save image metadata
	image, err := h.repository.SaveUserImage(ctx, userID, imageID, smallURL, mediumURL, largeURL)
	if err != nil {
		h.logger.Errorw("Failed to save image metadata", "userID", userID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, errors.New("failed to save image metadata"))
		return
	}
	
	// Return response
	h.respondWithJSON(w, http.StatusOK, image.ToUserImageResponse())
}

// GetUserImage handles retrieving the current user's image
func (h *handlerImpl) GetUserImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from context
	userID, ok := auth.GetUserID(ctx)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}
	
	// Get image metadata
	image, err := h.repository.GetUserImage(ctx, userID)
	if err != nil {
		h.logger.Errorw("Failed to get user image", "userID", userID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, errors.New("failed to get image metadata"))
		return
	}
	
	if image == nil {
		h.respondWithError(w, http.StatusNotFound, ErrImageNotFound)
		return
	}
	
	// Return response
	h.respondWithJSON(w, http.StatusOK, image.ToUserImageResponse())
}

// DeleteUserImage handles deleting the current user's image
func (h *handlerImpl) DeleteUserImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from context
	userID, ok := auth.GetUserID(ctx)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}
	
	// Get image metadata (to get the image ID for deletion)
	image, err := h.repository.GetUserImage(ctx, userID)
	if err != nil {
		h.logger.Errorw("Failed to get user image for deletion", "userID", userID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, errors.New("failed to get image metadata"))
		return
	}
	
	if image == nil {
		h.respondWithError(w, http.StatusNotFound, ErrImageNotFound)
		return
	}
	
	// Delete image files from storage
	if err := h.deleteImageFiles(ctx, "user", userID, image.GUID); err != nil {
		h.logger.Errorw("Failed to delete image files", "userID", userID, "imageID", image.GUID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, ErrStorageFailed)
		return
	}
	
	// Delete image metadata
	if err := h.repository.DeleteUserImage(ctx, userID); err != nil {
		h.logger.Errorw("Failed to delete image metadata", "userID", userID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, errors.New("failed to delete image metadata"))
		return
	}
	
	// Return success response
	h.respondWithJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GetPublicUserImage handles retrieving a user's image by user ID (public endpoint)
func (h *handlerImpl) GetPublicUserImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from URL parameter
	userID := chi.URLParam(r, "userGuid")
	if userID == "" {
		h.respondWithError(w, http.StatusBadRequest, errors.New("user ID is required"))
		return
	}
	
	// Get image metadata
	image, err := h.repository.GetPublicUserImage(ctx, userID)
	if err != nil {
		h.logger.Errorw("Failed to get public user image", "userID", userID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, errors.New("failed to get image metadata"))
		return
	}
	
	if image == nil {
		h.respondWithError(w, http.StatusNotFound, ErrImageNotFound)
		return
	}
	
	// Return response
	h.respondWithJSON(w, http.StatusOK, image.ToUserImageResponse())
}

// UploadOrganizationImage handles organization image uploads
func (h *handlerImpl) UploadOrganizationImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from context
	userID, ok := auth.GetUserID(ctx)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}
	
	// Get organization ID from URL parameter
	orgID := chi.URLParam(r, "orgGuid")
	if orgID == "" {
		h.respondWithError(w, http.StatusBadRequest, errors.New("organization ID is required"))
		return
	}
	
	// TODO: In future iterations, validate that the user has permission to modify this organization
	
	// Validate content type
	contentType := r.Header.Get("Content-Type")
	if !h.isValidContentType(contentType) {
		h.respondWithError(w, http.StatusBadRequest, ErrInvalidContentType)
		return
	}
	
	// Read image data with size limit
	imageData, err := h.readImageData(r)
	if err != nil {
		if errors.Is(err, ErrImageTooLarge) {
			h.respondWithError(w, http.StatusRequestEntityTooLarge, err)
		} else {
			h.respondWithError(w, http.StatusBadRequest, err)
		}
		return
	}
	
	// Generate a new image ID
	imageID := uuid.New().String()
	
	// Process the image (resize to different sizes)
	processedImages, err := h.processor.ProcessImage(ctx, "organization", imageData)
	if err != nil {
		h.logger.Errorw("Failed to process organization image", "orgID", orgID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, ErrProcessingFailed)
		return
	}
	
	// Upload images to storage
	smallURL, mediumURL, largeURL, err := h.uploadProcessedImages(ctx, "organization", orgID, imageID, processedImages)
	if err != nil {
		h.logger.Errorw("Failed to upload processed organization images", "orgID", orgID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, ErrStorageFailed)
		return
	}
	
	// Save image metadata
	image, err := h.repository.SaveOrganizationImage(ctx, orgID, imageID, smallURL, mediumURL, largeURL)
	if err != nil {
		h.logger.Errorw("Failed to save organization image metadata", "orgID", orgID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, errors.New("failed to save image metadata"))
		return
	}
	
	// Return response
	h.respondWithJSON(w, http.StatusOK, image.ToOrganizationImageResponse())
}

// GetOrganizationImage handles retrieving an organization's image
func (h *handlerImpl) GetOrganizationImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from context
	userID, ok := auth.GetUserID(ctx)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}
	
	// Get organization ID from URL parameter
	orgID := chi.URLParam(r, "orgGuid")
	if orgID == "" {
		h.respondWithError(w, http.StatusBadRequest, errors.New("organization ID is required"))
		return
	}
	
	// TODO: In future iterations, validate that the user has permission to access this organization
	
	// Get image metadata
	image, err := h.repository.GetOrganizationImage(ctx, orgID)
	if err != nil {
		h.logger.Errorw("Failed to get organization image", "orgID", orgID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, errors.New("failed to get image metadata"))
		return
	}
	
	if image == nil {
		h.respondWithError(w, http.StatusNotFound, ErrImageNotFound)
		return
	}
	
	// Return response
	h.respondWithJSON(w, http.StatusOK, image.ToOrganizationImageResponse())
}

// DeleteOrganizationImage handles deleting an organization's image
func (h *handlerImpl) DeleteOrganizationImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from context
	userID, ok := auth.GetUserID(ctx)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}
	
	// Get organization ID from URL parameter
	orgID := chi.URLParam(r, "orgGuid")
	if orgID == "" {
		h.respondWithError(w, http.StatusBadRequest, errors.New("organization ID is required"))
		return
	}
	
	// TODO: In future iterations, validate that the user has permission to modify this organization
	
	// Get image metadata (to get the image ID for deletion)
	image, err := h.repository.GetOrganizationImage(ctx, orgID)
	if err != nil {
		h.logger.Errorw("Failed to get organization image for deletion", "orgID", orgID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, errors.New("failed to get image metadata"))
		return
	}
	
	if image == nil {
		h.respondWithError(w, http.StatusNotFound, ErrImageNotFound)
		return
	}
	
	// Delete image files from storage
	if err := h.deleteImageFiles(ctx, "organization", orgID, image.GUID); err != nil {
		h.logger.Errorw("Failed to delete organization image files", "orgID", orgID, "imageID", image.GUID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, ErrStorageFailed)
		return
	}
	
	// Delete image metadata
	if err := h.repository.DeleteOrganizationImage(ctx, orgID); err != nil {
		h.logger.Errorw("Failed to delete organization image metadata", "orgID", orgID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, errors.New("failed to delete image metadata"))
		return
	}
	
	// Return success response
	h.respondWithJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GetPublicOrganizationImage handles retrieving an organization's image by ID (public endpoint)
func (h *handlerImpl) GetPublicOrganizationImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get organization ID from URL parameter
	orgID := chi.URLParam(r, "orgGuid")
	if orgID == "" {
		h.respondWithError(w, http.StatusBadRequest, errors.New("organization ID is required"))
		return
	}
	
	// Get image metadata
	image, err := h.repository.GetPublicOrganizationImage(ctx, orgID)
	if err != nil {
		h.logger.Errorw("Failed to get public organization image", "orgID", orgID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, errors.New("failed to get image metadata"))
		return
	}
	
	if image == nil {
		h.respondWithError(w, http.StatusNotFound, ErrImageNotFound)
		return
	}
	
	// Return response
	h.respondWithJSON(w, http.StatusOK, image.ToOrganizationImageResponse())
}

// Helper methods

// readImageData reads image data from the request body with a size limit
func (h *handlerImpl) readImageData(r *http.Request) ([]byte, error) {
	// Limit the size of the request body
	r.Body = http.MaxBytesReader(nil, r.Body, MaxImageSize)
	
	// Read the image data
	imageData, err := io.ReadAll(r.Body)
	if err != nil {
		if err.Error() == "http: request body too large" {
			return nil, ErrImageTooLarge
		}
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}
	
	if len(imageData) == 0 {
		return nil, ErrNoImageProvided
	}
	
	return imageData, nil
}

// isValidContentType checks if the content type is valid for image uploads
func (h *handlerImpl) isValidContentType(contentType string) bool {
	validTypes := h.processor.GetSupportedContentTypes()
	for _, t := range validTypes {
		if t == contentType {
			return true
		}
	}
	return false
}

// uploadProcessedImages uploads processed images to storage and returns the URLs
func (h *handlerImpl) uploadProcessedImages(
	ctx context.Context,
	imageType string,
	ownerID string,
	imageID string,
	processedImages map[string][]byte,
) (string, string, string, error) {
	// Upload small image
	smallKey := storage.BuildImageKey(imageType, ownerID, imageID, "small")
	smallURL, err := h.storage.UploadImage(ctx, smallKey, processedImages["small"], "image/jpeg")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to upload small image: %w", err)
	}
	
	// Upload medium image
	mediumKey := storage.BuildImageKey(imageType, ownerID, imageID, "medium")
	mediumURL, err := h.storage.UploadImage(ctx, mediumKey, processedImages["medium"], "image/jpeg")
	if err != nil {
		// Try to clean up the small image
		_ = h.storage.DeleteImage(ctx, smallKey)
		return "", "", "", fmt.Errorf("failed to upload medium image: %w", err)
	}
	
	// Upload large image
	largeKey := storage.BuildImageKey(imageType, ownerID, imageID, "large")
	largeURL, err := h.storage.UploadImage(ctx, largeKey, processedImages["large"], "image/jpeg")
	if err != nil {
		// Try to clean up the other images
		_ = h.storage.DeleteImage(ctx, smallKey)
		_ = h.storage.DeleteImage(ctx, mediumKey)
		return "", "", "", fmt.Errorf("failed to upload large image: %w", err)
	}
	
	return smallURL, mediumURL, largeURL, nil
}

// deleteImageFiles deletes all image files for a given image
func (h *handlerImpl) deleteImageFiles(ctx context.Context, imageType, ownerID, imageID string) error {
	// Delete small image
	smallKey := storage.BuildImageKey(imageType, ownerID, imageID, "small")
	if err := h.storage.DeleteImage(ctx, smallKey); err != nil {
		return fmt.Errorf("failed to delete small image: %w", err)
	}
	
	// Delete medium image
	mediumKey := storage.BuildImageKey(imageType, ownerID, imageID, "medium")
	if err := h.storage.DeleteImage(ctx, mediumKey); err != nil {
		return fmt.Errorf("failed to delete medium image: %w", err)
	}
	
	// Delete large image
	largeKey := storage.BuildImageKey(imageType, ownerID, imageID, "large")
	if err := h.storage.DeleteImage(ctx, largeKey); err != nil {
		return fmt.Errorf("failed to delete large image: %w", err)
	}
	
	return nil
}

// respondWithError sends an error response
func (h *handlerImpl) respondWithError(w http.ResponseWriter, code int, err error) {
	h.respondWithJSON(w, code, map[string]string{"error": err.Error()})
}

// respondWithJSON sends a JSON response
func (h *handlerImpl) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	// Set content type and status code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	// Encode the response
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		h.logger.Errorw("Failed to encode JSON response", "error", err)
		// If we can't encode the JSON, send a plain text error
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
