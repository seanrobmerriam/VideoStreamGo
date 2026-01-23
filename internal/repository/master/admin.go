package master

import (
	"context"

	"videostreamgo/internal/models/master"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminRepository handles database operations for admin users
type AdminRepository struct {
	db *gorm.DB
}

// NewAdminRepository creates a new AdminRepository
func NewAdminRepository(db *gorm.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// Create creates a new admin user
func (r *AdminRepository) Create(ctx context.Context, admin *master.AdminUser) error {
	return r.db.WithContext(ctx).Create(admin).Error
}

// GetByID retrieves an admin user by ID
func (r *AdminRepository) GetByID(ctx context.Context, id uuid.UUID) (*master.AdminUser, error) {
	var admin master.AdminUser
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// GetByEmail retrieves an admin user by email
func (r *AdminRepository) GetByEmail(ctx context.Context, email string) (*master.AdminUser, error) {
	var admin master.AdminUser
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// Update updates an admin user
func (r *AdminRepository) Update(ctx context.Context, admin *master.AdminUser) error {
	return r.db.WithContext(ctx).Save(admin).Error
}

// Delete soft deletes an admin user
func (r *AdminRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&master.AdminUser{}, "id = ?", id).Error
}

// List retrieves admin users with pagination
func (r *AdminRepository) List(ctx context.Context, offset, limit int, status string) ([]master.AdminUser, int64, error) {
	var admins []master.AdminUser
	var total int64

	query := r.db.WithContext(ctx).Model(&master.AdminUser{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&admins).Error
	if err != nil {
		return nil, 0, err
	}

	return admins, total, nil
}

// UpdateLastLogin updates the last login time
func (r *AdminRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&master.AdminUser{}).Where("id = ?", id).Update("last_login_at", "now()").Error
}
