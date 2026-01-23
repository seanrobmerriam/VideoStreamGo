package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
)

// UsageBillingService handles usage-based billing
type UsageBillingService struct {
	usageRepo    *masterRepo.UsageMetricsRepository
	billingRepo  *masterRepo.BillingRecordRepository
	instanceRepo *masterRepo.InstanceRepository
	customerRepo *masterRepo.CustomerRepository
	planRepo     *masterRepo.PlanRepository
}

// NewUsageBillingService creates a new UsageBillingService
func NewUsageBillingService(
	usageRepo *masterRepo.UsageMetricsRepository,
	billingRepo *masterRepo.BillingRecordRepository,
	instanceRepo *masterRepo.InstanceRepository,
	customerRepo *masterRepo.CustomerRepository,
	planRepo *masterRepo.PlanRepository,
) *UsageBillingService {
	return &UsageBillingService{
		usageRepo:    usageRepo,
		billingRepo:  billingRepo,
		instanceRepo: instanceRepo,
		customerRepo: customerRepo,
		planRepo:     planRepo,
	}
}

// UsageSummary represents a summary of usage for billing
type UsageSummary struct {
	InstanceID       uuid.UUID `json:"instance_id"`
	InstanceName     string    `json:"instance_name"`
	CustomerID       uuid.UUID `json:"customer_id"`
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	StorageUsedGB    float64   `json:"storage_used_gb"`
	StorageLimitGB   int       `json:"storage_limit_gb"`
	BandwidthUsedGB  float64   `json:"bandwidth_used_gb"`
	BandwidthLimitGB int       `json:"bandwidth_limit_gb"`
	VideosCount      int       `json:"videos_count"`
	VideosLimit      int       `json:"videos_limit"`
	UsersCount       int       `json:"users_count"`
	UsersLimit       int       `json:"users_limit"`
	ViewsCount       int64     `json:"views_count"`
	OverageCharges   float64   `json:"overage_charges"`
	WithinLimits     bool      `json:"within_limits"`
}

// BillingPeriod represents a billing period
type BillingPeriod struct {
	Start time.Time
	End   time.Time
}

// UsageMetrics represents usage metrics for tracking
type UsageMetrics struct {
	StorageGB   float64 `json:"storage_gb"`
	BandwidthGB float64 `json:"bandwidth_gb"`
	VideosCount int     `json:"videos_count"`
	UsersCount  int     `json:"users_count"`
	ViewsCount  int64   `json:"views_count"`
}

// UsageCheckResult represents the result of checking usage against limits
type UsageCheckResult struct {
	WithinLimits      bool    `json:"within_limits"`
	StorageOverGB     float64 `json:"storage_over_gb"`
	BandwidthOverGB   float64 `json:"bandwidth_over_gb"`
	VideosOver        int     `json:"videos_over"`
	UsersOver         int     `json:"users_over"`
	EstimatedOverages float64 `json:"estimated_overages"`
}

// CalculateUsage calculates the usage summary for an instance
func (s *UsageBillingService) CalculateUsage(ctx context.Context, instanceID uuid.UUID, period BillingPeriod) (*UsageSummary, error) {
	// Get instance
	instance, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	// Get customer
	customer, err := s.customerRepo.GetByID(ctx, instance.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	// Get customer's active subscription
	subscription, err := s.getActiveSubscription(ctx, customer.ID)
	if err != nil {
		return nil, fmt.Errorf("no active subscription: %w", err)
	}

	// Get plan
	plan, err := s.planRepo.GetByID(ctx, subscription.PlanID)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}

	// Get usage metrics from instance database
	// This is a simplified version - in production, you'd query the instance database
	usage := &UsageMetrics{
		StorageGB:   50.0,  // Placeholder - would come from instance DB
		BandwidthGB: 200.0, // Placeholder - would come from instance DB
		VideosCount: 100,   // Placeholder - would come from instance DB
		UsersCount:  5,     // Placeholder - would come from instance DB
		ViewsCount:  10000, // Placeholder - would come from instance DB
	}

	// Calculate overages
	storageOver := 0.0
	if usage.StorageGB > float64(plan.MaxStorageGB) {
		storageOver = usage.StorageGB - float64(plan.MaxStorageGB)
	}

	bandwidthOver := 0.0
	if usage.BandwidthGB > float64(plan.MaxBandwidthGB) {
		bandwidthOver = usage.BandwidthGB - float64(plan.MaxBandwidthGB)
	}

	videosOver := 0
	if usage.VideosCount > plan.MaxVideos {
		videosOver = usage.VideosCount - plan.MaxVideos
	}

	usersOver := 0
	if usage.UsersCount > plan.MaxUsers {
		usersOver = usage.UsersCount - plan.MaxUsers
	}

	// Calculate overage charges (e.g., $0.10 per GB over, $1 per extra user)
	overageCharges := (storageOver * 0.10) + (bandwidthOver * 0.15) + (float64(videosOver) * 0.01) + (float64(usersOver) * 5.0)

	withinLimits := storageOver == 0 && bandwidthOver == 0 && videosOver == 0 && usersOver == 0

	summary := &UsageSummary{
		InstanceID:       instanceID,
		InstanceName:     instance.Name,
		CustomerID:       customer.ID,
		PeriodStart:      period.Start,
		PeriodEnd:        period.End,
		StorageUsedGB:    usage.StorageGB,
		StorageLimitGB:   plan.MaxStorageGB,
		BandwidthUsedGB:  usage.BandwidthGB,
		BandwidthLimitGB: plan.MaxBandwidthGB,
		VideosCount:      usage.VideosCount,
		VideosLimit:      plan.MaxVideos,
		UsersCount:       usage.UsersCount,
		UsersLimit:       plan.MaxUsers,
		ViewsCount:       usage.ViewsCount,
		OverageCharges:   overageCharges,
		WithinLimits:     withinLimits,
	}

	return summary, nil
}

// GenerateUsageInvoice generates an invoice for usage-based billing
func (s *UsageBillingService) GenerateUsageInvoice(ctx context.Context, instanceID uuid.UUID, period BillingPeriod) (*master.BillingRecord, error) {
	usage, err := s.CalculateUsage(ctx, instanceID, period)
	if err != nil {
		return nil, err
	}

	if usage.OverageCharges <= 0 {
		return nil, nil // No overages, no invoice needed
	}

	// Get instance to find customer
	instance, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	// Create billing record
	billingRecord := &master.BillingRecord{
		CustomerID:  instance.CustomerID,
		Amount:      usage.OverageCharges,
		Currency:    "USD",
		Status:      master.BillingRecordStatusPending,
		Type:        "usage",
		Description: fmt.Sprintf("Usage overages for %s - Period: %s to %s", usage.InstanceName, period.Start.Format("2006-01-02"), period.End.Format("2006-01-02")),
		PeriodStart: &period.Start,
		PeriodEnd:   &period.End,
	}

	if err := s.billingRepo.Create(ctx, billingRecord); err != nil {
		return nil, fmt.Errorf("failed to create billing record: %w", err)
	}

	return billingRecord, nil
}

// RecordUsageMetrics records usage metrics for an instance
func (s *UsageBillingService) RecordUsageMetrics(ctx context.Context, instanceID uuid.UUID, metrics *UsageMetrics, period BillingPeriod) error {
	now := time.Now()

	// Create usage records for each metric type
	usageRecords := []master.UsageMetrics{
		{
			InstanceID:  instanceID,
			MetricType:  master.MetricTypeStorage,
			PeriodStart: period.Start,
			PeriodEnd:   period.End,
			Value:       int64(metrics.StorageGB * 1024), // Convert to MB
		},
		{
			InstanceID:  instanceID,
			MetricType:  master.MetricTypeBandwidth,
			PeriodStart: period.Start,
			PeriodEnd:   period.End,
			Value:       int64(metrics.BandwidthGB * 1024), // Convert to MB
		},
		{
			InstanceID:  instanceID,
			MetricType:  master.MetricTypeVideos,
			PeriodStart: period.Start,
			PeriodEnd:   period.End,
			Value:       int64(metrics.VideosCount),
		},
		{
			InstanceID:  instanceID,
			MetricType:  master.MetricTypeUsers,
			PeriodStart: period.Start,
			PeriodEnd:   period.End,
			Value:       int64(metrics.UsersCount),
		},
		{
			InstanceID:  instanceID,
			MetricType:  master.MetricTypeViews,
			PeriodStart: period.Start,
			PeriodEnd:   period.End,
			Value:       metrics.ViewsCount,
		},
	}

	for _, record := range usageRecords {
		record.CreatedAt = now
		if err := s.usageRepo.Create(ctx, &record); err != nil {
			return fmt.Errorf("failed to record %s metrics: %w", record.MetricType, err)
		}
	}

	return nil
}

// CheckUsageLimits checks if an instance is within its usage limits
func (s *UsageBillingService) CheckUsageLimits(ctx context.Context, instanceID uuid.UUID) (*UsageCheckResult, error) {
	// Get instance
	instance, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	// Get customer's active subscription
	subscription, err := s.getActiveSubscription(ctx, instance.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("no active subscription: %w", err)
	}

	// Get plan
	plan, err := s.planRepo.GetByID(ctx, subscription.PlanID)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}

	// Get current usage (placeholder - would come from instance database)
	usage := &UsageMetrics{
		StorageGB:   50.0,
		BandwidthGB: 200.0,
		VideosCount: 100,
		UsersCount:  5,
	}

	// Calculate overages
	storageOver := 0.0
	if usage.StorageGB > float64(plan.MaxStorageGB) {
		storageOver = usage.StorageGB - float64(plan.MaxStorageGB)
	}

	bandwidthOver := 0.0
	if usage.BandwidthGB > float64(plan.MaxBandwidthGB) {
		bandwidthOver = usage.BandwidthGB - float64(plan.MaxBandwidthGB)
	}

	videosOver := 0
	if usage.VideosCount > plan.MaxVideos {
		videosOver = usage.VideosCount - plan.MaxVideos
	}

	usersOver := 0
	if usage.UsersCount > plan.MaxUsers {
		usersOver = usage.UsersCount - plan.MaxUsers
	}

	// Calculate estimated overage charges
	estimatedOverages := (storageOver * 0.10) + (bandwidthOver * 0.15) + (float64(videosOver) * 0.01) + (float64(usersOver) * 5.0)

	return &UsageCheckResult{
		WithinLimits:      storageOver == 0 && bandwidthOver == 0 && videosOver == 0 && usersOver == 0,
		StorageOverGB:     storageOver,
		BandwidthOverGB:   bandwidthOver,
		VideosOver:        videosOver,
		UsersOver:         usersOver,
		EstimatedOverages: estimatedOverages,
	}, nil
}

// GetUsageHistory retrieves usage history for an instance
func (s *UsageBillingService) GetUsageHistory(ctx context.Context, instanceID uuid.UUID, limit int) ([]master.UsageMetrics, error) {
	return s.usageRepo.GetByInstanceID(ctx, instanceID)
}

// getActiveSubscription is a helper to get the active subscription for a customer
func (s *UsageBillingService) getActiveSubscription(ctx context.Context, customerID uuid.UUID) (*master.Subscription, error) {
	// This is a simplified version - in production, you'd query the subscription repository directly
	// For now, return a placeholder
	return nil, fmt.Errorf("no active subscription found")
}
