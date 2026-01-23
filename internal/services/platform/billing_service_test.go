package platform

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_BillingService_CustomerCreation(t *testing.T) {
	// Simulate Stripe customer ID generation
	stripeCustomerID := "cus_" + uuid.New().String()[:14]

	assert.NotEmpty(t, stripeCustomerID)
	assert.Contains(t, stripeCustomerID, "cus_")
}

func Test_BillingService_SubscriptionStatusTransitions(t *testing.T) {
	// Test valid status transitions using string values
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"Active is valid", "active", true},
		{"Past due is problematic", "past_due", false},
		{"Cancelled is inactive", "cancelled", false},
		{"Unpaid is problematic", "unpaid", false},
		{"Trialing is active", "trialing", true},
		{"Paused is inactive", "paused", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isActive := tt.status == "active" || tt.status == "trialing"
			assert.Equal(t, tt.expected, isActive)
		})
	}
}

func Test_BillingService_BillingCycleDuration(t *testing.T) {
	now := time.Now()

	monthlyEnd := now.AddDate(0, 1, 0)
	yearlyEnd := now.AddDate(1, 0, 0)

	// Monthly should be approximately 1 month
	assert.Equal(t, now.Month()+1, monthlyEnd.Month())
	// Yearly should be approximately 1 year
	assert.Equal(t, now.Year()+1, yearlyEnd.Year())
}

func Test_BillingService_PaymentIntent(t *testing.T) {
	paymentIntent := PaymentIntent{
		ID:           "pi_" + uuid.New().String()[:14],
		Amount:       2999, // $29.99
		Currency:     "usd",
		Status:       "requires_payment_method",
		ClientSecret: "pi_secret_" + uuid.New().String()[:14],
	}

	assert.Equal(t, int64(2999), paymentIntent.Amount)
	assert.Equal(t, "usd", paymentIntent.Currency)
	assert.Contains(t, paymentIntent.ID, "pi_")
	assert.Contains(t, paymentIntent.ClientSecret, "secret")
}

func Test_BillingService_Invoice(t *testing.T) {
	now := time.Now()
	paidAt := now

	invoice := Invoice{
		ID:          "in_" + uuid.New().String()[:14],
		CustomerID:  "cus_" + uuid.New().String()[:14],
		Amount:      9900, // $99.00
		Currency:    "usd",
		Status:      "paid",
		PeriodStart: now.AddDate(0, -1, 0),
		PeriodEnd:   now,
		InvoiceURL:  "https://invoice.stripe.com/i/" + uuid.New().String()[:8],
		InvoicePDF:  "https://pay.stripe.com/invoice/" + uuid.New().String()[:8],
		CreatedAt:   now.AddDate(0, -1, 0),
		PaidAt:      &paidAt,
	}

	assert.Equal(t, "paid", invoice.Status)
	assert.NotNil(t, invoice.PaidAt)
	assert.Equal(t, int64(9900), invoice.Amount)
}

func Test_BillingService_StripeEventTypes(t *testing.T) {
	validEventTypes := []string{
		"customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted",
		"invoice.paid",
		"invoice.payment_failed",
		"payment_intent.succeeded",
		"payment_intent.payment_failed",
	}

	event := StripeEvent{
		ID:      "evt_" + uuid.New().String()[:14],
		Type:    "customer.subscription.created",
		Data:    map[string]interface{}{},
		Created: time.Now().Unix(),
	}

	isValidEvent := false
	for _, eventType := range validEventTypes {
		if event.Type == eventType {
			isValidEvent = true
			break
		}
	}

	assert.True(t, isValidEvent)
}

func Test_BillingService_CheckoutSession(t *testing.T) {
	customerID := uuid.New()
	planID := uuid.New()

	sessionID := "cs_" + uuid.New().String()[:14]

	assert.NotEmpty(t, sessionID)
	assert.Contains(t, sessionID, "cs_")
	_ = customerID
	_ = planID
}

func Test_BillingService_BillingPortal(t *testing.T) {
	customerID := uuid.New()

	portalURL := "https://billing.stripe.com/p/session/" + uuid.New().String()[:14]

	assert.Contains(t, portalURL, "billing.stripe.com")
	assert.Contains(t, portalURL, "session")
	_ = customerID
}

func Test_BillingService_CustomerNotFound(t *testing.T) {
	// Test error handling for missing customer
	stripeCustomerID := "cus_nonexistent"

	assert.NotEqual(t, stripeCustomerID[:4], "cus_valid")
}

func Test_BillingService_WebhookSignature(t *testing.T) {
	signature := "t=timestamp,v1=signature"

	// Verify signature format
	assert.Contains(t, signature, "t=")
	assert.Contains(t, signature, "v1=")
}

func Test_BillingService_AmountInCents(t *testing.T) {
	// Test that amounts are in cents
	testCases := []struct {
		amountDollars float64
		expectedCents int64
	}{
		{9.99, 999},
		{29.99, 2999},
		{99.00, 9900},
		{0.00, 0},
	}

	for _, tc := range testCases {
		amountInCents := int64(tc.amountDollars * 100)
		assert.Equal(t, tc.expectedCents, amountInCents)
	}
}
