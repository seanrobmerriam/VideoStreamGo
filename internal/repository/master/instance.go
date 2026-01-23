package master

import (
	"context"

	"videostreamgo/internal/models/master"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InstanceRepository handles database operations for instances
type InstanceRepository struct {
	db *gorm.DB
}

// NewInstanceRepository creates a new InstanceRepository
func NewInstanceRepository(db *gorm.DB) *InstanceRepository {
	return &InstanceRepository{db: db}
}

// Create creates a new instance
func (r *InstanceRepository) Create(ctx context.Context, instance *master.Instance) error {
	return r.db.WithContext(ctx).Create(instance).Error
}

// GetByID retrieves an instance by ID
func (r *InstanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*master.Instance, error) {
	var instance master.Instance
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// GetBySubdomain retrieves an instance by subdomain
func (r *InstanceRepository) GetBySubdomain(ctx context.Context, subdomain string) (*master.Instance, error) {
	var instance master.Instance
	err := r.db.WithContext(ctx).Where("subdomain = ?", subdomain).First(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// GetByCustomerID retrieves all instances for a customer
func (r *InstanceRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]master.Instance, error) {
	var instances []master.Instance
	err := r.db.WithContext(ctx).Where("customer_id = ?", customerID).Find(&instances).Error
	return instances, err
}

// Update updates an instance
func (r *InstanceRepository) Update(ctx context.Context, instance *master.Instance) error {
	return r.db.WithContext(ctx).Save(instance).Error
}

// Delete soft deletes an instance
func (r *InstanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&master.Instance{}, "id = ?", id).Error
}

// List retrieves instances with pagination
func (r *InstanceRepository) List(ctx context.Context, offset, limit int, status string) ([]master.Instance, int64, error) {
	var instances []master.Instance
	var total int64

	query := r.db.WithContext(ctx).Model(&master.Instance{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&instances).Error
	if err != nil {
		return nil, 0, err
	}

	return instances, total, nil
}

// GetActiveInstances retrieves all active instances
func (r *InstanceRepository) GetActiveInstances(ctx context.Context) ([]master.Instance, error) {
	var instances []master.Instance
	err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&instances).Error
	return instances, err
}

// UpdateStatus updates the status of an instance
func (r *InstanceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&master.Instance{}).Where("id = ?", id).Update("status", status).Error
}
