package master

import (
	"context"
	"time"

	"videostreamgo/internal/models/master"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CustomerRepository handles database operations for customers
type CustomerRepository struct {
	db *gorm.DB
}

// NewCustomerRepository creates a new CustomerRepository
func NewCustomerRepository(db *gorm.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

// Create creates a new customer
func (r *CustomerRepository) Create(ctx context.Context, customer *master.Customer) error {
	return r.db.WithContext(ctx).Create(customer).Error
}

// GetByID retrieves a customer by ID
func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*master.Customer, error) {
	var customer master.Customer
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// GetByEmail retrieves a customer by email
func (r *CustomerRepository) GetByEmail(ctx context.Context, email string) (*master.Customer, error) {
	var customer master.Customer
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// Update updates a customer
func (r *CustomerRepository) Update(ctx context.Context, customer *master.Customer) error {
	return r.db.WithContext(ctx).Save(customer).Error
}

// Delete soft deletes a customer
func (r *CustomerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&master.Customer{}, "id = ?", id).Error
}

// List retrieves customers with pagination
func (r *CustomerRepository) List(ctx context.Context, offset, limit int, status string) ([]master.Customer, int64, error) {
	var customers []master.Customer
	var total int64

	query := r.db.WithContext(ctx).Model(&master.Customer{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&customers).Error
	if err != nil {
		return nil, 0, err
	}

	return customers, total, nil
}

// GetByStatus retrieves customers by status
func (r *CustomerRepository) GetByStatus(ctx context.Context, status string) ([]master.Customer, error) {
	var customers []master.Customer
	err := r.db.WithContext(ctx).Where("status = ?", status).Find(&customers).Error
	return customers, err
}

// GetByStripeCustomerID retrieves a customer by Stripe customer ID
func (r *CustomerRepository) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*master.Customer, error) {
	var customer master.Customer
	err := r.db.WithContext(ctx).Where("stripe_customer_id = ?", stripeCustomerID).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// GetActiveCount returns the count of active customers
func (r *CustomerRepository) GetActiveCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&master.Customer{}).Where("status = ?", master.CustomerStatusActive).Count(&count).Error
	return count, err
}

// GetNewCustomersCount returns the count of new customers in a time period
func (r *CustomerRepository) GetNewCustomersCount(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&master.Customer{}).Where("created_at >= ?", since).Count(&count).Error
	return count, err
}
