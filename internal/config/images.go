package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/antonrybalko/image-service-go/internal/domain"
	"gopkg.in/yaml.v3"
)

// LoadImageConfig loads the image configuration from the YAML file
// specified in the Config struct and returns a domain.ImageConfig
func LoadImageConfig(cfg *Config) (*domain.ImageConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	configPath := cfg.ImageConfig.ConfigPath
	if configPath == "" {
		return nil, fmt.Errorf("image config path is not set")
	}

	// Check if the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try to resolve relative to current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		
		configPath = filepath.Join(cwd, configPath)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("image config file not found at %s", cfg.ImageConfig.ConfigPath)
		}
	}

	// Read the file
	yamlData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image config file: %w", err)
	}

	// Parse the YAML
	var imageConfig domain.ImageConfig
	if err := yaml.Unmarshal(yamlData, &imageConfig); err != nil {
		return nil, fmt.Errorf("failed to parse image config YAML: %w", err)
	}

	// Validate the configuration
	if err := validateImageConfig(&imageConfig); err != nil {
		return nil, err
	}

	return &imageConfig, nil
}

// validateImageConfig performs validation on the loaded image configuration
func validateImageConfig(cfg *domain.ImageConfig) error {
	if cfg == nil {
		return fmt.Errorf("image config cannot be nil")
	}

	if len(cfg.Images) == 0 {
		return fmt.Errorf("no image types defined in configuration")
	}

	for i, imgType := range cfg.Images {
		if imgType.Name == "" {
			return fmt.Errorf("image type at index %d has no name", i)
		}

		if len(imgType.Sizes) == 0 {
			return fmt.Errorf("image type '%s' has no sizes defined", imgType.Name)
		}

		// Check for required size variants
		requiredSizes := []string{"small", "medium", "large"}
		for _, size := range requiredSizes {
			if _, exists := imgType.Sizes[size]; !exists {
				return fmt.Errorf("image type '%s' is missing required size '%s'", imgType.Name, size)
			}
		}

		// Validate each size
		for sizeName, size := range imgType.Sizes {
			if size.Width <= 0 {
				return fmt.Errorf("image type '%s', size '%s' has invalid width: %d", imgType.Name, sizeName, size.Width)
			}
			
			// Height can be 0 (auto-scale) but not negative
			if size.Height < 0 {
				return fmt.Errorf("image type '%s', size '%s' has invalid height: %d", imgType.Name, sizeName, size.Height)
			}
		}
	}

	return nil
}
