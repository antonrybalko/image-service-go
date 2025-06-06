package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service
type Config struct {
	// Core service configuration
	Environment string `mapstructure:"ENVIRONMENT"`
	Port        int    `mapstructure:"PORT"`

	// Database configuration
	DB struct {
		Host     string `mapstructure:"DB_HOST"`
		Port     int    `mapstructure:"DB_PORT"`
		User     string `mapstructure:"DB_USER"`
		Password string `mapstructure:"DB_PASSWORD"`
		Name     string `mapstructure:"DB_NAME"`
		SSLMode  string `mapstructure:"DB_SSLMODE"`
	}

	// S3 storage configuration
	S3 struct {
		Region          string `mapstructure:"S3_REGION"`
		Bucket          string `mapstructure:"S3_BUCKET"`
		AccessKeyID     string `mapstructure:"S3_ACCESS_KEY_ID"`
		SecretAccessKey string `mapstructure:"S3_SECRET_ACCESS_KEY"`
		Endpoint        string `mapstructure:"S3_ENDPOINT"`
		CDNBaseURL      string `mapstructure:"S3_CDN_BASE_URL"`
		UsePathStyle    bool   `mapstructure:"S3_USE_PATH_STYLE"`
	}

	// JWT Authentication configuration
	JWT struct {
		PublicKeyURL string `mapstructure:"JWT_PUBLIC_KEY_URL"`
		Secret       string `mapstructure:"JWT_SECRET"`
		Algorithm    string `mapstructure:"JWT_ALGORITHM"`
	}

	// Image configuration
	ImageConfig struct {
		ConfigPath string `mapstructure:"IMAGE_CONFIG_PATH"`
	}
}

// Load reads the configuration from environment variables and returns a Config struct
func Load() (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Read from environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Optional: Read from config file if specified
	configFile := v.GetString("CONFIG_FILE")
	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal config into struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Core service defaults
	v.SetDefault("ENVIRONMENT", "development")
	v.SetDefault("PORT", 8080)

	// Database defaults
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_USER", "postgres")
	v.SetDefault("DB_PASSWORD", "postgres")
	v.SetDefault("DB_NAME", "image_service")
	v.SetDefault("DB_SSLMODE", "disable")

	// S3 defaults
	v.SetDefault("S3_REGION", "us-east-1")
	v.SetDefault("S3_BUCKET", "images")
	v.SetDefault("S3_ENDPOINT", "")
	v.SetDefault("S3_CDN_BASE_URL", "")
	v.SetDefault("S3_USE_PATH_STYLE", false)

	// JWT defaults
	v.SetDefault("JWT_ALGORITHM", "RS256")

	// Image config defaults
	v.SetDefault("IMAGE_CONFIG_PATH", "config/images.yaml")
}
