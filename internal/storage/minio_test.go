package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"videostreamgo/internal/config"
)

func TestGenerateObjectKey(t *testing.T) {
	cfg := &config.Config{}
	cfg.S3.Endpoint = "localhost:9000"
	cfg.S3.AccessKey = "minioadmin"
	cfg.S3.SecretKey = "minioadmin"
	cfg.S3.Bucket = "videostreamgo"
	cfg.S3.BucketPrefix = "test"
	cfg.S3.Region = "us-east-1"
	cfg.S3.UseSSL = false

	client, err := NewMinioClient(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Test key generation
	objectKey := client.GenerateObjectKey("tenant123", "user456", "video.mp4")
	assert.Contains(t, objectKey, "tenant123/user456/")
	assert.Contains(t, objectKey, "video.mp4")
}

func TestGetBucketName(t *testing.T) {
	cfg := &config.Config{}
	cfg.S3.Endpoint = "localhost:9000"
	cfg.S3.BucketPrefix = "myapp"

	client, err := NewMinioClient(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Test with prefix
	bucketName := client.getBucketName("uploads")
	assert.Equal(t, "myapp-uploads", bucketName)
}

func TestGetBucketNameWithoutPrefix(t *testing.T) {
	cfg := &config.Config{}
	cfg.S3.Endpoint = "localhost:9000"
	cfg.S3.BucketPrefix = ""

	client, err := NewMinioClient(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	bucketName := client.getBucketName("uploads")
	assert.Equal(t, "uploads", bucketName)
}

func TestDefaultPresignedConfig(t *testing.T) {
	cfg := DefaultPresignedConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, "15m0s", cfg.UploadExpiry.String())
	assert.Equal(t, "1h0m0s", cfg.DownloadExpiry.String())
	assert.Equal(t, int64(5368709120), cfg.MaxUploadSize)
	assert.True(t, cfg.AllowedTypes["video/mp4"])
	assert.True(t, cfg.AllowedTypes["image/jpeg"])
	assert.False(t, cfg.AllowedTypes["application/pdf"])
}

func TestStorageQuotaKey(t *testing.T) {
	key := StorageQuotaKey("tenant123")
	assert.Equal(t, "storage:tenant123:bytes", key)
}

func TestStorageQuotaLimitKey(t *testing.T) {
	key := StorageQuotaLimitKey("tenant123")
	assert.Equal(t, "storage:tenant123:limit", key)
}

func TestBucketNames(t *testing.T) {
	assert.Equal(t, "uploads", BucketUploads)
	assert.Equal(t, "exports", BucketExports)
	assert.Equal(t, "backups", BucketBackups)
	assert.Equal(t, "deleted", BucketDeleted)
	assert.Equal(t, "public", BucketPublic)
}

func TestMinioClientCreationWithSSL(t *testing.T) {
	cfg := &config.Config{}
	cfg.S3.Endpoint = "minio.example.com:9000"
	cfg.S3.AccessKey = "minioadmin"
	cfg.S3.SecretKey = "minioadmin"
	cfg.S3.Bucket = "videostreamgo"
	cfg.S3.Region = "us-east-1"
	cfg.S3.UseSSL = true

	client, err := NewMinioClient(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, cfg, client.cfg)
	assert.Equal(t, "", client.bucketPrefix)
}

func TestMinioClientWithPrefix(t *testing.T) {
	cfg := &config.Config{}
	cfg.S3.Endpoint = "localhost:9000"
	cfg.S3.AccessKey = "minioadmin"
	cfg.S3.SecretKey = "minioadmin"
	cfg.S3.Bucket = "videostreamgo"
	cfg.S3.BucketPrefix = "production"
	cfg.S3.Region = "us-east-1"
	cfg.S3.UseSSL = false

	client, err := NewMinioClient(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "production", client.bucketPrefix)
}

func TestClose(t *testing.T) {
	cfg := &config.Config{}
	cfg.S3.Endpoint = "localhost:9000"
	cfg.S3.AccessKey = "minioadmin"
	cfg.S3.SecretKey = "minioadmin"
	cfg.S3.Bucket = "videostreamgo"
	cfg.S3.Region = "us-east-1"
	cfg.S3.UseSSL = false

	client, err := NewMinioClient(cfg)
	assert.NoError(t, err)

	// Close should not error
	err = client.Close()
	assert.NoError(t, err)
}

func TestDefaultStorageConstants(t *testing.T) {
	// Test that constants are defined correctly
	assert.True(t, DefaultStorageQuota > 0)
	assert.Equal(t, 10, DefaultChunkSizeMB)
	assert.Equal(t, 30, SoftDeleteRetentionDays)
}

func TestDefaultTimeout(t *testing.T) {
	assert.Equal(t, "30s", DefaultTimeout.String())
}

func TestPresignedURLGeneratorCreation(t *testing.T) {
	cfg := &config.Config{}
	cfg.S3.Endpoint = "localhost:9000"
	cfg.S3.AccessKey = "minioadmin"
	cfg.S3.SecretKey = "minioadmin"
	cfg.S3.Bucket = "videostreamgo"
	cfg.S3.Region = "us-east-1"
	cfg.S3.UseSSL = false

	minioClient, err := NewMinioClient(cfg)
	assert.NoError(t, err)

	presignedGen := NewPresignedURLGenerator(minioClient, cfg)
	assert.NotNil(t, presignedGen)
}

func TestGenerateUploadResponseForTenant(t *testing.T) {
	cfg := &config.Config{}
	cfg.S3.Endpoint = "localhost:9000"
	cfg.S3.AccessKey = "minioadmin"
	cfg.S3.SecretKey = "minioadmin"
	cfg.S3.Bucket = "videostreamgo"
	cfg.S3.Region = "us-east-1"
	cfg.S3.UseSSL = false

	minioClient, err := NewMinioClient(cfg)
	assert.NoError(t, err)

	presignedGen := NewPresignedURLGenerator(minioClient, cfg)
	assert.NotNil(t, presignedGen)

	// Test that ownership validation works
	isOwner := presignedGen.ValidateOwnership(nil, "tenant123", "user456", "tenant123/user456/video.mp4")
	assert.True(t, isOwner)

	isNotOwner := presignedGen.ValidateOwnership(nil, "tenant123", "user456", "tenant789/user456/video.mp4")
	assert.False(t, isNotOwner)
}
