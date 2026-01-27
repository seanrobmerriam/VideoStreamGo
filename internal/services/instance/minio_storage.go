package instance

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"videostreamgo/internal/config"
	"videostreamgo/internal/storage"
)

// MinioStorageService implements StorageService using MinIO
type MinioStorageService struct {
	minioClient *storage.MinioClient
	presigned   *storage.PresignedURLGenerator
	cdnURL      string
	maxRetries  int
}

// NewMinioStorageService creates a new MinIO-based storage service
func NewMinioStorageService(cfg *config.Config, minioClient *storage.MinioClient) (StorageService, error) {
	presigned := storage.NewPresignedURLGenerator(minioClient, cfg)

	// Build CDN URL
	scheme := "http"
	if cfg.S3.UseSSL {
		scheme = "https"
	}
	cdnURL := fmt.Sprintf("%s://%s", scheme, cfg.S3.Endpoint)

	return &MinioStorageService{
		minioClient: minioClient,
		presigned:   presigned,
		cdnURL:      cdnURL,
		maxRetries:  cfg.S3.MaxRetries,
	}, nil
}

// generateStoragePath generates a unique storage path for a video
func (s *MinioStorageService) generateStoragePath(videoID uuid.UUID, filename string) string {
	date := time.Now().Format("2006/01/02")
	return fmt.Sprintf("videos/%s/%s/%s", date, videoID.String()[:8], filename)
}

// UploadVideo uploads a complete video file
func (s *MinioStorageService) UploadVideo(req *UploadRequest) (*StorageResult, error) {
	ctx := context.Background()
	startTime := time.Now()

	// Open the temp file
	file, err := os.Open(req.TempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open temp file: %w", err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Generate storage path
	storagePath := s.generateStoragePath(req.VideoID, req.FileName)

	// Upload to MinIO
	uploadInfo, err := s.minioClient.PutObject(ctx, storage.BucketUploads, storagePath, file, fileInfo.Size(), req.ContentType, map[string]string{
		"original_filename": req.FileName,
		"video_id":          req.VideoID.String(),
		"user_id":           req.UserID.String(),
		"uploaded_at":       time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload to MinIO: %w", err)
	}

	return &StorageResult{
		VideoID:        req.VideoID,
		StoragePath:    storagePath,
		URL:            s.cdnURL + "/" + storage.BucketUploads + "/" + storagePath,
		FileSize:       fileInfo.Size(),
		ContentType:    req.ContentType,
		ETag:           uploadInfo.ETag,
		UploadDuration: time.Since(startTime),
	}, nil
}

// UploadChunk uploads a single chunk of a video
func (s *MinioStorageService) UploadChunk(sessionID uuid.UUID, chunkNumber int, chunkData []byte) error {
	ctx := context.Background()

	// Generate chunk storage path
	chunkPath := fmt.Sprintf("chunks/%s/%d", sessionID.String(), chunkNumber)

	// Upload chunk to MinIO
	reader := bytes.NewReader(chunkData)
	_, err := s.minioClient.PutObject(ctx, storage.BucketUploads, chunkPath, reader, int64(len(chunkData)), "application/octet-stream", map[string]string{
		"session_id":   sessionID.String(),
		"chunk_number": fmt.Sprintf("%d", chunkNumber),
		"uploaded_at":  time.Now().Format(time.RFC3339),
	})
	return err
}

// CompleteUpload completes an upload session by combining chunks
func (s *MinioStorageService) CompleteUpload(sessionID uuid.UUID) error {
	// Chunks are already uploaded, this is a no-op for MinIO
	// The actual assembly would happen during processing
	return nil
}

// DeleteVideo deletes a video using soft delete
func (s *MinioStorageService) DeleteVideo(videoID string) error {
	ctx := context.Background()

	// List all objects for this video and soft delete them
	prefix := fmt.Sprintf("videos/%s/", videoID[:8])
	for object := range s.minioClient.ListObjects(ctx, storage.BucketUploads, prefix) {
		if object.Err != nil {
			continue
		}
		if err := s.minioClient.SoftDelete(ctx, storage.BucketUploads, object.Key); err != nil {
			// Log but continue
			fmt.Printf("warning: failed to soft delete %s: %v\n", object.Key, err)
		}
	}

	return nil
}

// GetSignedURL generates a signed URL for video access
func (s *MinioStorageService) GetSignedURL(videoID string, action string, expiry time.Duration) (string, error) {
	ctx := context.Background()

	objectKey := fmt.Sprintf("videos/%s/video.mp4", videoID[:8])
	return s.presigned.GenerateDownloadURL(ctx, storage.BucketUploads, objectKey)
}

// GetStreamURL generates a streaming URL for a video
func (s *MinioStorageService) GetStreamURL(videoID string, quality string) (string, error) {
	if quality == "hls" {
		return s.cdnURL + "/" + storage.BucketUploads + "/videos/" + videoID[:8] + "/master.m3u8", nil
	}
	return s.cdnURL + "/" + storage.BucketUploads + "/videos/" + videoID[:8] + "/" + quality + "/index.m3u8", nil
}

// GetUploadPresignedURL generates a presigned URL for chunk upload
func (s *MinioStorageService) GetUploadPresignedURL(sessionID uuid.UUID, chunkNumber int, expiry time.Duration) (string, error) {
	ctx := context.Background()
	chunkPath := fmt.Sprintf("chunks/%s/%d", sessionID.String(), chunkNumber)
	return s.presigned.GenerateUploadURL(ctx, storage.BucketUploads, chunkPath, "application/octet-stream", 0)
}

// InitHLSUpload initializes HLS upload
func (s *MinioStorageService) InitHLSUpload(videoID string) (string, string, error) {
	prefix := "videos/" + videoID[:8] + "/hls/"
	return prefix, storage.BucketUploads + "/" + prefix, nil
}

// UploadHLSSegment uploads an HLS segment
func (s *MinioStorageService) UploadHLSSegment(videoID string, segmentNumber int, data []byte) error {
	ctx := context.Background()
	segmentPath := fmt.Sprintf("videos/%s/hls/%d.ts", videoID[:8], segmentNumber)

	reader := bytes.NewReader(data)
	_, err := s.minioClient.PutObject(ctx, storage.BucketUploads, segmentPath, reader, int64(len(data)), "video/mp2t", map[string]string{
		"video_id":       videoID,
		"segment_number": fmt.Sprintf("%d", segmentNumber),
	})
	return err
}

// GetTempFilePath returns a temporary file path for video uploads
func (s *MinioStorageService) GetTempFilePath(videoID uuid.UUID, filename string) string {
	// For MinIO, we still use temp files for staging
	return fmt.Sprintf("/tmp/videostreamgo-%s_%s", videoID.String(), filename)
}

// Ensure MinioStorageService implements StorageService
var _ StorageService = (*MinioStorageService)(nil)
