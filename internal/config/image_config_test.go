package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/antonrybalko/image-service-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadImageConfig tests loading a valid image configuration
func TestLoadImageConfig(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "image-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a valid image config file
	configPath := filepath.Join(tempDir, "valid-config.yaml")
	configContent := `
images:
  - name: user
    sizes:
      small:
        width: 50
        height: 50
      medium:
        width: 100
        height: 100
      large:
        width: 800
        height: 800
  - name: organization
    sizes:
      small:
        width: 400
        height: 0
      medium:
        width: 800
        height: 0
      large:
        width: 1000
        height: 0
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the configuration
	config, err := LoadImageConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Verify the configuration
	assert.Len(t, config.Types, 2)

	// Verify user image type
	userType := config.Types[0]
	assert.Equal(t, "user", userType.Name)
	assert.Len(t, userType.Sizes, 3)
	assert.Equal(t, 50, userType.Sizes["small"].Width)
	assert.Equal(t, 50, userType.Sizes["small"].Height)
	assert.Equal(t, 100, userType.Sizes["medium"].Width)
	assert.Equal(t, 100, userType.Sizes["medium"].Height)
	assert.Equal(t, 800, userType.Sizes["large"].Width)
	assert.Equal(t, 800, userType.Sizes["large"].Height)

	// Verify organization image type
	orgType := config.Types[1]
	assert.Equal(t, "organization", orgType.Name)
	assert.Len(t, orgType.Sizes, 3)
	assert.Equal(t, 400, orgType.Sizes["small"].Width)
	assert.Equal(t, 0, orgType.Sizes["small"].Height)
	assert.Equal(t, 800, orgType.Sizes["medium"].Width)
	assert.Equal(t, 0, orgType.Sizes["medium"].Height)
	assert.Equal(t, 1000, orgType.Sizes["large"].Width)
	assert.Equal(t, 0, orgType.Sizes["large"].Height)
}

// TestLoadImageConfig_FileNotFound tests loading a non-existent configuration file
func TestLoadImageConfig_FileNotFound(t *testing.T) {
	_, err := LoadImageConfig("non-existent-file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestLoadImageConfig_InvalidYAML tests loading an invalid YAML file
func TestLoadImageConfig_InvalidYAML(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "image-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create an invalid YAML file
	configPath := filepath.Join(tempDir, "invalid-yaml.yaml")
	configContent := `
images:
  - name: user
    sizes:
      small:
        width: 50
        height: "not-a-number" # Invalid YAML - height should be a number
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Try to load the configuration
	_, err = LoadImageConfig(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

// TestValidateImageConfig tests the validation of image configurations
func TestValidateImageConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *domain.ImageConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "config is nil",
		},
		{
			name: "Empty config",
			config: &domain.ImageConfig{
				Types: []domain.ImageType{},
			},
			expectError: true,
			errorMsg:    "no image types defined",
		},
		{
			name: "Missing name",
			config: &domain.ImageConfig{
				Types: []domain.ImageType{
					{
						Name: "",
						Sizes: domain.SizeSet{
							"small":  {Width: 50, Height: 50},
							"medium": {Width: 100, Height: 100},
							"large":  {Width: 800, Height: 800},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "has no name",
		},
		{
			name: "Duplicate names",
			config: &domain.ImageConfig{
				Types: []domain.ImageType{
					{
						Name: "user",
						Sizes: domain.SizeSet{
							"small":  {Width: 50, Height: 50},
							"medium": {Width: 100, Height: 100},
							"large":  {Width: 800, Height: 800},
						},
					},
					{
						Name: "user", // Duplicate name
						Sizes: domain.SizeSet{
							"small":  {Width: 50, Height: 50},
							"medium": {Width: 100, Height: 100},
							"large":  {Width: 800, Height: 800},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "duplicate image type name",
		},
		{
			name: "No sizes",
			config: &domain.ImageConfig{
				Types: []domain.ImageType{
					{
						Name:  "user",
						Sizes: domain.SizeSet{},
					},
				},
			},
			expectError: true,
			errorMsg:    "has no sizes defined",
		},
		{
			name: "Invalid dimensions",
			config: &domain.ImageConfig{
				Types: []domain.ImageType{
					{
						Name: "user",
						Sizes: domain.SizeSet{
							"small":  {Width: 0, Height: 0}, // Invalid dimensions
							"medium": {Width: 100, Height: 100},
							"large":  {Width: 800, Height: 800},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid dimensions",
		},
		{
			name: "Missing required size",
			config: &domain.ImageConfig{
				Types: []domain.ImageType{
					{
						Name: "user",
						Sizes: domain.SizeSet{
							"small":  {Width: 50, Height: 50},
							"medium": {Width: 100, Height: 100},
							// Missing "large" size
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "missing required size",
		},
		{
			name: "Valid config",
			config: &domain.ImageConfig{
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
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateImageConfig(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetImageTypeByName tests finding an image type by name
func TestGetImageTypeByName(t *testing.T) {
	// Create a test config
	config := &domain.ImageConfig{
		Types: []domain.ImageType{
			{
				Name: "user",
				Sizes: domain.SizeSet{
					"small":  {Width: 50, Height: 50},
					"medium": {Width: 100, Height: 100},
					"large":  {Width: 800, Height: 800},
				},
			},
			{
				Name: "organization",
				Sizes: domain.SizeSet{
					"small":  {Width: 400, Height: 0},
					"medium": {Width: 800, Height: 0},
					"large":  {Width: 1000, Height: 0},
				},
			},
		},
	}

	// Test finding an existing image type
	t.Run("Existing type", func(t *testing.T) {
		imageType, err := GetImageTypeByName(config, "user")
		assert.NoError(t, err)
		assert.NotNil(t, imageType)
		assert.Equal(t, "user", imageType.Name)
		assert.Equal(t, 50, imageType.Sizes["small"].Width)
	})

	// Test finding a non-existent image type
	t.Run("Non-existent type", func(t *testing.T) {
		imageType, err := GetImageTypeByName(config, "product")
		assert.Error(t, err)
		assert.Nil(t, imageType)
		assert.Contains(t, err.Error(), "not found")
	})

	// Test with nil config
	t.Run("Nil config", func(t *testing.T) {
		imageType, err := GetImageTypeByName(nil, "user")
		assert.Error(t, err)
		assert.Nil(t, imageType)
		assert.Contains(t, err.Error(), "config is nil")
	})
}

// TestGetDefaultImageConfigPath tests getting the default image config path
func TestGetDefaultImageConfigPath(t *testing.T) {
	// Save original environment
	origEnv := os.Getenv("IMAGE_CONFIG_PATH")
	defer os.Setenv("IMAGE_CONFIG_PATH", origEnv)

	// Test with environment variable set
	os.Setenv("IMAGE_CONFIG_PATH", "/custom/path/config.yaml")
	path := GetDefaultImageConfigPath()
	assert.Equal(t, "/custom/path/config.yaml", path)

	// Test with environment variable unset
	os.Unsetenv("IMAGE_CONFIG_PATH")
	path = GetDefaultImageConfigPath()
	assert.Equal(t, filepath.Join("config", "images.yaml"), path)
}
