package master

import (
	"context"
	"time"

	"videostreamgo/internal/models/master"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BillingRecordRepository handles database operations for billing records
type BillingRecordRepository struct {
	db *gorm.DB
}

// NewBillingRecordRepository creates a new BillingRecordRepository
func NewBillingRecordRepository(db *gorm.DB) *BillingRecordRepository {
	return &BillingRecordRepository{db: db}
}

// Create creates a new billing record
func (r *BillingRecordRepository) Create(ctx context.Context, record *master.BillingRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// GetByID retrieves a billing record by ID
func (r *BillingRecordRepository) GetByID(ctx context.Context, id uuid.UUID) (*master.BillingRecord, error) {
	var record master.BillingRecord
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByCustomerID retrieves billing records by customer ID
func (r *BillingRecordRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]master.BillingRecord, error) {
	var records []master.BillingRecord
	err := r.db.WithContext(ctx).Where("customer_id = ?", customerID).Order("created_at DESC").Find(&records).Error
	return records, err
}

// Update updates a billing record
func (r *BillingRecordRepository) Update(ctx context.Context, record *master.BillingRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

// List retrieves billing records with pagination
func (r *BillingRecordRepository) List(ctx context.Context, offset, limit int, status string) ([]master.BillingRecord, int64, error) {
	var records []master.BillingRecord
	var total int64

	query := r.db.WithContext(ctx).Model(&master.BillingRecord{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&records).Error
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// GetTotalRevenue retrieves total revenue
func (r *BillingRecordRepository) GetTotalRevenue(ctx context.Context) (float64, error) {
	var total float64
	err := r.db.WithContext(ctx).Model(&master.BillingRecord{}).
		Where("status = ?", "paid").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}

// ListByCustomer retrieves billing records for a specific customer with pagination
func (r *BillingRecordRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID, offset, limit int) ([]master.BillingRecord, int64, error) {
	var records []master.BillingRecord
	var total int64

	query := r.db.WithContext(ctx).Model(&master.BillingRecord{}).Where("customer_id = ?", customerID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&records).Error
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// GetRevenueByPeriod retrieves revenue for a specific time period
func (r *BillingRecordRepository) GetRevenueByPeriod(ctx context.Context, start, end time.Time) (float64, error) {
	var total float64
	err := r.db.WithContext(ctx).Model(&master.BillingRecord{}).
		Where("status = ? AND created_at >= ? AND created_at <= ?", "paid", start, end).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}

// GetRevenueByPlan returns revenue grouped by plan
func (r *BillingRecordRepository) GetRevenueByPlan(ctx context.Context) (map[string]float64, error) {
	// This would typically join with subscriptions and plans tables
	// For now, return empty map
	return make(map[string]float64), nil
}

// GetMonthlyRevenue returns monthly revenue for the last N months
func (r *BillingRecordRepository) GetMonthlyRevenue(ctx context.Context, months int) (map[string]float64, error) {
	// This would typically group by month
	// For now, return empty map
	return make(map[string]float64), nil
}

// GetRevenueByStatus returns revenue grouped by status
func (r *BillingRecordRepository) GetRevenueByStatus(ctx context.Context) (map[string]float64, error) {
	var result []struct {
		Status string
		Total  float64
	}

	err := r.db.WithContext(ctx).Model(&master.BillingRecord{}).
		Select("status, COALESCE(SUM(amount), 0) as total").
		Group("status").
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	revenueByStatus := make(map[string]float64)
	for _, r := range result {
		revenueByStatus[r.Status] = r.Total
	}

	return revenueByStatus, nil
}

// UsageMetricsRepository handles database operations for usage metrics
type UsageMetricsRepository struct {
	db *gorm.DB
}

// NewUsageMetricsRepository creates a new UsageMetricsRepository
func NewUsageMetricsRepository(db *gorm.DB) *UsageMetricsRepository {
	return &UsageMetricsRepository{db: db}
}

// Create creates a new usage metrics record
func (r *UsageMetricsRepository) Create(ctx context.Context, metrics *master.UsageMetrics) error {
	return r.db.WithContext(ctx).Create(metrics).Error
}

// GetByInstanceID retrieves usage metrics by instance ID
func (r *UsageMetricsRepository) GetByInstanceID(ctx context.Context, instanceID uuid.UUID) ([]master.UsageMetrics, error) {
	var metrics []master.UsageMetrics
	err := r.db.WithContext(ctx).Where("instance_id = ?", instanceID).Order("created_at DESC").Find(&metrics).Error
	return metrics, err
}

// GetLatestByInstanceID retrieves latest usage metrics for an instance
func (r *UsageMetricsRepository) GetLatestByInstanceID(ctx context.Context, instanceID uuid.UUID) (*master.UsageMetrics, error) {
	var metrics master.UsageMetrics
	err := r.db.WithContext(ctx).Where("instance_id = ?", instanceID).Order("created_at DESC").First(&metrics).Error
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}
