package instance

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"videostreamgo/internal/models/instance"
)

// StorageService handles video storage operations
type StorageService interface {
	UploadVideo(file *UploadRequest) (*StorageResult, error)
	UploadChunk(sessionID uuid.UUID, chunkNumber int, chunkData []byte) error
	CompleteUpload(sessionID uuid.UUID) error
	DeleteVideo(videoID string) error
	GetSignedURL(videoID string, action string, expiry time.Duration) (string, error)
	GetStreamURL(videoID string, quality string) (string, error)
	GetUploadPresignedURL(sessionID uuid.UUID, chunkNumber int, expiry time.Duration) (string, error)
	InitHLSUpload(videoID string) (string, string, error)
	UploadHLSSegment(videoID string, segmentNumber int, data []byte) error
	GetTempFilePath(videoID uuid.UUID, filename string) string
}

// VideoProcessingService handles video transcoding and thumbnail generation
type VideoProcessingService interface {
	ProcessVideo(videoID string) error
	GenerateThumbnails(videoID string) error
	TranscodeVideo(videoID string, qualities []instance.VideoQuality) error
	ExtractMetadata(videoID string) (*VideoMetadata, error)
	CleanupTempFiles(videoID string) error
	GetProcessingStatus(videoID string) (*ProcessingStatusResponse, error)
}

// UploadRequest represents a video upload request
type UploadRequest struct {
	VideoID     uuid.UUID
	UserID      uuid.UUID
	FileName    string
	ContentType string
	FileSize    int64
	ChunkSize   int64
	TotalChunks int
	FileHash    string
	TempPath    string
}

// StorageResult represents the result of a storage operation
type StorageResult struct {
	VideoID        uuid.UUID
	StoragePath    string
	URL            string
	FileSize       int64
	ContentType    string
	ETag           string
	UploadDuration time.Duration
}

// VideoMetadata represents extracted video metadata
type VideoMetadata struct {
	Duration    float64 `json:"duration"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Bitrate     int     `json:"bitrate"`
	Codec       string  `json:"codec"`
	AudioCodec  string  `json:"audio_codec"`
	FrameRate   float64 `json:"frame_rate"`
	AspectRatio string  `json:"aspect_ratio"`
}

// ProcessingStatusResponse represents the processing status response
type ProcessingStatusResponse struct {
	VideoID       uuid.UUID                 `json:"video_id"`
	Status        instance.ProcessingStatus `json:"status"`
	Progress      int                       `json:"progress"`
	CurrentStep   string                    `json:"current_step"`
	EstimatedTime time.Duration             `json:"estimated_time,omitempty"`
	Error         string                    `json:"error,omitempty"`
}

// VideoRepositoryInterface defines the video repository interface for handlers
type VideoRepositoryInterface interface {
	Create(ctx context.Context, video *instance.Video) error
	GetByID(ctx context.Context, id uuid.UUID) (*instance.Video, error)
	Update(ctx context.Context, video *instance.Video) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// GORMVideoRepository wraps GORM operations for video repository
type GORMVideoRepository struct {
	db *gorm.DB
}

// NewGORMVideoRepository creates a new GORM video repository
func NewGORMVideoRepository(db *gorm.DB) *GORMVideoRepository {
	return &GORMVideoRepository{db: db}
}

// Create creates a new video
func (r *GORMVideoRepository) Create(ctx context.Context, video *instance.Video) error {
	return r.db.WithContext(ctx).Create(video).Error
}

// GetByID retrieves a video by ID
func (r *GORMVideoRepository) GetByID(ctx context.Context, id uuid.UUID) (*instance.Video, error) {
	var video instance.Video
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&video).Error
	if err != nil {
		return nil, err
	}
	return &video, nil
}

// Update updates a video
func (r *GORMVideoRepository) Update(ctx context.Context, video *instance.Video) error {
	return r.db.WithContext(ctx).Save(video).Error
}

// Delete soft deletes a video
func (r *GORMVideoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&instance.Video{}, "id = ?", id).Error
}

// Ensure GORMVideoRepository implements VideoRepositoryInterface
var _ VideoRepositoryInterface = (*GORMVideoRepository)(nil)
