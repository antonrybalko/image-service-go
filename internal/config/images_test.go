package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/antonrybalko/image-service-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadImageConfig(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "image-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("ValidConfig", func(t *testing.T) {
		// Create a valid config file
		validYAML := `
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
		configPath := filepath.Join(tempDir, "valid-config.yaml")
		err := os.WriteFile(configPath, []byte(validYAML), 0644)
		require.NoError(t, err)

		cfg := &Config{
			ImageConfig: struct {
				ConfigPath string `mapstructure:"IMAGE_CONFIG_PATH"`
			}{
				ConfigPath: configPath,
			},
		}

		imageConfig, err := LoadImageConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, imageConfig)

		// Verify the loaded configuration
		assert.Equal(t, 2, len(imageConfig.Images))
		
		// Check user image type
		assert.Equal(t, "user", imageConfig.Images[0].Name)
		assert.Equal(t, 3, len(imageConfig.Images[0].Sizes))
		assert.Equal(t, 50, imageConfig.Images[0].Sizes["small"].Width)
		assert.Equal(t, 50, imageConfig.Images[0].Sizes["small"].Height)
		assert.Equal(t, 100, imageConfig.Images[0].Sizes["medium"].Width)
		assert.Equal(t, 800, imageConfig.Images[0].Sizes["large"].Width)
		
		// Check organization image type
		assert.Equal(t, "organization", imageConfig.Images[1].Name)
		assert.Equal(t, 3, len(imageConfig.Images[1].Sizes))
		assert.Equal(t, 400, imageConfig.Images[1].Sizes["small"].Width)
		assert.Equal(t, 0, imageConfig.Images[1].Sizes["small"].Height)
		assert.Equal(t, 1000, imageConfig.Images[1].Sizes["large"].Width)
	})

	t.Run("FileNotFound", func(t *testing.T) {
		cfg := &Config{
			ImageConfig: struct {
				ConfigPath string `mapstructure:"IMAGE_CONFIG_PATH"`
			}{
				ConfigPath: "non-existent-file.yaml",
			},
		}

		_, err := LoadImageConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		// Create an invalid YAML file
		invalidYAML := `
images:
  - name: user
    sizes:
      small:
        width: "not-a-number"
`
		configPath := filepath.Join(tempDir, "invalid-yaml.yaml")
		err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
		require.NoError(t, err)

		cfg := &Config{
			ImageConfig: struct {
				ConfigPath string `mapstructure:"IMAGE_CONFIG_PATH"`
			}{
				ConfigPath: configPath,
			},
		}

		_, err = LoadImageConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
	})

	t.Run("NilConfig", func(t *testing.T) {
		_, err := LoadImageConfig(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config cannot be nil")
	})

	t.Run("EmptyConfigPath", func(t *testing.T) {
		cfg := &Config{
			ImageConfig: struct {
				ConfigPath string `mapstructure:"IMAGE_CONFIG_PATH"`
			}{
				ConfigPath: "",
			},
		}

		_, err := LoadImageConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path is not set")
	})
}

func TestValidateImageConfig(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		config := &domain.ImageConfig{
			Images: []domain.ImageType{
				{
					Name: "user",
					Sizes: map[string]domain.Size{
						"small":  {Width: 50, Height: 50},
						"medium": {Width: 100, Height: 100},
						"large":  {Width: 800, Height: 800},
					},
				},
			},
		}

		err := validateImageConfig(config)
		assert.NoError(t, err)
	})

	t.Run("NilConfig", func(t *testing.T) {
		err := validateImageConfig(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config cannot be nil")
	})

	t.Run("EmptyImageTypes", func(t *testing.T) {
		config := &domain.ImageConfig{
			Images: []domain.ImageType{},
		}

		err := validateImageConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no image types defined")
	})

	t.Run("MissingName", func(t *testing.T) {
		config := &domain.ImageConfig{
			Images: []domain.ImageType{
				{
					Name: "",
					Sizes: map[string]domain.Size{
						"small":  {Width: 50, Height: 50},
						"medium": {Width: 100, Height: 100},
						"large":  {Width: 800, Height: 800},
					},
				},
			},
		}

		err := validateImageConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "has no name")
	})

	t.Run("NoSizes", func(t *testing.T) {
		config := &domain.ImageConfig{
			Images: []domain.ImageType{
				{
					Name:  "user",
					Sizes: map[string]domain.Size{},
				},
			},
		}

		err := validateImageConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "has no sizes defined")
	})

	t.Run("MissingRequiredSize", func(t *testing.T) {
		config := &domain.ImageConfig{
			Images: []domain.ImageType{
				{
					Name: "user",
					Sizes: map[string]domain.Size{
						"small":  {Width: 50, Height: 50},
						"medium": {Width: 100, Height: 100},
						// Missing "large" size
					},
				},
			},
		}

		err := validateImageConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required size")
	})

	t.Run("InvalidWidth", func(t *testing.T) {
		config := &domain.ImageConfig{
			Images: []domain.ImageType{
				{
					Name: "user",
					Sizes: map[string]domain.Size{
						"small":  {Width: -10, Height: 50},
						"medium": {Width: 100, Height: 100},
						"large":  {Width: 800, Height: 800},
					},
				},
			},
		}

		err := validateImageConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid width")
	})

	t.Run("InvalidHeight", func(t *testing.T) {
		config := &domain.ImageConfig{
			Images: []domain.ImageType{
				{
					Name: "user",
					Sizes: map[string]domain.Size{
						"small":  {Width: 50, Height: -10},
						"medium": {Width: 100, Height: 100},
						"large":  {Width: 800, Height: 800},
					},
				},
			},
		}

		err := validateImageConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid height")
	})

	t.Run("ZeroHeightIsValid", func(t *testing.T) {
		config := &domain.ImageConfig{
			Images: []domain.ImageType{
				{
					Name: "user",
					Sizes: map[string]domain.Size{
						"small":  {Width: 50, Height: 0},
						"medium": {Width: 100, Height: 0},
						"large":  {Width: 800, Height: 0},
					},
				},
			},
		}

		err := validateImageConfig(config)
		assert.NoError(t, err)
	})
}
