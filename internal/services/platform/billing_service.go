package platform

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"videostreamgo/internal/config"
	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
)

// Common errors
var (
	ErrInvalidSubscription     = errors.New("invalid subscription")
	ErrPaymentFailed           = errors.New("payment failed")
	ErrWebhookSignatureInvalid = errors.New("webhook signature is invalid")
)

// BillingService handles Stripe integration for payments
type BillingService struct {
	cfg              *config.Config
	billingRepo      *masterRepo.BillingRecordRepository
	customerRepo     *masterRepo.CustomerRepository
	subscriptionRepo *masterRepo.SubscriptionRepository
	planRepo         *masterRepo.PlanRepository
}

// NewBillingService creates a new BillingService
func NewBillingService(
	cfg *config.Config,
	billingRepo *masterRepo.BillingRecordRepository,
	customerRepo *masterRepo.CustomerRepository,
	subscriptionRepo *masterRepo.SubscriptionRepository,
	planRepo *masterRepo.PlanRepository,
) *BillingService {
	return &BillingService{
		cfg:              cfg,
		billingRepo:      billingRepo,
		customerRepo:     customerRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
	}
}

// Customer represents a customer for billing
type Customer struct {
	ID       string
	Email    string
	Name     string
	Phone    string
	Metadata map[string]string
}

// Subscription represents a Stripe subscription
type Subscription struct {
	ID                 string
	CustomerID         string
	Status             string
	PlanID             string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	CancelAtPeriodEnd  bool
}

// PaymentIntent represents a payment intent
type PaymentIntent struct {
	ID           string
	Amount       int64
	Currency     string
	Status       string
	ClientSecret string
}

// Invoice represents an invoice
type Invoice struct {
	ID          string
	CustomerID  string
	Amount      int64
	Currency    string
	Status      string
	PeriodStart time.Time
	PeriodEnd   time.Time
	InvoiceURL  string
	InvoicePDF  string
	CreatedAt   time.Time
	PaidAt      *time.Time
}

// StripeEvent represents a Stripe webhook event
type StripeEvent struct {
	ID      string
	Type    string
	Data    map[string]interface{}
	Created int64
}

// CreateCustomer creates a new customer in Stripe
func (s *BillingService) CreateCustomer(ctx context.Context, customer *master.Customer) (string, error) {
	// In production, this would call the Stripe API
	// For now, we simulate the creation
	stripeCustomerID := fmt.Sprintf("cus_%s", uuid.New().String()[:14])

	// Update customer with Stripe ID
	customer.StripeCustomerID = stripeCustomerID
	if err := s.customerRepo.Update(ctx, customer); err != nil {
		return "", fmt.Errorf("failed to update customer: %w", err)
	}

	return stripeCustomerID, nil
}

// CreateSubscription creates a new subscription for a customer
func (s *BillingService) CreateSubscription(ctx context.Context, customerID uuid.UUID, planID uuid.UUID, billingCycle master.BillingCycle) (*master.Subscription, error) {
	// Get the plan to validate it exists
	_, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}

	// Get the customer
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	if customer.StripeCustomerID == "" {
		return nil, errors.New("customer does not have a Stripe customer ID")
	}

	// In production, this would create a subscription in Stripe
	// For now, we simulate the creation
	stripeSubscriptionID := fmt.Sprintf("sub_%s", uuid.New().String()[:14])

	now := time.Now()
	var end time.Time
	if billingCycle == master.BillingCycleYearly {
		end = now.AddDate(1, 0, 0)
	} else {
		end = now.AddDate(0, 1, 0)
	}

	subscription := &master.Subscription{
		CustomerID:           customerID,
		PlanID:               planID,
		Status:               master.SubscriptionStatusActive,
		BillingCycle:         billingCycle,
		StripeSubscriptionID: stripeSubscriptionID,
		StripeCustomerID:     customer.StripeCustomerID,
		CurrentPeriodStart:   &now,
		CurrentPeriodEnd:     &end,
	}

	if err := s.subscriptionRepo.Create(ctx, subscription); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return subscription, nil
}

// CancelSubscription cancels a subscription
func (s *BillingService) CancelSubscription(ctx context.Context, subscriptionID uuid.UUID, cancelImmediately bool) error {
	subscription, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	if cancelImmediately {
		subscription.Status = master.SubscriptionStatusCancelled
	} else {
		subscription.CancelAtPeriodEnd = true
	}

	if err := s.subscriptionRepo.Update(ctx, subscription); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

// UpdateSubscription updates a subscription to a new plan
func (s *BillingService) UpdateSubscription(ctx context.Context, subscriptionID uuid.UUID, newPlanID uuid.UUID, billingCycle master.BillingCycle) (*master.Subscription, error) {
	subscription, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	// Verify the new plan exists
	_, err = s.planRepo.GetByID(ctx, newPlanID)
	if err != nil {
		return nil, fmt.Errorf("new plan not found: %w", err)
	}

	// In production, this would update the subscription in Stripe
	subscription.PlanID = newPlanID
	subscription.BillingCycle = billingCycle

	if err := s.subscriptionRepo.Update(ctx, subscription); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	return subscription, nil
}

// GetSubscription retrieves a subscription by ID
func (s *BillingService) GetSubscription(ctx context.Context, subscriptionID uuid.UUID) (*master.Subscription, error) {
	subscription, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}
	return subscription, nil
}

// CreatePaymentIntent creates a payment intent for one-time payments
func (s *BillingService) CreatePaymentIntent(ctx context.Context, customerID uuid.UUID, amount int64, currency string, description string) (*PaymentIntent, error) {
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	if customer.StripeCustomerID == "" {
		return nil, ErrCustomerNotFound
	}

	// In production, this would create a payment intent in Stripe
	paymentIntentID := fmt.Sprintf("pi_%s", uuid.New().String()[:14])
	clientSecret := fmt.Sprintf("%s_secret_%s", paymentIntentID, uuid.New().String()[:14])

	return &PaymentIntent{
		ID:           paymentIntentID,
		Amount:       amount,
		Currency:     currency,
		Status:       "requires_payment_method",
		ClientSecret: clientSecret,
	}, nil
}

// HandleWebhook handles Stripe webhook events
func (s *BillingService) HandleWebhook(ctx context.Context, event StripeEvent) error {
	switch event.Type {
	case "customer.subscription.created":
		return s.handleSubscriptionCreated(ctx, event)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(ctx, event)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(ctx, event)
	case "invoice.paid":
		return s.handleInvoicePaid(ctx, event)
	case "invoice.payment_failed":
		return s.handleInvoicePaymentFailed(ctx, event)
	case "payment_intent.succeeded":
		return s.handlePaymentIntentSucceeded(ctx, event)
	case "payment_intent.payment_failed":
		return s.handlePaymentIntentFailed(ctx, event)
	default:
		// Ignore unknown event types
		return nil
	}
}

// handleSubscriptionCreated handles subscription.created events
func (s *BillingService) handleSubscriptionCreated(ctx context.Context, event StripeEvent) error {
	// In production, parse the event data and update local subscription records
	return nil
}

// handleSubscriptionUpdated handles subscription.updated events
func (s *BillingService) handleSubscriptionUpdated(ctx context.Context, event StripeEvent) error {
	// In production, parse the event data and update local subscription records
	return nil
}

// handleSubscriptionDeleted handles subscription.deleted events
func (s *BillingService) handleSubscriptionDeleted(ctx context.Context, event StripeEvent) error {
	// In production, parse the event data and cancel local subscription records
	return nil
}

// handleInvoicePaid handles invoice.paid events
func (s *BillingService) handleInvoicePaid(ctx context.Context, event StripeEvent) error {
	// In production, create billing record for the payment
	return nil
}

// handleInvoicePaymentFailed handles invoice.payment_failed events
func (s *BillingService) handleInvoicePaymentFailed(ctx context.Context, event StripeEvent) error {
	// In production, handle failed payment - could suspend subscription
	return nil
}

// handlePaymentIntentSucceeded handles payment_intent.succeeded events
func (s *BillingService) handlePaymentIntentSucceeded(ctx context.Context, event StripeEvent) error {
	// In production, record successful payment
	return nil
}

// handlePaymentIntentFailed handles payment_intent.payment_failed events
func (s *BillingService) handlePaymentIntentFailed(ctx context.Context, event StripeEvent) error {
	// In production, handle failed payment
	return nil
}

// GetInvoices retrieves invoices for a customer
func (s *BillingService) GetInvoices(ctx context.Context, customerID uuid.UUID, limit int) ([]*Invoice, error) {
	// In production, this would call the Stripe API to list invoices
	// For now, return empty list
	return []*Invoice{}, nil
}

// CreateBillingRecord creates a billing record for a transaction
func (s *BillingService) CreateBillingRecord(ctx context.Context, record *master.BillingRecord) error {
	return s.billingRepo.Create(ctx, record)
}

// GetBillingRecords retrieves billing records for a customer
func (s *BillingService) GetBillingRecords(ctx context.Context, customerID uuid.UUID, offset, limit int) ([]master.BillingRecord, int64, error) {
	return s.billingRepo.ListByCustomer(ctx, customerID, offset, limit)
}

// GetTotalRevenue returns the total revenue from billing records
func (s *BillingService) GetTotalRevenue(ctx context.Context) (float64, error) {
	return s.billingRepo.GetTotalRevenue(ctx)
}

// GetRevenueByPeriod returns revenue for a specific period
func (s *BillingService) GetRevenueByPeriod(ctx context.Context, start, end time.Time) (float64, error) {
	return s.billingRepo.GetRevenueByPeriod(ctx, start, end)
}

// CreateStripeCheckoutSession creates a Stripe checkout session for subscription
func (s *BillingService) CreateStripeCheckoutSession(ctx context.Context, customerID uuid.UUID, planID uuid.UUID, successURL, cancelURL string) (string, error) {
	// In production, this would create a Stripe checkout session
	sessionID := fmt.Sprintf("cs_%s", uuid.New().String()[:14])
	return sessionID, nil
}

// CreateStripePortalSession creates a Stripe billing portal session
func (s *BillingService) CreateStripePortalSession(ctx context.Context, customerID uuid.UUID, returnURL string) (string, error) {
	// In production, this would create a Stripe billing portal session
	portalURL := fmt.Sprintf("https://billing.stripe.com/p/session/%s", uuid.New().String()[:14])
	return portalURL, nil
}
