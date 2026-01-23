package instance

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"videostreamgo/internal/models/instance"
)

// TagRepository handles database operations for tags
type TagRepository struct {
	db *gorm.DB
}

// NewTagRepository creates a new TagRepository
func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

// Create creates a new tag
func (r *TagRepository) Create(ctx context.Context, tag *instance.Tag) error {
	return r.db.WithContext(ctx).Create(tag).Error
}

// GetByID retrieves a tag by ID
func (r *TagRepository) GetByID(ctx context.Context, id uuid.UUID) (*instance.Tag, error) {
	var tag instance.Tag
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// GetBySlug retrieves a tag by slug
func (r *TagRepository) GetBySlug(ctx context.Context, slug string) (*instance.Tag, error) {
	var tag instance.Tag
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// GetByName retrieves a tag by name
func (r *TagRepository) GetByName(ctx context.Context, name string) (*instance.Tag, error) {
	var tag instance.Tag
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// Update updates a tag
func (r *TagRepository) Update(ctx context.Context, tag *instance.Tag) error {
	return r.db.WithContext(ctx).Save(tag).Error
}

// Delete soft deletes a tag
func (r *TagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&instance.Tag{}, "id = ?", id).Error
}

// List retrieves tags with pagination
func (r *TagRepository) List(ctx context.Context, offset, limit int) ([]instance.Tag, int64, error) {
	var tags []instance.Tag
	var total int64

	query := r.db.WithContext(ctx).Model(&instance.Tag{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Order("usage_count DESC, name ASC").Find(&tags).Error
	if err != nil {
		return nil, 0, err
	}

	return tags, total, nil
}

// GetPopular retrieves popular tags by usage count
func (r *TagRepository) GetPopular(ctx context.Context, limit int) ([]instance.Tag, error) {
	var tags []instance.Tag
	err := r.db.WithContext(ctx).Order("usage_count DESC").Limit(limit).Find(&tags).Error
	return tags, err
}

// IncrementUsageCount increments the usage count of a tag
func (r *TagRepository) IncrementUsageCount(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&instance.Tag{}).Where("id = ?", id).
		UpdateColumn("usage_count", gorm.Expr("usage_count + ?", 1)).Error
}

// DecrementUsageCount decrements the usage count of a tag
func (r *TagRepository) DecrementUsageCount(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&instance.Tag{}).Where("id = ?", id).
		UpdateColumn("usage_count", gorm.Expr("GREATEST(usage_count - 1, 0)")).Error
}
