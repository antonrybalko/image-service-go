package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Save original environment variables to restore later
	origEnv := make(map[string]string)
	for _, env := range []string{
		"ENVIRONMENT", "PORT",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"S3_REGION", "S3_BUCKET", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY",
		"S3_ENDPOINT", "S3_CDN_BASE_URL", "S3_USE_PATH_STYLE",
		"JWT_PUBLIC_KEY_URL", "JWT_SECRET", "JWT_ALGORITHM",
		"IMAGE_CONFIG_PATH",
	} {
		if val, ok := os.LookupEnv(env); ok {
			origEnv[env] = val
		}
	}

	// Restore environment variables after test
	defer func() {
		for key := range origEnv {
			os.Unsetenv(key)
		}
		for key, val := range origEnv {
			os.Setenv(key, val)
		}
	}()

	// Clear all relevant environment variables before test
	for key := range origEnv {
		os.Unsetenv(key)
	}

	t.Run("DefaultValues", func(t *testing.T) {
		// Clear all environment variables to test defaults
		for key := range origEnv {
			os.Unsetenv(key)
		}

		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)

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
		assert.Empty(t, cfg.S3.AccessKeyID)
		assert.Empty(t, cfg.S3.SecretAccessKey)
		assert.Empty(t, cfg.S3.Endpoint)
		assert.Empty(t, cfg.S3.CDNBaseURL)
		assert.False(t, cfg.S3.UsePathStyle)

		// JWT defaults
		assert.Equal(t, "RS256", cfg.JWT.Algorithm)
		assert.Empty(t, cfg.JWT.PublicKeyURL)
		assert.Empty(t, cfg.JWT.Secret)

		// Image config defaults
		assert.Equal(t, "config/images.yaml", cfg.ImageConfig.ConfigPath)
	})

	t.Run("EnvironmentVariables", func(t *testing.T) {
		// Set environment variables
		os.Setenv("ENVIRONMENT", "production")
		os.Setenv("PORT", "9090")
		os.Setenv("DB_HOST", "db.example.com")
		os.Setenv("DB_PORT", "5433")
		os.Setenv("DB_USER", "testuser")
		os.Setenv("DB_PASSWORD", "testpass")
		os.Setenv("DB_NAME", "testdb")
		os.Setenv("DB_SSLMODE", "require")
		os.Setenv("S3_REGION", "eu-west-1")
		os.Setenv("S3_BUCKET", "test-bucket")
		os.Setenv("S3_ACCESS_KEY_ID", "test-key")
		os.Setenv("S3_SECRET_ACCESS_KEY", "test-secret")
		os.Setenv("S3_ENDPOINT", "https://s3.example.com")
		os.Setenv("S3_CDN_BASE_URL", "https://cdn.example.com")
		os.Setenv("S3_USE_PATH_STYLE", "true")
		os.Setenv("JWT_PUBLIC_KEY_URL", "https://auth.example.com/.well-known/jwks.json")
		os.Setenv("JWT_SECRET", "test-secret")
		os.Setenv("JWT_ALGORITHM", "HS256")
		os.Setenv("IMAGE_CONFIG_PATH", "/etc/image-service/config.yaml")

		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Verify environment values
		assert.Equal(t, "production", cfg.Environment)
		assert.Equal(t, 9090, cfg.Port)

		// Database values
		assert.Equal(t, "db.example.com", cfg.DB.Host)
		assert.Equal(t, 5433, cfg.DB.Port)
		assert.Equal(t, "testuser", cfg.DB.User)
		assert.Equal(t, "testpass", cfg.DB.Password)
		assert.Equal(t, "testdb", cfg.DB.Name)
		assert.Equal(t, "require", cfg.DB.SSLMode)

		// S3 values
		assert.Equal(t, "eu-west-1", cfg.S3.Region)
		assert.Equal(t, "test-bucket", cfg.S3.Bucket)
		assert.Equal(t, "test-key", cfg.S3.AccessKeyID)
		assert.Equal(t, "test-secret", cfg.S3.SecretAccessKey)
		assert.Equal(t, "https://s3.example.com", cfg.S3.Endpoint)
		assert.Equal(t, "https://cdn.example.com", cfg.S3.CDNBaseURL)
		assert.True(t, cfg.S3.UsePathStyle)

		// JWT values
		assert.Equal(t, "HS256", cfg.JWT.Algorithm)
		assert.Equal(t, "https://auth.example.com/.well-known/jwks.json", cfg.JWT.PublicKeyURL)
		assert.Equal(t, "test-secret", cfg.JWT.Secret)

		// Image config values
		assert.Equal(t, "/etc/image-service/config.yaml", cfg.ImageConfig.ConfigPath)
	})

	t.Run("InvalidConfigFile", func(t *testing.T) {
		os.Setenv("CONFIG_FILE", "non_existent_file.yaml")
		
		_, err := Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
		
		os.Unsetenv("CONFIG_FILE")
	})
}
