package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"go.uber.org/zap"
)

// Interface defines the storage operations
type Interface interface {
	// UploadImage uploads an image to storage and returns the public URL
	UploadImage(ctx context.Context, key string, data []byte, contentType string) (string, error)
	
	// DeleteImage deletes an image from storage
	DeleteImage(ctx context.Context, key string) error
	
	// GetImageURL returns the public URL for an image without checking if it exists
	GetImageURL(key string) string
}

// S3Client implements the storage Interface using AWS S3
type S3Client struct {
	client    *s3.Client
	bucket    string
	region    string
	cdnBaseURL string
	logger    *zap.SugaredLogger
}

// Config holds the configuration for the S3 client
type Config struct {
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	CDNBaseURL      string
	UsePathStyle    bool
}

// NewS3Client creates a new S3 client
func NewS3Client(cfg Config, logger *zap.SugaredLogger) (*S3Client, error) {
	// Create AWS config
	awsConfig, err := createAWSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	return &S3Client{
		client:    client,
		bucket:    cfg.Bucket,
		region:    cfg.Region,
		cdnBaseURL: cfg.CDNBaseURL,
		logger:    logger,
	}, nil
}

// createAWSConfig creates AWS SDK configuration
func createAWSConfig(cfg Config) (aws.Config, error) {
	var awsConfig aws.Config
	var err error

	// If credentials are provided, use them
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		// Use static credentials
		awsConfig, err = config.LoadDefaultConfig(
			context.Background(),
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			)),
		)
	} else {
		// Use default credential chain (environment, shared credentials, IAM role)
		awsConfig, err = config.LoadDefaultConfig(
			context.Background(),
			config.WithRegion(cfg.Region),
		)
	}

	if err != nil {
		return aws.Config{}, err
	}

	return awsConfig, nil
}

// UploadImage uploads an image to S3 and returns the public URL
func (s *S3Client) UploadImage(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	// Ensure key doesn't start with a slash
	key = strings.TrimPrefix(key, "/")

	// Create PutObject input
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPublicRead, // Make the object publicly readable
	}

	// Upload the object
	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		s.logger.Errorw("Failed to upload image to S3",
			"bucket", s.bucket,
			"key", key,
			"error", err,
		)
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	// Generate and return the URL
	url := s.GetImageURL(key)
	s.logger.Debugw("Successfully uploaded image to S3",
		"bucket", s.bucket,
		"key", key,
		"url", url,
	)
	
	return url, nil
}

// DeleteImage deletes an image from S3
func (s *S3Client) DeleteImage(ctx context.Context, key string) error {
	// Ensure key doesn't start with a slash
	key = strings.TrimPrefix(key, "/")

	// Create DeleteObject input
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	// Delete the object
	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		s.logger.Errorw("Failed to delete image from S3",
			"bucket", s.bucket,
			"key", key,
			"error", err,
		)
		return fmt.Errorf("failed to delete image: %w", err)
	}

	s.logger.Debugw("Successfully deleted image from S3",
		"bucket", s.bucket,
		"key", key,
	)
	
	return nil
}

// GetImageURL returns the public URL for an image
func (s *S3Client) GetImageURL(key string) string {
	// Ensure key doesn't start with a slash
	key = strings.TrimPrefix(key, "/")

	// If CDN base URL is provided, use it
	if s.cdnBaseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(s.cdnBaseURL, "/"), key)
	}

	// Otherwise, use the S3 URL format
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
}

// StreamToS3 uploads a stream to S3 (useful for large files)
func (s *S3Client) StreamToS3(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	// Ensure key doesn't start with a slash
	key = strings.TrimPrefix(key, "/")

	// Create PutObject input
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPublicRead, // Make the object publicly readable
	}

	// Upload the object
	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		s.logger.Errorw("Failed to stream upload to S3",
			"bucket", s.bucket,
			"key", key,
			"error", err,
		)
		return "", fmt.Errorf("failed to stream upload: %w", err)
	}

	// Generate and return the URL
	url := s.GetImageURL(key)
	s.logger.Debugw("Successfully streamed upload to S3",
		"bucket", s.bucket,
		"key", key,
		"url", url,
	)
	
	return url, nil
}

// BuildImageKey constructs a consistent S3 key for an image
func BuildImageKey(imageType, ownerGUID, imageGUID, size string) string {
	return path.Join("images", imageType, ownerGUID, imageGUID, size+".jpg")
}
