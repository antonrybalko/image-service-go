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
	"github.com/google/uuid"
)

// S3Config holds configuration for S3 storage
type S3Config struct {
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // Optional: for MinIO or other S3-compatible services
	CDNBaseURL      string // Optional: for URL rewriting
	UsePathStyle    bool   // Use path-style addressing (for MinIO)
}

// S3Interface defines the operations for S3 storage
type S3Interface interface {
	// Put uploads an object to S3 and returns the public URL
	Put(ctx context.Context, key string, body []byte, contentType string) (string, error)
	
	// Get retrieves an object from S3
	Get(ctx context.Context, key string) ([]byte, error)
	
	// Delete removes an object from S3
	Delete(ctx context.Context, key string) error
	
	// GenerateUserImageKey generates a consistent key for user images
	GenerateUserImageKey(userGUID uuid.UUID, imageGUID uuid.UUID, size string) string
	
	// GenerateOrganizationImageKey generates a consistent key for organization images
	GenerateOrganizationImageKey(orgGUID uuid.UUID, imageGUID uuid.UUID, size string) string
	
	// GetURL returns the URL for an object
	GetURL(key string) string
}

// S3Client implements S3Interface using AWS SDK
type S3Client struct {
	client    *s3.Client
	bucket    string
	region    string
	cdnBaseURL string
}

// NewS3Client creates a new S3 client
func NewS3Client(cfg S3Config) (S3Interface, error) {
	var opts []func(*config.LoadOptions) error
	
	// Use custom endpoint if provided (for MinIO, etc.)
	if cfg.Endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				HostnameImmutable: true,
				SigningRegion:     cfg.Region,
			}, nil
		})
		opts = append(opts, config.WithEndpointResolverWithOptions(customResolver))
	}
	
	// Use credentials if provided
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}
	
	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(cfg.Region),
		config.WithDefaultsMode(aws.DefaultsModeStandard),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	
	// Create S3 client
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.UsePathStyle {
			o.UsePathStyle = true
		}
	})
	
	return &S3Client{
		client:    s3Client,
		bucket:    cfg.Bucket,
		region:    cfg.Region,
		cdnBaseURL: cfg.CDNBaseURL,
	}, nil
}

// Put uploads an object to S3 and returns the public URL
func (s *S3Client) Put(ctx context.Context, key string, body []byte, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object to S3: %w", err)
	}
	
	return s.GetURL(key), nil
}

// Get retrieves an object from S3
func (s *S3Client) Get(ctx context.Context, key string) ([]byte, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()
	
	return io.ReadAll(result.Body)
}

// Delete removes an object from S3
func (s *S3Client) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from S3: %w", err)
	}
	
	return nil
}

// GenerateUserImageKey generates a consistent key for user images
func (s *S3Client) GenerateUserImageKey(userGUID uuid.UUID, imageGUID uuid.UUID, size string) string {
	return fmt.Sprintf("images/user/%s/%s/%s.jpg", userGUID.String(), imageGUID.String(), size)
}

// GenerateOrganizationImageKey generates a consistent key for organization images
func (s *S3Client) GenerateOrganizationImageKey(orgGUID uuid.UUID, imageGUID uuid.UUID, size string) string {
	return fmt.Sprintf("images/organization/%s/%s/%s.jpg", orgGUID.String(), imageGUID.String(), size)
}

// GetURL returns the URL for an object
func (s *S3Client) GetURL(key string) string {
	// If CDN base URL is provided, use it
	if s.cdnBaseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.cdnBaseURL, "/"), key)
	}
	
	// Otherwise, construct S3 URL
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
}

// Helper function to extract the filename from a key
func GetFilenameFromKey(key string) string {
	return path.Base(key)
}
