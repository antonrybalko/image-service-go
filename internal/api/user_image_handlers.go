package api

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/antonrybalko/image-service-go/internal/auth"
	"github.com/antonrybalko/image-service-go/internal/domain"
	"github.com/antonrybalko/image-service-go/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// UserImageResponse represents the response format for user image endpoints
type UserImageResponse struct {
	UserGUID  uuid.UUID `json:"userGuid"`
	ImageGUID uuid.UUID `json:"imageGuid"`
	SmallURL  string    `json:"smallUrl"`
	MediumURL string    `json:"mediumUrl"`
	LargeURL  string    `json:"largeUrl"`
	UpdatedAt string    `json:"updatedAt"`
}

// UserImageHandlers contains handlers for user image endpoints
type UserImageHandlers struct {
	imageService *service.ImageService
}

// NewUserImageHandlers creates a new set of user image handlers
func NewUserImageHandlers(imageService *service.ImageService) *UserImageHandlers {
	return &UserImageHandlers{
		imageService: imageService,
	}
}

// UploadUserImage handles PUT /v1/me/image
func (h *UserImageHandlers) UploadUserImage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context (set by JWT middleware)
		userIDStr, ok := auth.GetUserIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "Unauthorized", "Invalid or missing authentication")
			return
		}

		// Parse user ID
		userGUID, err := uuid.Parse(userIDStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "InvalidUserID", "User ID is not a valid UUID")
			return
		}

		// Check content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "image/jpeg" && contentType != "image/png" {
			writeError(w, http.StatusBadRequest, "InvalidContentType", "Only JPEG and PNG images are supported")
			return
		}

		// Read image data
		imageData, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "ReadError", "Failed to read image data")
			return
		}
		defer r.Body.Close()

		// Check if image data is empty
		if len(imageData) == 0 {
			writeError(w, http.StatusBadRequest, "EmptyImage", "Image data is empty")
			return
		}

		// Process and store the image
		userImage, err := h.imageService.UploadUserImage(r.Context(), userGUID, imageData)
		if err != nil {
			handleImageServiceError(w, err)
			return
		}

		// Prepare response
		response := UserImageResponse{
			UserGUID:  userImage.UserGUID,
			ImageGUID: userImage.ImageGUID,
			SmallURL:  userImage.SmallURL,
			MediumURL: userImage.MediumURL,
			LargeURL:  userImage.LargeURL,
			UpdatedAt: userImage.UpdatedAt.Format(http.TimeFormat),
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// GetCurrentUserImage handles GET /v1/me/image
func (h *UserImageHandlers) GetCurrentUserImage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context (set by JWT middleware)
		userIDStr, ok := auth.GetUserIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "Unauthorized", "Invalid or missing authentication")
			return
		}

		// Parse user ID
		userGUID, err := uuid.Parse(userIDStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "InvalidUserID", "User ID is not a valid UUID")
			return
		}

		// Get the user's image
		userImage, err := h.imageService.GetUserImage(r.Context(), userGUID)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeError(w, http.StatusNotFound, "ImageNotFound", "User has no image")
				return
			}
			writeError(w, http.StatusInternalServerError, "ServiceError", "Failed to retrieve user image")
			return
		}

		// Prepare response
		response := UserImageResponse{
			UserGUID:  userImage.UserGUID,
			ImageGUID: userImage.ImageGUID,
			SmallURL:  userImage.SmallURL,
			MediumURL: userImage.MediumURL,
			LargeURL:  userImage.LargeURL,
			UpdatedAt: userImage.UpdatedAt.Format(http.TimeFormat),
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// DeleteUserImage handles DELETE /v1/me/image
func (h *UserImageHandlers) DeleteUserImage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context (set by JWT middleware)
		userIDStr, ok := auth.GetUserIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "Unauthorized", "Invalid or missing authentication")
			return
		}

		// Parse user ID
		userGUID, err := uuid.Parse(userIDStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "InvalidUserID", "User ID is not a valid UUID")
			return
		}

		// Delete the user's image
		err = h.imageService.DeleteUserImage(r.Context(), userGUID)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeError(w, http.StatusNotFound, "ImageNotFound", "User has no image to delete")
				return
			}
			writeError(w, http.StatusInternalServerError, "ServiceError", "Failed to delete user image")
			return
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Image deleted successfully",
		})
	}
}

// GetUserImage handles GET /v1/users/{userGuid}/image
func (h *UserImageHandlers) GetUserImage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user GUID from URL path
		userGuidStr := chi.URLParam(r, "userGuid")
		if userGuidStr == "" {
			writeError(w, http.StatusBadRequest, "BadRequest", "User GUID is required")
			return
		}

		// Parse user GUID
		userGUID, err := uuid.Parse(userGuidStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "InvalidUserID", "User ID is not a valid UUID")
			return
		}

		// Get the user's image
		userImage, err := h.imageService.GetUserImage(r.Context(), userGUID)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeError(w, http.StatusNotFound, "ImageNotFound", "User has no image")
				return
			}
			writeError(w, http.StatusInternalServerError, "ServiceError", "Failed to retrieve user image")
			return
		}

		// Prepare response
		response := UserImageResponse{
			UserGUID:  userImage.UserGUID,
			ImageGUID: userImage.ImageGUID,
			SmallURL:  userImage.SmallURL,
			MediumURL: userImage.MediumURL,
			LargeURL:  userImage.LargeURL,
			UpdatedAt: userImage.UpdatedAt.Format(http.TimeFormat),
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// Helper functions

// writeError writes a standardized error response
func writeError(w http.ResponseWriter, status int, errType, message string) {
	resp := ErrorResponse{
		Error:   errType,
		Message: message,
		Code:    status,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// handleImageServiceError maps service errors to HTTP responses
func handleImageServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidImage):
		writeError(w, http.StatusBadRequest, "InvalidImage", "Invalid image data")
	case errors.Is(err, service.ErrImageTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "ImageTooLarge", "Image exceeds maximum allowed size")
	case errors.Is(err, service.ErrUnsupportedType):
		writeError(w, http.StatusUnsupportedMediaType, "UnsupportedType", "Unsupported image format")
	case errors.Is(err, service.ErrProcessingFailed):
		writeError(w, http.StatusUnprocessableEntity, "ProcessingFailed", "Failed to process image")
	case errors.Is(err, service.ErrStorageFailed):
		writeError(w, http.StatusInternalServerError, "StorageFailed", "Failed to store image")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "NotFound", "Image not found")
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, http.StatusForbidden, "Forbidden", "Unauthorized access to image")
	default:
		writeError(w, http.StatusInternalServerError, "InternalError", "An unexpected error occurred")
	}
}
