package instance

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"videostreamgo/internal/models/instance"
)

// VideoRepository handles database operations for videos in an instance database
type VideoRepository struct {
	db *gorm.DB
}

// NewVideoRepository creates a new VideoRepository
func NewVideoRepository(db *gorm.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

// Create creates a new video
func (r *VideoRepository) Create(ctx context.Context, video *instance.Video) error {
	return r.db.WithContext(ctx).Create(video).Error
}

// GetByID retrieves a video by ID
func (r *VideoRepository) GetByID(ctx context.Context, id uuid.UUID) (*instance.Video, error) {
	var video instance.Video
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&video).Error
	if err != nil {
		return nil, err
	}
	return &video, nil
}

// GetBySlug retrieves a video by slug
func (r *VideoRepository) GetBySlug(ctx context.Context, slug string) (*instance.Video, error) {
	var video instance.Video
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&video).Error
	if err != nil {
		return nil, err
	}
	return &video, nil
}

// Update updates a video
func (r *VideoRepository) Update(ctx context.Context, video *instance.Video) error {
	return r.db.WithContext(ctx).Save(video).Error
}

// Delete soft deletes a video
func (r *VideoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&instance.Video{}, "id = ?", id).Error
}

// List retrieves videos with pagination and filtering
func (r *VideoRepository) List(ctx context.Context, offset, limit int, status, categoryID string) ([]instance.Video, int64, error) {
	var videos []instance.Video
	var total int64

	query := r.db.WithContext(ctx).Model(&instance.Video{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&videos).Error
	if err != nil {
		return nil, 0, err
	}

	return videos, total, nil
}

// GetByUser retrieves videos by user ID
func (r *VideoRepository) GetByUser(ctx context.Context, userID uuid.UUID) ([]instance.Video, error) {
	var videos []instance.Video
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&videos).Error
	return videos, err
}

// GetFeatured retrieves featured videos
func (r *VideoRepository) GetFeatured(ctx context.Context, limit int) ([]instance.Video, error) {
	var videos []instance.Video
	err := r.db.WithContext(ctx).Where("is_featured = ? AND status = ?", true, "active").
		Limit(limit).Order("created_at DESC").Find(&videos).Error
	return videos, err
}

// GetByCategory retrieves videos by category ID
func (r *VideoRepository) GetByCategory(ctx context.Context, categoryID uuid.UUID, offset, limit int) ([]instance.Video, int64, error) {
	var videos []instance.Video
	var total int64

	query := r.db.WithContext(ctx).Model(&instance.Video{}).Where("category_id = ? AND status = ?", categoryID, "active")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&videos).Error
	if err != nil {
		return nil, 0, err
	}

	return videos, total, nil
}

// IncrementViewCount increments the view count of a video
func (r *VideoRepository) IncrementViewCount(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&instance.Video{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

// IncrementLikeCount increments the like count of a video
func (r *VideoRepository) IncrementLikeCount(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&instance.Video{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
}

// DecrementLikeCount decrements the like count of a video
func (r *VideoRepository) DecrementLikeCount(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&instance.Video{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
}

// GetVideoCountByCategory returns the count of videos in a category
func (r *VideoRepository) GetVideoCountByCategory(ctx context.Context, categoryID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&instance.Video{}).
		Where("category_id = ? AND status = ?", categoryID, "active").
		Count(&count).Error
	return count, err
}
