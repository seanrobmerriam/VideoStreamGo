package instance

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VideoStatus represents the status of a video
type VideoStatus string

const (
	VideoStatusPending     VideoStatus = "pending"
	VideoStatusProcessing  VideoStatus = "processing"
	VideoStatusTranscoding VideoStatus = "transcoding"
	VideoStatusReady       VideoStatus = "ready"
	VideoStatusActive      VideoStatus = "active"
	VideoStatusHidden      VideoStatus = "hidden"
	VideoStatusFailed      VideoStatus = "failed"
	VideoStatusDeleted     VideoStatus = "deleted"
)

// ProcessingStatus represents the processing state
type ProcessingStatus string

const (
	ProcessingStatusPending     ProcessingStatus = "pending"
	ProcessingStatusUploading   ProcessingStatus = "uploading"
	ProcessingStatusUploaded    ProcessingStatus = "uploaded"
	ProcessingStatusExtracting  ProcessingStatus = "extracting_metadata"
	ProcessingStatusTranscoding ProcessingStatus = "transcoding"
	ProcessingStatusGenerating  ProcessingStatus = "generating_thumbnails"
	ProcessingStatusCompleted   ProcessingStatus = "completed"
	ProcessingStatusFailed      ProcessingStatus = "failed"
)

// VideoQuality represents available video qualities
type VideoQuality string

const (
	Quality360p  VideoQuality = "360p"
	Quality480p  VideoQuality = "480p"
	Quality720p  VideoQuality = "720p"
	Quality1080p VideoQuality = "1080p"
	Quality4K    VideoQuality = "4k"
)

// Video represents an uploaded video
type Video struct {
	ID                 uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID         uuid.UUID        `gorm:"type:uuid;not null;index" json:"instance_id"`
	Title              string           `gorm:"type:varchar(255);not null" json:"title"`
	Slug               string           `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Description        string           `gorm:"type:text" json:"description"`
	UserID             uuid.UUID        `gorm:"type:uuid;not null;index" json:"user_id"`
	CategoryID         *uuid.UUID       `gorm:"type:uuid;index" json:"category_id"`
	Status             VideoStatus      `gorm:"type:varchar(50);default:'pending';index" json:"status"`
	VideoURL           string           `gorm:"type:varchar(500);not null" json:"video_url"`
	ThumbnailURL       string           `gorm:"type:varchar(500)" json:"thumbnail_url"`
	HLSPath            string           `gorm:"type:varchar(500)" json:"hls_path"`
	DashPath           string           `gorm:"type:varchar(500)" json:"dash_path"`
	Duration           float64          `gorm:"default:0" json:"duration"`
	FileSize           int64            `gorm:"default:0" json:"file_size"`
	Resolution         string           `gorm:"type:varchar(20)" json:"resolution"`
	ResolutionLabel    string           `gorm:"type:varchar(20)" json:"resolution_label"`
	Bitrate            int              `gorm:"default:0" json:"bitrate"`
	Codec              string           `gorm:"type:varchar(50)" json:"codec"`
	AudioCodec         string           `gorm:"type:varchar(50)" json:"audio_codec"`
	FrameRate          float64          `gorm:"default:0" json:"frame_rate"`
	ProcessingStatus   ProcessingStatus `gorm:"type:varchar(50);default:'pending'" json:"processing_status"`
	ProcessingProgress int              `gorm:"default:0" json:"processing_progress"`
	ProcessingError    string           `gorm:"type:text" json:"processing_error,omitempty"`
	ViewCount          int64            `gorm:"default:0;index" json:"view_count"`
	LikeCount          int              `gorm:"default:0" json:"like_count"`
	DislikeCount       int              `gorm:"default:0" json:"dislike_count"`
	CommentCount       int              `gorm:"default:0" json:"comment_count"`
	IsFeatured         bool             `gorm:"default:false;index" json:"is_featured"`
	IsPublic           bool             `gorm:"default:true" json:"is_public"`
	PublishedAt        *time.Time       `json:"published_at"`
	Metadata           JSONMap          `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt          time.Time        `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt          time.Time        `gorm:"autoUpdateTime" json:"updated_at"`

	// Soft delete
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User      User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Category  *Category  `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Tags      []Tag      `gorm:"many2many:video_tags;" json:"tags,omitempty"`
	Comments  []Comment  `gorm:"foreignKey:VideoID" json:"comments,omitempty"`
	Ratings   []Rating   `gorm:"foreignKey:VideoID" json:"ratings,omitempty"`
	Favorites []Favorite `gorm:"foreignKey:VideoID" json:"favorites,omitempty"`
}

// TableName sets the table name for Video
func (Video) TableName() string {
	return "videos"
}

// BeforeCreate generates a UUID before creating a new Video
func (v *Video) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

// VideoQualityVariant represents a transcoded quality variant
type VideoQualityVariant struct {
	ID        uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	VideoID   uuid.UUID    `gorm:"type:uuid;not null;index" json:"video_id"`
	Quality   VideoQuality `gorm:"type:varchar(20);not null" json:"quality"`
	Width     int          `gorm:"not null" json:"width"`
	Height    int          `gorm:"not null" json:"height"`
	Bitrate   int          `gorm:"not null" json:"bitrate"`
	FileSize  int64        `gorm:"not null" json:"file_size"`
	FilePath  string       `gorm:"type:varchar(500);not null" json:"file_path"`
	FrameRate float64      `gorm:"default:0" json:"frame_rate"`
	CreatedAt time.Time    `gorm:"autoCreateTime" json:"created_at"`
}

// TableName sets the table name for VideoQualityVariant
func (VideoQualityVariant) TableName() string {
	return "video_quality_variants"
}

// VideoThumbnail represents generated thumbnails
type VideoThumbnail struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	VideoID    uuid.UUID `gorm:"type:uuid;not null;index" json:"video_id"`
	Width      int       `gorm:"not null" json:"width"`
	Height     int       `gorm:"not null" json:"height"`
	TimeOffset float64   `gorm:"not null" json:"time_offset"`
	URL        string    `gorm:"type:varchar(500);not null" json:"url"`
	IsDefault  bool      `gorm:"default:false" json:"is_default"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName sets the table name for VideoThumbnail
func (VideoThumbnail) TableName() string {
	return "video_thumbnails"
}

// UploadSession represents an upload session for chunked uploads
type UploadSession struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	VideoID        uuid.UUID `gorm:"type:uuid;not null;index" json:"video_id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	TotalChunks    int       `gorm:"not null" json:"total_chunks"`
	UploadedChunks int       `gorm:"default:0" json:"uploaded_chunks"`
	ChunkSize      int64     `gorm:"not null" json:"chunk_size"`
	TotalSize      int64     `gorm:"not null" json:"total_size"`
	FileName       string    `gorm:"type:varchar(255);not null" json:"file_name"`
	ContentType    string    `gorm:"type:varchar(100)" json:"content_type"`
	FileHash       string    `gorm:"type:varchar(64)" json:"file_hash"`
	Status         string    `gorm:"type:varchar(50);default:'pending'" json:"status"`
	ExpiresAt      time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName sets the table name for UploadSession
func (UploadSession) TableName() string {
	return "upload_sessions"
}

// VideoView represents a video view event for analytics
type VideoView struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"instance_id"`
	VideoID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"video_id"`
	UserID        *uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	IPAddress     string     `gorm:"type:inet" json:"ip_address"`
	UserAgent     string     `gorm:"type:text" json:"user_agent"`
	Referrer      string     `gorm:"type:text" json:"referrer"`
	CountryCode   string     `gorm:"type:char(2)" json:"country_code"`
	WatchDuration int        `gorm:"default:0" json:"watch_duration"`
	CreatedAt     time.Time  `gorm:"autoCreateTime;index" json:"created_at"`
}

// TableName sets the table name for VideoView
func (VideoView) TableName() string {
	return "video_views"
}

// BeforeCreate generates a UUID before creating a new VideoView
func (vv *VideoView) BeforeCreate(tx *gorm.DB) error {
	if vv.ID == uuid.Nil {
		vv.ID = uuid.New()
	}
	return nil
}
