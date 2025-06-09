package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/antonrybalko/image-service-go/internal/domain"
	"gopkg.in/yaml.v3"
)

// LoadImageConfig loads image type configurations from a YAML file
func LoadImageConfig(configPath string) (*domain.ImageConfig, error) {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("image config file not found: %s", configPath)
	}

	// Read file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image config file: %w", err)
	}

	// Parse YAML
	var config domain.ImageConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse image config YAML: %w", err)
	}

	// Validate config
	if err := validateImageConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid image config: %w", err)
	}

	return &config, nil
}

// validateImageConfig checks that the image configuration is valid
func validateImageConfig(config *domain.ImageConfig) error {
	if config == nil {
		return errors.New("config is nil")
	}

	if len(config.Types) == 0 {
		return errors.New("no image types defined")
	}

	// Check each image type
	typeNames := make(map[string]bool)
	for i, imageType := range config.Types {
		// Check name
		if imageType.Name == "" {
			return fmt.Errorf("image type at index %d has no name", i)
		}

		// Check for duplicate names
		if typeNames[imageType.Name] {
			return fmt.Errorf("duplicate image type name: %s", imageType.Name)
		}
		typeNames[imageType.Name] = true

		// Check sizes
		if len(imageType.Sizes) == 0 {
			return fmt.Errorf("image type '%s' has no sizes defined", imageType.Name)
		}

		// Check each size
		for sizeName, size := range imageType.Sizes {
			if sizeName == "" {
				return fmt.Errorf("image type '%s' has a size with no name", imageType.Name)
			}

			// At least one dimension must be specified
			if size.Width <= 0 && size.Height <= 0 {
				return fmt.Errorf("image type '%s', size '%s' has invalid dimensions: width and height cannot both be zero or negative",
					imageType.Name, sizeName)
			}
		}

		// Check for required size names: small, medium, large
		requiredSizes := []string{"small", "medium", "large"}
		for _, required := range requiredSizes {
			if _, exists := imageType.Sizes[required]; !exists {
				return fmt.Errorf("image type '%s' is missing required size '%s'", imageType.Name, required)
			}
		}
	}

	return nil
}

// GetImageTypeByName returns the image type with the specified name
func GetImageTypeByName(config *domain.ImageConfig, name string) (*domain.ImageType, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}

	for _, imageType := range config.Types {
		if imageType.Name == name {
			return &imageType, nil
		}
	}

	return nil, fmt.Errorf("image type not found: %s", name)
}

// GetDefaultImageConfigPath returns the default path for the image config file
func GetDefaultImageConfigPath() string {
	// Check if config path is specified in environment
	configPath := os.Getenv("IMAGE_CONFIG_PATH")
	if configPath != "" {
		return configPath
	}

	// Default to config/images.yaml in the current directory
	return filepath.Join("config", "images.yaml")
}

// LoadDefaultImageConfig loads the image config from the default location
func LoadDefaultImageConfig() (*domain.ImageConfig, error) {
	configPath := GetDefaultImageConfigPath()
	return LoadImageConfig(configPath)
}
