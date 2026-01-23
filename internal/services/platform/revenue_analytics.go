package platform

import (
	"context"
	"fmt"
	"time"

	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
)

// RevenueAnalyticsService handles revenue analytics calculations
type RevenueAnalyticsService struct {
	billingRepo      *masterRepo.BillingRecordRepository
	customerRepo     *masterRepo.CustomerRepository
	subscriptionRepo *masterRepo.SubscriptionRepository
	planRepo         *masterRepo.PlanRepository
}

// NewRevenueAnalyticsService creates a new RevenueAnalyticsService
func NewRevenueAnalyticsService(
	billingRepo *masterRepo.BillingRecordRepository,
	customerRepo *masterRepo.CustomerRepository,
	subscriptionRepo *masterRepo.SubscriptionRepository,
	planRepo *masterRepo.PlanRepository,
) *RevenueAnalyticsService {
	return &RevenueAnalyticsService{
		billingRepo:      billingRepo,
		customerRepo:     customerRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
	}
}

// RevenueAnalytics represents revenue analytics data
type RevenueAnalytics struct {
	MRR                 float64            `json:"mrr"` // Monthly Recurring Revenue
	ARR                 float64            `json:"arr"` // Annual Recurring Revenue
	TotalRevenue        float64            `json:"total_revenue"`
	RevenueByPlan       map[string]float64 `json:"revenue_by_plan"`
	RevenueByMonth      map[string]float64 `json:"revenue_by_month"`
	ChurnRate           float64            `json:"churn_rate"`
	AverageLTV          float64            `json:"average_ltv"`
	NewCustomers        int64              `json:"new_customers"`
	ActiveCustomers     int64              `json:"active_customers"`
	ActiveSubscriptions int64              `json:"active_subscriptions"`
	PeriodStart         time.Time          `json:"period_start"`
	PeriodEnd           time.Time          `json:"period_end"`
}

// CalculateMRR calculates the Monthly Recurring Revenue
func (s *RevenueAnalyticsService) CalculateMRR(ctx context.Context) (float64, error) {
	subscriptions, err := s.subscriptionRepo.GetActiveSubscriptions(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get active subscriptions: %w", err)
	}

	mrr := 0.0
	for _, sub := range subscriptions {
		plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
		if err != nil {
			continue
		}
		if sub.BillingCycle == master.BillingCycleYearly {
			mrr += plan.YearlyPrice / 12
		} else {
			mrr += plan.MonthlyPrice
		}
	}

	return mrr, nil
}

// CalculateARR calculates the Annual Recurring Revenue
func (s *RevenueAnalyticsService) CalculateARR(ctx context.Context) (float64, error) {
	mrr, err := s.CalculateMRR(ctx)
	if err != nil {
		return 0, err
	}
	return mrr * 12, nil
}

// CalculateChurnRate calculates the subscription churn rate
func (s *RevenueAnalyticsService) CalculateChurnRate(ctx context.Context) (float64, error) {
	subscriptions, err := s.subscriptionRepo.GetActiveSubscriptions(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get subscriptions: %w", err)
	}

	totalSubscriptions := len(subscriptions)
	if totalSubscriptions == 0 {
		return 0, nil
	}

	cancelledCount := 0
	for _, sub := range subscriptions {
		if sub.Status == master.SubscriptionStatusCancelled {
			cancelledCount++
		}
	}

	churnRate := float64(cancelledCount) / float64(totalSubscriptions) * 100
	return churnRate, nil
}

// CalculateLTV estimates the Lifetime Value of a customer
func (s *RevenueAnalyticsService) CalculateLTV(averageMRR float64, churnRate float64) float64 {
	if churnRate <= 0 {
		return 0
	}
	return averageMRR / (churnRate / 100)
}

// GetRevenueByPlanTier returns revenue grouped by plan tier
func (s *RevenueAnalyticsService) GetRevenueByPlanTier(ctx context.Context) (map[string]float64, error) {
	revenueByPlan := make(map[string]float64)

	subscriptions, err := s.subscriptionRepo.GetActiveSubscriptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}

	for _, sub := range subscriptions {
		plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
		if err != nil {
			continue
		}
		revenueByPlan[plan.Name] += plan.MonthlyPrice
	}

	return revenueByPlan, nil
}

// GetRevenueTrends returns revenue trends over time
func (s *RevenueAnalyticsService) GetRevenueTrends(ctx context.Context, months int) (map[string]float64, error) {
	trends := make(map[string]float64)

	endDate := time.Now()
	for i := 0; i < months; i++ {
		monthStart := endDate.AddDate(0, -i, 0)
		monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, monthStart.Location())
		monthEnd := monthStart.AddDate(0, 1, 0)

		revenue, err := s.billingRepo.GetRevenueByPeriod(ctx, monthStart, monthEnd)
		if err != nil {
			continue
		}

		monthKey := monthStart.Format("2006-01")
		trends[monthKey] = revenue
	}

	return trends, nil
}

// GetAnalytics returns comprehensive revenue analytics
func (s *RevenueAnalyticsService) GetAnalytics(ctx context.Context, months int) (*RevenueAnalytics, error) {
	now := time.Now()
	periodStart := now.AddDate(0, -months, 0)

	mrr, err := s.CalculateMRR(ctx)
	if err != nil {
		return nil, err
	}

	arr, err := s.CalculateARR(ctx)
	if err != nil {
		return nil, err
	}

	churnRate, err := s.CalculateChurnRate(ctx)
	if err != nil {
		return nil, err
	}

	ltv := s.CalculateLTV(mrr, churnRate)

	revenueByPlan, err := s.GetRevenueByPlanTier(ctx)
	if err != nil {
		return nil, err
	}

	revenueByMonth, err := s.GetRevenueTrends(ctx, months)
	if err != nil {
		return nil, err
	}

	totalRevenue, err := s.billingRepo.GetTotalRevenue(ctx)
	if err != nil {
		return nil, err
	}

	activeCustomers, _ := s.customerRepo.GetActiveCount(ctx)
	newCustomers, _ := s.customerRepo.GetNewCustomersCount(ctx, periodStart)
	subscriptions, _ := s.subscriptionRepo.GetActiveSubscriptions(ctx)

	return &RevenueAnalytics{
		MRR:                 mrr,
		ARR:                 arr,
		TotalRevenue:        totalRevenue,
		RevenueByPlan:       revenueByPlan,
		RevenueByMonth:      revenueByMonth,
		ChurnRate:           churnRate,
		AverageLTV:          ltv,
		NewCustomers:        newCustomers,
		ActiveCustomers:     activeCustomers,
		ActiveSubscriptions: int64(len(subscriptions)),
		PeriodStart:         periodStart,
		PeriodEnd:           now,
	}, nil
}
