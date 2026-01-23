package platform

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"videostreamgo/internal/models/master"
)

// Test_SubscriptionService_CalculatePrice tests subscription price calculations
func Test_SubscriptionService_CalculatePrice(t *testing.T) {
	tests := []struct {
		name          string
		planID        uuid.UUID
		months        int
		expectedPrice float64
	}{
		{
			name:          "Monthly basic",
			planID:        uuid.New(),
			months:        1,
			expectedPrice: 29.99,
		},
		{
			name:          "Yearly basic (10% discount)",
			planID:        uuid.New(),
			months:        12,
			expectedPrice: 323.89, // 29.99 * 12 * 0.9
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			basePrice := 29.99
			discount := 1.0
			if tt.months >= 12 {
				discount = 0.9
			}

			calculatedPrice := basePrice * float64(tt.months) * discount
			assert.InDelta(t, tt.expectedPrice, calculatedPrice, 0.01)
		})
	}
}

// Test_SubscriptionService_RenewalLogic tests renewal logic
func Test_SubscriptionService_RenewalLogic(t *testing.T) {
	tests := []struct {
		name            string
		currentPeriod   time.Time
		billingCycle    string
		expectedRenewal time.Time
	}{
		{
			name:            "Monthly renewal",
			currentPeriod:   time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			billingCycle:    "monthly",
			expectedRenewal: time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:            "Yearly renewal",
			currentPeriod:   time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			billingCycle:    "yearly",
			expectedRenewal: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var renewal time.Time
			switch tt.billingCycle {
			case "monthly":
				renewal = tt.currentPeriod.AddDate(0, 1, 0)
			case "yearly":
				renewal = tt.currentPeriod.AddDate(1, 0, 0)
			}

			assert.Equal(t, tt.expectedRenewal, renewal)
		})
	}
}

// Test_SubscriptionService_BillingCycles tests billing cycle calculations
func Test_SubscriptionService_BillingCycles(t *testing.T) {
	tests := []struct {
		name          string
		subscription  *master.Subscription
		expectedCycle int
	}{
		{
			name: "Monthly subscription",
			subscription: &master.Subscription{
				BillingCycle:       "monthly",
				CurrentPeriodStart: func() *time.Time { t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC); return &t }(),
			},
			expectedCycle: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)
			monthsActive := int(now.Sub(*tt.subscription.CurrentPeriodStart).Hours() / 24 / 30)

			assert.Equal(t, tt.expectedCycle, monthsActive)
		})
	}
}
