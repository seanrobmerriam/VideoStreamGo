package instance

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"videostreamgo/internal/models/instance"
)

// CategoryRepository handles database operations for categories
type CategoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository creates a new CategoryRepository
func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// Create creates a new category
func (r *CategoryRepository) Create(ctx context.Context, category *instance.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}

// GetByID retrieves a category by ID
func (r *CategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*instance.Category, error) {
	var category instance.Category
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// GetBySlug retrieves a category by slug
func (r *CategoryRepository) GetBySlug(ctx context.Context, slug string) (*instance.Category, error) {
	var category instance.Category
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// Update updates a category
func (r *CategoryRepository) Update(ctx context.Context, category *instance.Category) error {
	return r.db.WithContext(ctx).Save(category).Error
}

// Delete soft deletes a category
func (r *CategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&instance.Category{}, "id = ?", id).Error
}

// List retrieves categories with pagination
func (r *CategoryRepository) List(ctx context.Context, offset, limit int) ([]instance.Category, int64, error) {
	var categories []instance.Category
	var total int64

	query := r.db.WithContext(ctx).Model(&instance.Category{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Order("sort_order ASC, name ASC").Find(&categories).Error
	if err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

// GetActive retrieves active categories
func (r *CategoryRepository) GetActive(ctx context.Context) ([]instance.Category, error) {
	var categories []instance.Category
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("sort_order ASC, name ASC").Find(&categories).Error
	return categories, err
}

// GetByParentID retrieves categories by parent ID
func (r *CategoryRepository) GetByParentID(ctx context.Context, parentID uuid.UUID) ([]instance.Category, error) {
	var categories []instance.Category
	err := r.db.WithContext(ctx).Where("parent_id = ?", parentID).Order("sort_order ASC, name ASC").Find(&categories).Error
	return categories, err
}
