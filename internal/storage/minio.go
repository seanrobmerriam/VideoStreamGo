package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"

	"videostreamgo/internal/config"
)

// Default timeout for MinIO operations
const DefaultTimeout = 30 * time.Second

// Default storage quotas
const (
	DefaultStorageQuota     = 10 * 1024 * 1024 * 1024 // 10GB per tenant
	DefaultChunkSizeMB      = 10
	SoftDeleteRetentionDays = 30
)

// Bucket names
const (
	BucketUploads = "uploads"
	BucketExports = "exports"
	BucketBackups = "backups"
	BucketDeleted = "deleted"
	BucketPublic  = "public"
)

// MinioClient wraps the minio-go client with additional functionality
type MinioClient struct {
	client       *minio.Client
	cfg          *config.Config
	bucketPrefix string
}

// NewMinioClient creates a new MinIO client with connection pooling
func NewMinioClient(cfg *config.Config) (*MinioClient, error) {
	// Create credentials provider
	creds := credentials.NewStaticV4(
		cfg.S3.AccessKey,
		cfg.S3.SecretKey,
		"",
	)

	// Create MinIO client
	client, err := minio.New(cfg.S3.Endpoint, &minio.Options{
		Creds:  creds,
		Secure: cfg.S3.UseSSL,
		Region: cfg.S3.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	minioClient := &MinioClient{
		client:       client,
		cfg:          cfg,
		bucketPrefix: cfg.S3.BucketPrefix,
	}

	return minioClient, nil
}

// Connect establishes connection and creates required buckets
func (m *MinioClient) Connect(ctx context.Context) error {
	// Create all required buckets
	buckets := []string{BucketUploads, BucketExports, BucketBackups, BucketDeleted, BucketPublic}
	for _, bucket := range buckets {
		if err := m.createBucketIfNotExists(ctx, bucket); err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
		}
	}

	// Set lifecycle policies for soft delete
	if err := m.setupLifecyclePolicies(ctx); err != nil {
		log.Printf("Warning: Failed to setup lifecycle policies: %v", err)
	}

	log.Printf("MinIO connection established with buckets: %v", buckets)
	return nil
}

// createBucketIfNotExists creates a bucket if it doesn't exist
func (m *MinioClient) createBucketIfNotExists(ctx context.Context, bucket string) error {
	bucketName := m.getBucketName(bucket)
	exists, err := m.client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = m.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
			Region: m.cfg.S3.Region,
		})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Created bucket: %s", bucketName)
	}

	return nil
}

// setupLifecycle policies for soft delete and automatic cleanup
func (m *MinioClient) setupLifecyclePolicies(ctx context.Context) error {
	// Lifecycle policy for deleted bucket - auto delete after 30 days
	policy := lifecycle.NewConfiguration()
	policy.Rules = []lifecycle.Rule{
		{
			ID:     "AutoDeleteAfter30Days",
			Status: "Enabled",
			Expiration: lifecycle.Expiration{
				Days: SoftDeleteRetentionDays,
			},
		},
	}

	return m.client.SetBucketLifecycle(ctx, m.getBucketName(BucketDeleted), policy)
}

// getBucketName returns the full bucket name with prefix
func (m *MinioClient) getBucketName(bucket string) string {
	if m.bucketPrefix != "" {
		return m.bucketPrefix + "-" + bucket
	}
	return bucket
}

// PutObject uploads an object to MinIO with optional metadata
func (m *MinioClient) PutObject(ctx context.Context, bucket, objectKey string, reader io.Reader, objectSize int64, contentType string, metadata map[string]string) (minio.UploadInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	// Set default content type
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Set default part size
	partSize := uint64(10 * 1024 * 1024) // 10MB default
	if m.cfg.S3.ChunkSizeMB > 0 {
		partSize = uint64(m.cfg.S3.ChunkSizeMB) * 1024 * 1024
	}

	// Prepare user metadata with prefix
	userMetadata := make(map[string]string)
	for k, v := range metadata {
		userMetadata["X-Amz-Meta-"+k] = v
	}

	opts := minio.PutObjectOptions{
		ContentType:  contentType,
		UserMetadata: userMetadata,
		PartSize:     partSize,
	}

	return m.client.PutObject(ctx, m.getBucketName(bucket), objectKey, reader, objectSize, opts)
}

// GetObject retrieves an object from MinIO
func (m *MinioClient) GetObject(ctx context.Context, bucket, objectKey string) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return m.client.GetObject(ctx, m.getBucketName(bucket), objectKey, minio.GetObjectOptions{})
}

// GetObjectInfo retrieves object metadata
func (m *MinioClient) GetObjectInfo(ctx context.Context, bucket, objectKey string) (minio.ObjectInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return m.client.StatObject(ctx, m.getBucketName(bucket), objectKey, minio.StatObjectOptions{})
}

// DeleteObject deletes an object from MinIO
func (m *MinioClient) DeleteObject(ctx context.Context, bucket, objectKey string) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return m.client.RemoveObject(ctx, m.getBucketName(bucket), objectKey, minio.RemoveObjectOptions{})
}

// SoftDelete copies an object to the deleted bucket with retention metadata, then deletes the original
func (m *MinioClient) SoftDelete(ctx context.Context, sourceBucket, objectKey string) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	// Copy object to deleted bucket with deletion timestamp
	deletedKey := fmt.Sprintf("%s/%d_%s", filepath.Dir(objectKey), time.Now().Unix(), filepath.Base(objectKey))

	// Copy using the source as a reader
	src, err := m.GetObject(ctx, sourceBucket, objectKey)
	if err != nil {
		return fmt.Errorf("failed to get source object: %w", err)
	}
	defer src.Close()

	// Get object info for size
	info, err := m.GetObjectInfo(ctx, sourceBucket, objectKey)
	if err != nil {
		return fmt.Errorf("failed to get object info: %w", err)
	}

	// Upload to deleted bucket with metadata
	metadata := map[string]string{
		"DeletedAt":      time.Now().Format(time.RFC3339),
		"OriginalBucket": sourceBucket,
		"OriginalKey":    objectKey,
	}

	_, err = m.PutObject(ctx, BucketDeleted, deletedKey, src, info.Size, info.ContentType, metadata)
	if err != nil {
		return fmt.Errorf("failed to copy to deleted bucket: %w", err)
	}

	// Delete from source
	return m.DeleteObject(ctx, sourceBucket, objectKey)
}

// ListObjects lists objects in a bucket with prefix
func (m *MinioClient) ListObjects(ctx context.Context, bucket, prefix string) <-chan minio.ObjectInfo {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return m.client.ListObjects(ctx, m.getBucketName(bucket), minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
}

// GenerateObjectKey generates a unique object key for tenant uploads
func (m *MinioClient) GenerateObjectKey(tenantID, userID, filename string) string {
	timestamp := time.Now().Format("20060102150405")
	sanitizedFilename := strings.ReplaceAll(filename, " ", "_")
	return fmt.Sprintf("%s/%s/%s_%s", tenantID, userID, timestamp, sanitizedFilename)
}

// Health checks MinIO connectivity
func (m *MinioClient) Health(ctx context.Context) map[string]interface{} {
	status := "healthy"
	var errMsg string

	_, err := m.client.BucketExists(ctx, m.getBucketName(BucketUploads))
	if err != nil {
		status = "unhealthy"
		errMsg = err.Error()
	}

	return map[string]interface{}{
		"status":        status,
		"endpoint":      m.cfg.S3.Endpoint,
		"bucket_prefix": m.bucketPrefix,
		"error":         errMsg,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}
}

// GetClient returns the underlying minio client
func (m *MinioClient) GetClient() *minio.Client {
	return m.client
}

// Close closes the MinIO client (no-op for minio-go)
func (m *MinioClient) Close() error {
	return nil
}
