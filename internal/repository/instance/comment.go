package instance

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"videostreamgo/internal/models/instance"
)

// CommentRepository handles database operations for comments
type CommentRepository struct {
	db *gorm.DB
}

// NewCommentRepository creates a new CommentRepository
func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create creates a new comment
func (r *CommentRepository) Create(ctx context.Context, comment *instance.Comment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

// GetByID retrieves a comment by ID
func (r *CommentRepository) GetByID(ctx context.Context, id uuid.UUID) (*instance.Comment, error) {
	var comment instance.Comment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// Update updates a comment
func (r *CommentRepository) Update(ctx context.Context, comment *instance.Comment) error {
	return r.db.WithContext(ctx).Save(comment).Error
}

// Delete soft deletes a comment
func (r *CommentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&instance.Comment{}, "id = ?", id).Error
}

// HardDelete permanently deletes a comment
func (r *CommentRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&instance.Comment{}, "id = ?", id).Error
}

// List retrieves comments with pagination
func (r *CommentRepository) List(ctx context.Context, offset, limit int, videoID string) ([]instance.Comment, int64, error) {
	var comments []instance.Comment
	var total int64

	query := r.db.WithContext(ctx).Model(&instance.Comment{}).Where("is_deleted = ?", false)

	if videoID != "" {
		videoUUID, err := uuid.Parse(videoID)
		if err == nil {
			query = query.Where("video_id = ?", videoUUID)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&comments).Error
	if err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

// GetByVideoID retrieves all comments for a video
func (r *CommentRepository) GetByVideoID(ctx context.Context, videoID uuid.UUID) ([]instance.Comment, error) {
	var comments []instance.Comment
	err := r.db.WithContext(ctx).Where("video_id = ? AND is_deleted = ?", videoID, false).
		Order("created_at ASC").Find(&comments).Error
	return comments, err
}

// GetTopLevelByVideoID retrieves top-level comments for a video
func (r *CommentRepository) GetTopLevelByVideoID(ctx context.Context, videoID uuid.UUID, offset, limit int) ([]instance.Comment, int64, error) {
	var comments []instance.Comment
	var total int64

	query := r.db.WithContext(ctx).Model(&instance.Comment{}).
		Where("video_id = ? AND parent_id IS NULL AND is_deleted = ?", videoID, false)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&comments).Error
	if err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

// GetReplies retrieves replies to a comment
func (r *CommentRepository) GetReplies(ctx context.Context, parentID uuid.UUID) ([]instance.Comment, error) {
	var comments []instance.Comment
	err := r.db.WithContext(ctx).Where("parent_id = ? AND is_deleted = ?", parentID, false).
		Order("created_at ASC").Find(&comments).Error
	return comments, err
}

// GetByUserID retrieves comments by user ID
func (r *CommentRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]instance.Comment, error) {
	var comments []instance.Comment
	err := r.db.WithContext(ctx).Where("user_id = ? AND is_deleted = ?", userID, false).
		Order("created_at DESC").Find(&comments).Error
	return comments, err
}

// IncrementLikeCount increments the like count of a comment
func (r *CommentRepository) IncrementLikeCount(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&instance.Comment{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
}

// DecrementLikeCount decrements the like count of a comment
func (r *CommentRepository) DecrementLikeCount(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&instance.Comment{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
}

// GetCountByVideoID returns the count of comments for a video
func (r *CommentRepository) GetCountByVideoID(ctx context.Context, videoID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&instance.Comment{}).
		Where("video_id = ? AND is_deleted = ?", videoID, false).
		Count(&count).Error
	return count, err
}

// RatingRepository handles database operations for ratings
type RatingRepository struct {
	db *gorm.DB
}

// NewRatingRepository creates a new RatingRepository
func NewRatingRepository(db *gorm.DB) *RatingRepository {
	return &RatingRepository{db: db}
}

// Create creates a new rating
func (r *RatingRepository) Create(ctx context.Context, rating *instance.Rating) error {
	return r.db.WithContext(ctx).Create(rating).Error
}

// GetByID retrieves a rating by ID
func (r *RatingRepository) GetByID(ctx context.Context, id uuid.UUID) (*instance.Rating, error) {
	var rating instance.Rating
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rating).Error
	if err != nil {
		return nil, err
	}
	return &rating, nil
}

// GetByVideoAndUser retrieves a rating by video and user
func (r *RatingRepository) GetByVideoAndUser(ctx context.Context, videoID, userID uuid.UUID) (*instance.Rating, error) {
	var rating instance.Rating
	err := r.db.WithContext(ctx).Where("video_id = ? AND user_id = ?", videoID, userID).First(&rating).Error
	if err != nil {
		return nil, err
	}
	return &rating, nil
}

// Update updates a rating
func (r *RatingRepository) Update(ctx context.Context, rating *instance.Rating) error {
	return r.db.WithContext(ctx).Save(rating).Error
}

// Delete deletes a rating
func (r *RatingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&instance.Rating{}, "id = ?", id).Error
}

// GetVideoRatingStats returns the like and dislike counts for a video
func (r *RatingRepository) GetVideoRatingStats(ctx context.Context, videoID uuid.UUID) (likes, dislikes int64, err error) {
	err = r.db.WithContext(ctx).Model(&instance.Rating{}).
		Where("video_id = ? AND rating = ?", videoID, 1).Count(&likes).Error
	if err != nil {
		return
	}
	err = r.db.WithContext(ctx).Model(&instance.Rating{}).
		Where("video_id = ? AND rating = ?", videoID, -1).Count(&dislikes).Error
	return
}

// HasUserRated checks if a user has rated a video
func (r *RatingRepository) HasUserRated(ctx context.Context, videoID, userID uuid.UUID) bool {
	var count int64
	r.db.WithContext(ctx).Model(&instance.Rating{}).
		Where("video_id = ? AND user_id = ?", videoID, userID).Count(&count)
	return count > 0
}
