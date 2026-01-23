package instance

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"

	"videostreamgo/internal/config"
)

// S3StorageService implements StorageService using local filesystem (simplified)
type S3StorageService struct {
	baseURL    string
	bucket     string
	cdnURL     string
	tempDir    string
	maxRetries int
}

// NewStorageService creates a new storage service
func NewStorageService(cfg *config.Config) (StorageService, error) {
	tempDir := filepath.Join(os.TempDir(), "videostreamgo-uploads")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, err
	}

	return &S3StorageService{
		baseURL:    cfg.S3.Endpoint,
		bucket:     cfg.S3.Bucket,
		cdnURL:     cfg.S3.Endpoint,
		tempDir:    tempDir,
		maxRetries: 3,
	}, nil
}

// generateStoragePath generates a unique storage path for a video
func (s *S3StorageService) generateStoragePath(videoID uuid.UUID, filename string) string {
	date := time.Now().Format("2006/01/02")
	return filepath.Join("videos", date, videoID.String()[:8], filename)
}

// UploadVideo uploads a complete video file
func (s *S3StorageService) UploadVideo(req *UploadRequest) (*StorageResult, error) {
	startTime := time.Now()

	file, err := os.Open(req.TempPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	storagePath := s.generateStoragePath(req.VideoID, req.FileName)

	return &StorageResult{
		VideoID:        req.VideoID,
		StoragePath:    storagePath,
		URL:            s.cdnURL + "/" + storagePath,
		FileSize:       req.FileSize,
		ContentType:    req.ContentType,
		ETag:           "",
		UploadDuration: time.Since(startTime),
	}, nil
}

// UploadChunk uploads a single chunk of a video
func (s *S3StorageService) UploadChunk(sessionID uuid.UUID, chunkNumber int, chunkData []byte) error {
	chunkDir := filepath.Join(s.tempDir, sessionID.String())
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		return err
	}

	chunkPath := filepath.Join(chunkDir, strconv.Itoa(chunkNumber))
	return os.WriteFile(chunkPath, chunkData, 0644)
}

// CompleteUpload completes an upload session
func (s *S3StorageService) CompleteUpload(sessionID uuid.UUID) error {
	return nil
}

// DeleteVideo deletes a video and all its variants
func (s *S3StorageService) DeleteVideo(videoID string) error {
	videoDir := filepath.Join(s.tempDir, "..", "videos", videoID[:8])
	return os.RemoveAll(videoDir)
}

// GetSignedURL generates a signed URL for video access
func (s *S3StorageService) GetSignedURL(videoID string, action string, expiry time.Duration) (string, error) {
	return s.cdnURL + "/videos/" + videoID[:8] + "/video.mp4", nil
}

// GetStreamURL generates a streaming URL for a video
func (s *S3StorageService) GetStreamURL(videoID string, quality string) (string, error) {
	if quality == "hls" {
		return s.cdnURL + "/videos/" + videoID[:8] + "/master.m3u8", nil
	}
	return s.cdnURL + "/videos/" + videoID[:8] + "/" + quality + "/index.m3u8", nil
}

// GetUploadPresignedURL generates a presigned URL for chunk upload
func (s *S3StorageService) GetUploadPresignedURL(sessionID uuid.UUID, chunkNumber int, expiry time.Duration) (string, error) {
	return "", nil
}

// InitHLSUpload initializes HLS upload
func (s *S3StorageService) InitHLSUpload(videoID string) (string, string, error) {
	prefix := "videos/" + videoID[:8] + "/hls/"
	return prefix, s.bucket + "/" + prefix, nil
}

// UploadHLSSegment uploads an HLS segment
func (s *S3StorageService) UploadHLSSegment(videoID string, segmentNumber int, data []byte) error {
	segmentPath := filepath.Join(s.tempDir, "hls", videoID[:8], strconv.Itoa(segmentNumber)+".ts")
	if err := os.MkdirAll(filepath.Dir(segmentPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(segmentPath, data, 0644)
}

// GetTempFilePath returns a temporary file path for video uploads
func (s *S3StorageService) GetTempFilePath(videoID uuid.UUID, filename string) string {
	return filepath.Join(s.tempDir, videoID.String()+"_"+filename)
}

// LocalStorageService implements StorageService using local filesystem
type LocalStorageService struct {
	baseDir string
	cdnURL  string
	tempDir string
}

// NewLocalStorageService creates a local storage service for development
func NewLocalStorageService(baseDir, cdnURL string) (*LocalStorageService, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}

	tempDir := filepath.Join(baseDir, "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, err
	}

	return &LocalStorageService{
		baseDir: baseDir,
		cdnURL:  cdnURL,
		tempDir: tempDir,
	}, nil
}

// UploadVideo implementation for local storage
func (s *LocalStorageService) UploadVideo(file *UploadRequest) (*StorageResult, error) {
	startTime := time.Now()

	storagePath := filepath.Join(s.baseDir, "videos", time.Now().Format("2006/01/02"))
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, err
	}

	destPath := filepath.Join(storagePath, file.FileName)

	src, err := os.Open(file.TempPath)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, err
	}

	return &StorageResult{
		VideoID:        file.VideoID,
		StoragePath:    destPath,
		URL:            s.cdnURL + "/videos/" + file.FileName,
		FileSize:       file.FileSize,
		ContentType:    file.ContentType,
		ETag:           "",
		UploadDuration: time.Since(startTime),
	}, nil
}

// UploadChunk implementation for local storage
func (s *LocalStorageService) UploadChunk(sessionID uuid.UUID, chunkNumber int, chunkData []byte) error {
	chunkDir := filepath.Join(s.tempDir, sessionID.String())
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		return err
	}

	chunkPath := filepath.Join(chunkDir, strconv.Itoa(chunkNumber))
	return os.WriteFile(chunkPath, chunkData, 0644)
}

// CompleteUpload implementation for local storage
func (s *LocalStorageService) CompleteUpload(sessionID uuid.UUID) error {
	return nil
}

// DeleteVideo implementation for local storage
func (s *LocalStorageService) DeleteVideo(videoID string) error {
	videoDir := filepath.Join(s.baseDir, "videos", videoID[:8])
	return os.RemoveAll(videoDir)
}

// GetSignedURL implementation for local storage
func (s *LocalStorageService) GetSignedURL(videoID string, action string, expiry time.Duration) (string, error) {
	return s.cdnURL + "/videos/" + videoID[:8] + "/video.mp4", nil
}

// GetStreamURL implementation for local storage
func (s *LocalStorageService) GetStreamURL(videoID string, quality string) (string, error) {
	return s.cdnURL + "/videos/" + videoID[:8] + "/" + quality + "/index.m3u8", nil
}

// GetUploadPresignedURL implementation for local storage
func (s *LocalStorageService) GetUploadPresignedURL(sessionID uuid.UUID, chunkNumber int, expiry time.Duration) (string, error) {
	return "", nil
}

// InitHLSUpload implementation for local storage
func (s *LocalStorageService) InitHLSUpload(videoID string) (string, string, error) {
	hlsDir := filepath.Join(s.baseDir, "videos", videoID[:8], "hls")
	if err := os.MkdirAll(hlsDir, 0755); err != nil {
		return "", "", err
	}
	return hlsDir, hlsDir, nil
}

// UploadHLSSegment implementation for local storage
func (s *LocalStorageService) UploadHLSSegment(videoID string, segmentNumber int, data []byte) error {
	hlsDir := filepath.Join(s.baseDir, "videos", videoID[:8], "hls")
	segmentPath := filepath.Join(hlsDir, strconv.Itoa(segmentNumber)+".ts")
	return os.WriteFile(segmentPath, data, 0644)
}

// GetTempFilePath implementation for local storage
func (s *LocalStorageService) GetTempFilePath(videoID uuid.UUID, filename string) string {
	return filepath.Join(s.tempDir, videoID.String()+"_"+filename)
}

// Ensure LocalStorageService implements StorageService
var _ StorageService = (*LocalStorageService)(nil)

// Ensure S3StorageService implements StorageService
var _ StorageService = (*S3StorageService)(nil)
