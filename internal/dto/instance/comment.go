package instance

import (
	"github.com/google/uuid"

	"videostreamgo/internal/models/instance"
)

// CreateCommentRequest represents a request to create a new comment
type CreateCommentRequest struct {
	VideoID  string `json:"video_id" binding:"required,uuid"`
	ParentID string `json:"parent_id" binding:"omitempty,uuid"`
	Content  string `json:"content" binding:"required,min=1,max=5000"`
}

// UpdateCommentRequest represents a request to update a comment
type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=5000"`
}

// CommentResponse represents a comment in API responses
type CommentResponse struct {
	ID         uuid.UUID         `json:"id"`
	VideoID    uuid.UUID         `json:"video_id"`
	UserID     uuid.UUID         `json:"user_id"`
	Username   string            `json:"username"`
	UserAvatar string            `json:"user_avatar,omitempty"`
	ParentID   *uuid.UUID        `json:"parent_id,omitempty"`
	Content    string            `json:"content"`
	LikeCount  int               `json:"like_count"`
	Replies    []CommentResponse `json:"replies,omitempty"`
	CreatedAt  string            `json:"created_at"`
	UpdatedAt  string            `json:"updated_at"`
}

// ToCommentResponse converts a Comment to CommentResponse
func ToCommentResponse(comment *instance.Comment, username, userAvatar string) CommentResponse {
	return CommentResponse{
		ID:         comment.ID,
		VideoID:    comment.VideoID,
		UserID:     comment.UserID,
		Username:   username,
		UserAvatar: userAvatar,
		ParentID:   comment.ParentID,
		Content:    comment.Content,
		LikeCount:  comment.LikeCount,
		CreatedAt:  comment.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  comment.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// CommentListResponse represents a list of comments with pagination
type CommentListResponse struct {
	Comments []CommentResponse `json:"comments"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PerPage  int               `json:"per_page"`
}

// RateVideoRequest represents a request to rate a video
type RateVideoRequest struct {
	VideoID string `json:"video_id" binding:"required,uuid"`
	Rating  int    `json:"rating" binding:"required,oneof=1 -1 0"` // 1 = like, -1 = dislike, 0 = remove rating
}

// FavoriteVideoRequest represents a request to favorite/unfavorite a video
type FavoriteVideoRequest struct {
	VideoID string `json:"video_id" binding:"required,uuid"`
}

// FavoriteResponse represents a favorite in API responses
type FavoriteResponse struct {
	ID         uuid.UUID `json:"id"`
	VideoID    uuid.UUID `json:"video_id"`
	VideoTitle string    `json:"video_title"`
	CreatedAt  string    `json:"created_at"`
}

// FavoriteListResponse represents a list of favorites with pagination
type FavoriteListResponse struct {
	Favorites []FavoriteResponse `json:"favorites"`
	Total     int64              `json:"total"`
	Page      int                `json:"page"`
	PerPage   int                `json:"per_page"`
}
