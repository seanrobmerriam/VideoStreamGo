package storage

import (
	"context"
	"fmt"
	"time"

	"videostreamgo/internal/config"
)

// PresignedURLConfig holds configuration for presigned URLs
type PresignedURLConfig struct {
	UploadExpiry   time.Duration
	DownloadExpiry time.Duration
	MaxUploadSize  int64
	AllowedTypes   map[string]bool
}

// DefaultPresignedConfig returns default configuration
func DefaultPresignedConfig() *PresignedURLConfig {
	return &PresignedURLConfig{
		UploadExpiry:   15 * time.Minute,
		DownloadExpiry: 1 * time.Hour,
		MaxUploadSize:  5 * 1024 * 1024 * 1024, // 5GB
		AllowedTypes: map[string]bool{
			"video/mp4":        true,
			"video/mpeg":       true,
			"video/quicktime":  true,
			"video/x-msvideo":  true,
			"video/x-matroska": true,
			"video/webm":       true,
			"video/avi":        true,
			"video/x-flv":      true,
			"video/mp2t":       true,
			"image/jpeg":       true,
			"image/png":        true,
			"image/gif":        true,
			"image/webp":       true,
		},
	}
}

// PresignedURLGenerator generates presigned URLs for uploads and downloads
type PresignedURLGenerator struct {
	client       *MinioClient
	cfg          *config.Config
	presignedCfg *PresignedURLConfig
}

// NewPresignedURLGenerator creates a new presigned URL generator
func NewPresignedURLGenerator(client *MinioClient, cfg *config.Config) *PresignedURLGenerator {
	return &PresignedURLGenerator{
		client:       client,
		cfg:          cfg,
		presignedCfg: DefaultPresignedConfig(),
	}
}

// GenerateUploadURL generates a presigned URL for uploading an object
func (g *PresignedURLGenerator) GenerateUploadURL(ctx context.Context, bucket, objectKey, contentType string, contentLength int64) (string, error) {
	// Validate content type
	if !g.presignedCfg.AllowedTypes[contentType] {
		return "", fmt.Errorf("invalid content type: %s", contentType)
	}

	// Validate file size
	if contentLength > g.presignedCfg.MaxUploadSize {
		return "", fmt.Errorf("file size exceeds maximum allowed size of %d bytes", g.presignedCfg.MaxUploadSize)
	}

	// Generate presigned PUT URL for simple upload
	url, err := g.client.GetClient().PresignedPutObject(ctx, g.client.getBucketName(bucket), objectKey, g.presignedCfg.UploadExpiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}

	return url.String(), nil
}

// GenerateDownloadURL generates a presigned URL for downloading an object
func (g *PresignedURLGenerator) GenerateDownloadURL(ctx context.Context, bucket, objectKey string) (string, error) {
	// Check if object exists
	_, err := g.client.GetObjectInfo(ctx, bucket, objectKey)
	if err != nil {
		return "", fmt.Errorf("object not found: %w", err)
	}

	// Generate presigned GET URL
	url, err := g.client.GetClient().PresignedGetObject(ctx, g.client.getBucketName(bucket), objectKey, g.presignedCfg.DownloadExpiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}

	return url.String(), nil
}

// GenerateUploadResponse combines upload URL with object key for client
type GenerateUploadResponse struct {
	UploadURL   string `json:"upload_url"`
	ObjectKey   string `json:"object_key"`
	Bucket      string `json:"bucket"`
	ExpiresIn   int    `json:"expires_in_seconds"`
	ContentType string `json:"content_type,omitempty"`
}

// GenerateUploadResponseForTenant generates an upload URL for a tenant with auto-generated object key
func (g *PresignedURLGenerator) GenerateUploadResponseForTenant(ctx context.Context, tenantID, userID, filename, contentType string, contentLength int64) (*GenerateUploadResponse, error) {
	// Generate unique object key
	objectKey := g.client.GenerateObjectKey(tenantID, userID, filename)

	// Generate presigned URL
	uploadURL, err := g.GenerateUploadURL(ctx, BucketUploads, objectKey, contentType, contentLength)
	if err != nil {
		return nil, err
	}

	return &GenerateUploadResponse{
		UploadURL:   uploadURL,
		ObjectKey:   objectKey,
		Bucket:      BucketUploads,
		ExpiresIn:   int(g.presignedCfg.UploadExpiry.Seconds()),
		ContentType: contentType,
	}, nil
}

// GenerateDownloadResponse combines download URL with metadata
type GenerateDownloadResponse struct {
	DownloadURL string `json:"download_url"`
	ObjectKey   string `json:"object_key"`
	Bucket      string `json:"bucket"`
	ExpiresIn   int    `json:"expires_in_seconds"`
	ContentType string `json:"content_type"`
	FileSize    int64  `json:"file_size"`
}

// GenerateDownloadResponseForObject generates a download URL for an object
func (g *PresignedURLGenerator) GenerateDownloadResponseForObject(ctx context.Context, bucket, objectKey string) (*GenerateDownloadResponse, error) {
	// Get object info
	info, err := g.client.GetObjectInfo(ctx, bucket, objectKey)
	if err != nil {
		return nil, fmt.Errorf("object not found: %w", err)
	}

	// Generate presigned URL
	downloadURL, err := g.GenerateDownloadURL(ctx, bucket, objectKey)
	if err != nil {
		return nil, err
	}

	return &GenerateDownloadResponse{
		DownloadURL: downloadURL,
		ObjectKey:   objectKey,
		Bucket:      bucket,
		ExpiresIn:   int(g.presignedCfg.DownloadExpiry.Seconds()),
		ContentType: info.ContentType,
		FileSize:    info.Size,
	}, nil
}

// ValidateOwnership validates that a user has access to an object
func (g *PresignedURLGenerator) ValidateOwnership(ctx context.Context, tenantID, userID, objectKey string) bool {
	// Check if the object key starts with the tenant/user path
	expectedPrefix := fmt.Sprintf("%s/%s/", tenantID, userID)
	return len(objectKey) >= len(expectedPrefix) && objectKey[:len(expectedPrefix)] == expectedPrefix
}
