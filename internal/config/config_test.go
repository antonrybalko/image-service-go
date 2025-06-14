package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clear any environment variables that might affect the test
	os.Clearenv()

	// Load config with default values
	cfg, err := Load()
	require.NoError(t, err, "Loading default config should not error")
	require.NotNil(t, cfg, "Config should not be nil")

	// Verify default values
	assert.Equal(t, "development", cfg.Environment)
	assert.Equal(t, 8080, cfg.Port)

	// Database defaults
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, "postgres", cfg.DB.User)
	assert.Equal(t, "postgres", cfg.DB.Password)
	assert.Equal(t, "image_service", cfg.DB.Name)
	assert.Equal(t, "disable", cfg.DB.SSLMode)

	// S3 defaults
	assert.Equal(t, "us-east-1", cfg.S3.Region)
	assert.Equal(t, "images", cfg.S3.Bucket)
	assert.Equal(t, "", cfg.S3.Endpoint)
	assert.Equal(t, "", cfg.S3.CDNBaseURL)
	assert.Equal(t, false, cfg.S3.UsePathStyle)

	// JWT defaults
	assert.Equal(t, "RS256", cfg.JWT.Algorithm)

	// Image config defaults
	assert.Equal(t, "config/images.yaml", cfg.ImageConfig.ConfigPath)
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	// Clear any environment variables that might affect the test
	os.Clearenv()

	// Set environment variables
	envVars := map[string]string{
		"ENVIRONMENT":          "production",
		"PORT":                 "9090",
		"DB_HOST":              "db.example.com",
		"DB_PORT":              "5433",
		"DB_USER":              "dbuser",
		"DB_PASSWORD":          "dbpass",
		"DB_NAME":              "imagedb",
		"DB_SSLMODE":           "require",
		"S3_REGION":            "eu-west-1",
		"S3_BUCKET":            "my-images",
		"S3_ACCESS_KEY_ID":     "access123",
		"S3_SECRET_ACCESS_KEY": "secret456",
		"S3_ENDPOINT":          "https://minio.example.com",
		"S3_CDN_BASE_URL":      "https://cdn.example.com",
		"S3_USE_PATH_STYLE":    "true",
		"JWT_PUBLIC_KEY_URL":   "https://auth.example.com/.well-known/jwks.json",
		"JWT_SECRET":           "supersecret",
		"JWT_ALGORITHM":        "HS256",
		"IMAGE_CONFIG_PATH":    "test/images.yaml",
	}

	for k, v := range envVars {
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("Failed to set environment variable %s: %v", k, err)
		}
	}

	// Load config with environment variables
	cfg, err := Load()
	require.NoError(t, err, "Loading config with environment variables should not error")
	require.NotNil(t, cfg, "Config should not be nil")

	// Verify environment variables were loaded correctly
	assert.Equal(t, "production", cfg.Environment)
	assert.Equal(t, 9090, cfg.Port)

	// Database config
	assert.Equal(t, "db.example.com", cfg.DB.Host)
	assert.Equal(t, 5433, cfg.DB.Port)
	assert.Equal(t, "dbuser", cfg.DB.User)
	assert.Equal(t, "dbpass", cfg.DB.Password)
	assert.Equal(t, "imagedb", cfg.DB.Name)
	assert.Equal(t, "require", cfg.DB.SSLMode)

	// S3 config
	assert.Equal(t, "eu-west-1", cfg.S3.Region)
	assert.Equal(t, "my-images", cfg.S3.Bucket)
	assert.Equal(t, "access123", cfg.S3.AccessKeyID)
	assert.Equal(t, "secret456", cfg.S3.SecretAccessKey)
	assert.Equal(t, "https://minio.example.com", cfg.S3.Endpoint)
	assert.Equal(t, "https://cdn.example.com", cfg.S3.CDNBaseURL)
	assert.Equal(t, true, cfg.S3.UsePathStyle)

	// JWT config
	assert.Equal(t, "https://auth.example.com/.well-known/jwks.json", cfg.JWT.PublicKeyURL)
	assert.Equal(t, "supersecret", cfg.JWT.Secret)
	assert.Equal(t, "HS256", cfg.JWT.Algorithm)

	// Image config
	assert.Equal(t, "test/images.yaml", cfg.ImageConfig.ConfigPath)

	// Clean up
	os.Clearenv()
}

func TestLoad_InvalidValues(t *testing.T) {
	// Clear any environment variables that might affect the test
	os.Clearenv()

	// Test with invalid PORT (non-numeric)
	if err := os.Setenv("PORT", "invalid"); err != nil {
		t.Fatalf("Failed to set PORT environment variable: %v", err)
	}
	_, err := Load()
	assert.Error(t, err, "Loading config with invalid PORT should error")

	// Clean up
	os.Clearenv()
}
