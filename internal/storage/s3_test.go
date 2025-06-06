package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockS3Client(t *testing.T) {
	baseURL := "https://test-cdn.example.com"
	mock := NewMockS3Client(baseURL)

	t.Run("UploadImage", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		key := "test/image.jpg"
		data := []byte("fake image data")
		contentType := "image/jpeg"

		// Test upload
		url, err := mock.UploadImage(ctx, key, data, contentType)
		require.NoError(t, err)
		assert.Equal(t, baseURL+"/"+key, url)

		// Verify image was stored
		storedData, storedContentType, exists := mock.GetImage(key)
		assert.True(t, exists)
		assert.Equal(t, data, storedData)
		assert.Equal(t, contentType, storedContentType)

		// Check call counts
		uploads, _, _ := mock.GetCallCounts()
		assert.Equal(t, 1, uploads)
	})

	t.Run("DeleteImage", func(t *testing.T) {
		// Setup - upload an image first
		ctx := context.Background()
		key := "test/delete-me.jpg"
		data := []byte("to be deleted")
		contentType := "image/jpeg"

		_, err := mock.UploadImage(ctx, key, data, contentType)
		require.NoError(t, err)
		assert.True(t, mock.HasImage(key))

		// Test delete
		err = mock.DeleteImage(ctx, key)
		require.NoError(t, err)
		assert.False(t, mock.HasImage(key))

		// Check call counts
		_, deletes, _ := mock.GetCallCounts()
		assert.Equal(t, 1, deletes)

		// Test delete non-existent image
		err = mock.DeleteImage(ctx, "non-existent.jpg")
		assert.Error(t, err)
	})

	t.Run("GetImageURL", func(t *testing.T) {
		key := "test/url-test.jpg"
		url := mock.GetImageURL(key)
		assert.Equal(t, baseURL+"/"+key, url)

		// Check call counts
		_, _, getURLs := mock.GetCallCounts()
		assert.Equal(t, 1, getURLs)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		ctx := context.Background()
		key := "test/error.jpg"
		data := []byte("error test")
		errorMsg := "forced error for testing"

		// Set error mode
		mock.SetError(true, errorMsg)

		// Test upload with error
		_, err := mock.UploadImage(ctx, key, data, "image/jpeg")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), errorMsg)
		assert.False(t, mock.HasImage(key))

		// Test delete with error
		err = mock.DeleteImage(ctx, "any-key.jpg")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), errorMsg)

		// Disable error mode
		mock.SetError(false, "")

		// Should work now
		_, err = mock.UploadImage(ctx, key, data, "image/jpeg")
		assert.NoError(t, err)
	})

	t.Run("Reset", func(t *testing.T) {
		// Setup - upload some images
		ctx := context.Background()
		for i := 0; i < 3; i++ {
			key := "test/reset-test" + string(rune('A'+i)) + ".jpg"
			_, err := mock.UploadImage(ctx, key, []byte("test"), "image/jpeg")
			require.NoError(t, err)
		}

		// Verify images exist
		assert.Equal(t, 3, mock.ImageCount())

		// Reset and verify
		mock.Reset()
		assert.Equal(t, 0, mock.ImageCount())

		// Check call counters were reset
		uploads, deletes, getURLs := mock.GetCallCounts()
		assert.Equal(t, 0, uploads)
		assert.Equal(t, 0, deletes)
		assert.Equal(t, 0, getURLs)
	})
}

func TestURLGeneration(t *testing.T) {
	t.Run("StandardS3URL", func(t *testing.T) {
		// Create client with standard S3 URL generation
		client := &S3Client{
			bucket: "test-bucket",
			region: "us-west-2",
			// No CDN base URL
		}

		key := "images/user/123/456/small.jpg"
		url := client.GetImageURL(key)
		expected := "https://test-bucket.s3.us-west-2.amazonaws.com/images/user/123/456/small.jpg"
		assert.Equal(t, expected, url)
	})

	t.Run("CDNBaseURL", func(t *testing.T) {
		// Create client with CDN base URL
		client := &S3Client{
			bucket:     "test-bucket",
			region:     "us-west-2",
			cdnBaseURL: "https://cdn.example.com",
		}

		key := "images/user/123/456/small.jpg"
		url := client.GetImageURL(key)
		expected := "https://cdn.example.com/images/user/123/456/small.jpg"
		assert.Equal(t, expected, url)
	})

	t.Run("CDNBaseURLWithTrailingSlash", func(t *testing.T) {
		// Create client with CDN base URL that has a trailing slash
		client := &S3Client{
			bucket:     "test-bucket",
			region:     "us-west-2",
			cdnBaseURL: "https://cdn.example.com/",
		}

		key := "images/user/123/456/small.jpg"
		url := client.GetImageURL(key)
		expected := "https://cdn.example.com/images/user/123/456/small.jpg"
		assert.Equal(t, expected, url)
	})

	t.Run("KeyWithLeadingSlash", func(t *testing.T) {
		client := &S3Client{
			bucket:     "test-bucket",
			region:     "us-west-2",
			cdnBaseURL: "https://cdn.example.com",
		}

		// Key with leading slash should have it trimmed
		key := "/images/user/123/456/small.jpg"
		url := client.GetImageURL(key)
		expected := "https://cdn.example.com/images/user/123/456/small.jpg"
		assert.Equal(t, expected, url)
	})
}

func TestBuildImageKey(t *testing.T) {
	testCases := []struct {
		name       string
		imageType  string
		ownerGUID  string
		imageGUID  string
		size       string
		expectedKey string
	}{
		{
			name:       "UserImage",
			imageType:  "user",
			ownerGUID:  "user-123",
			imageGUID:  "image-456",
			size:       "small",
			expectedKey: "images/user/user-123/image-456/small.jpg",
		},
		{
			name:       "OrganizationImage",
			imageType:  "organization",
			ownerGUID:  "org-789",
			imageGUID:  "image-abc",
			size:       "large",
			expectedKey: "images/organization/org-789/image-abc/large.jpg",
		},
		{
			name:       "ProductImage",
			imageType:  "product",
			ownerGUID:  "prod-xyz",
			imageGUID:  "image-def",
			size:       "medium",
			expectedKey: "images/product/prod-xyz/image-def/medium.jpg",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := BuildImageKey(tc.imageType, tc.ownerGUID, tc.imageGUID, tc.size)
			assert.Equal(t, tc.expectedKey, key)
		})
	}
}
