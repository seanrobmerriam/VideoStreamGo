package instance

import (
	"github.com/google/uuid"

	"videostreamgo/internal/models/instance"
)

// CreateVideoRequest represents a request to create a new video
type CreateVideoRequest struct {
	Title        string   `json:"title" binding:"required,min=3,max=255"`
	Description  string   `json:"description" binding:"max=5000"`
	CategoryID   string   `json:"category_id" binding:"omitempty,uuid"`
	VideoURL     string   `json:"video_url" binding:"required,url"`
	ThumbnailURL string   `json:"thumbnail_url" binding:"omitempty,url"`
	Duration     float64  `json:"duration" binding:"gte=0"`
	Resolution   string   `json:"resolution" binding:"omitempty,oneof=360p 480p 720p 1080p 4k"`
	IsPublic     *bool    `json:"is_public"`
	Tags         []string `json:"tags"`
}

// UpdateVideoRequest represents a request to update a video
type UpdateVideoRequest struct {
	Title        *string  `json:"title" binding:"omitempty,min=3,max=255"`
	Description  *string  `json:"description" binding:"omitempty,max=5000"`
	CategoryID   *string  `json:"category_id" binding:"omitempty,uuid"`
	ThumbnailURL *string  `json:"thumbnail_url" binding:"omitempty,url"`
	Duration     *float64 `json:"duration" binding:"omitempty,gte=0"`
	Resolution   *string  `json:"resolution" binding:"omitempty,oneof=360p 480p 720p 1080p 4k"`
	IsPublic     *bool    `json:"is_public"`
	Status       *string  `json:"status" binding:"omitempty,oneof=processing active hidden deleted"`
}

// VideoResponse represents a video in API responses
type VideoResponse struct {
	ID               uuid.UUID                 `json:"id"`
	Title            string                    `json:"title"`
	Slug             string                    `json:"slug"`
	Description      string                    `json:"description"`
	UserID           uuid.UUID                 `json:"user_id"`
	Username         string                    `json:"username"`
	CategoryID       *uuid.UUID                `json:"category_id,omitempty"`
	CategoryName     string                    `json:"category_name,omitempty"`
	Status           instance.VideoStatus      `json:"status"`
	VideoURL         string                    `json:"video_url"`
	ThumbnailURL     string                    `json:"thumbnail_url"`
	Duration         float64                   `json:"duration"`
	Resolution       string                    `json:"resolution"`
	ViewCount        int64                     `json:"view_count"`
	LikeCount        int                       `json:"like_count"`
	DislikeCount     int                       `json:"dislike_count"`
	CommentCount     int                       `json:"comment_count"`
	IsFeatured       bool                      `json:"is_featured"`
	IsPublic         bool                      `json:"is_public"`
	PublishedAt      *string                   `json:"published_at,omitempty"`
	CreatedAt        string                    `json:"created_at"`
	UpdatedAt        string                    `json:"updated_at"`
	ProcessingStatus instance.ProcessingStatus `json:"processing_status,omitempty"`
}

// ToVideoResponse converts a Video to VideoResponse
func ToVideoResponse(video *instance.Video, username, categoryName string) VideoResponse {
	resp := VideoResponse{
		ID:               video.ID,
		Title:            video.Title,
		Slug:             video.Slug,
		Description:      video.Description,
		UserID:           video.UserID,
		Username:         username,
		CategoryID:       video.CategoryID,
		Status:           video.Status,
		VideoURL:         video.VideoURL,
		ThumbnailURL:     video.ThumbnailURL,
		Duration:         video.Duration,
		Resolution:       video.Resolution,
		ViewCount:        video.ViewCount,
		LikeCount:        video.LikeCount,
		DislikeCount:     video.DislikeCount,
		CommentCount:     video.CommentCount,
		IsFeatured:       video.IsFeatured,
		IsPublic:         video.IsPublic,
		ProcessingStatus: video.ProcessingStatus,
		CreatedAt:        video.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        video.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if categoryName != "" {
		resp.CategoryName = categoryName
	}
	if video.PublishedAt != nil {
		published := video.PublishedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.PublishedAt = &published
	}
	return resp
}

// VideoListResponse represents a list of videos with pagination
type VideoListResponse struct {
	Videos  []VideoResponse `json:"videos"`
	Total   int64           `json:"total"`
	Page    int             `json:"page"`
	PerPage int             `json:"per_page"`
}

// RecordViewRequest represents a request to record a video view
type RecordViewRequest struct {
	WatchDuration int `json:"watch_duration" binding:"gte=0"`
}

// VideoAnalyticsResponse represents video analytics data
type VideoAnalyticsResponse struct {
	VideoID        uuid.UUID `json:"video_id"`
	TotalViews     int64     `json:"total_views"`
	UniqueViewers  int64     `json:"unique_viewers"`
	TotalWatchTime int64     `json:"total_watch_time"`
	AvgWatchTime   float64   `json:"avg_watch_time"`
	LikeCount      int       `json:"like_count"`
	DislikeCount   int       `json:"dislike_count"`
	CommentCount   int       `json:"comment_count"`
}
