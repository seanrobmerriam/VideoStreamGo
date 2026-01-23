package platform

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"videostreamgo/internal/config"
	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
	"videostreamgo/internal/services/platform"
	"videostreamgo/internal/types"
)

// StripeWebhookHandler handles Stripe webhook events
type StripeWebhookHandler struct {
	cfg              *config.Config
	billingService   *platform.BillingService
	customerRepo     *masterRepo.CustomerRepository
	subscriptionRepo *masterRepo.SubscriptionRepository
	billingRepo      *masterRepo.BillingRecordRepository
}

// NewStripeWebhookHandler creates a new StripeWebhookHandler
func NewStripeWebhookHandler(
	cfg *config.Config,
	billingService *platform.BillingService,
	customerRepo *masterRepo.CustomerRepository,
	subscriptionRepo *masterRepo.SubscriptionRepository,
	billingRepo *masterRepo.BillingRecordRepository,
) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		cfg:              cfg,
		billingService:   billingService,
		customerRepo:     customerRepo,
		subscriptionRepo: subscriptionRepo,
		billingRepo:      billingRepo,
	}
}

// StripeWebhookPayload represents the raw webhook payload
type StripeWebhookPayload struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
	Created int64           `json:"created"`
}

// HandleWebhook processes incoming Stripe webhook events
func (h *StripeWebhookHandler) HandleWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("WEBHOOK_ERROR", "Failed to read request body", err.Error()))
		return
	}

	// Parse the event
	var event StripeWebhookPayload
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("WEBHOOK_ERROR", "Failed to parse event", err.Error()))
		return
	}

	ctx := c.Request.Context()

	// Handle the event based on type
	switch event.Type {
	case "customer.subscription.created":
		h.handleSubscriptionCreated(ctx, event.Data)
	case "customer.subscription.updated":
		h.handleSubscriptionUpdated(ctx, event.Data)
	case "customer.subscription.deleted":
		h.handleSubscriptionDeleted(ctx, event.Data)
	case "invoice.paid":
		h.handleInvoicePaid(ctx, event.Data)
	case "invoice.payment_failed":
		h.handleInvoicePaymentFailed(ctx, event.Data)
	case "payment_intent.succeeded":
		h.handlePaymentIntentSucceeded(ctx, event.Data)
	case "payment_intent.payment_failed":
		h.handlePaymentIntentFailed(ctx, event.Data)
	default:
		// Ignore unknown event types
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"received": true,
		"event_id": event.ID,
	}, ""))
}

// handleSubscriptionCreated handles subscription.created events
func (h *StripeWebhookHandler) handleSubscriptionCreated(ctx interface{}, data json.RawMessage) {
	// Parse the subscription data
	var subData struct {
		ID                 string `json:"id"`
		Customer           string `json:"customer"`
		Status             string `json:"status"`
		CurrentPeriodStart int64  `json:"current_period_start"`
		CurrentPeriodEnd   int64  `json:"current_period_end"`
	}

	if err := json.Unmarshal(data, &subData); err != nil {
		return
	}

	// In production, find customer by Stripe ID and create local subscription record
	_ = subData
}

// handleSubscriptionUpdated handles subscription.updated events
func (h *StripeWebhookHandler) handleSubscriptionUpdated(ctx interface{}, data json.RawMessage) {
	var subData struct {
		ID   string `json:"id"`
		Data struct {
			Object struct {
				ID                 string `json:"id"`
				Status             string `json:"status"`
				CurrentPeriodStart int64  `json:"current_period_start"`
				CurrentPeriodEnd   int64  `json:"current_period_end"`
			} `json:"object"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &subData); err != nil {
		return
	}

	// In production, update local subscription record
	_ = subData
}

// handleSubscriptionDeleted handles subscription.deleted events
func (h *StripeWebhookHandler) handleSubscriptionDeleted(ctx interface{}, data json.RawMessage) {
	var subData struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(data, &subData); err != nil {
		return
	}

	// In production, find and cancel local subscription
	_ = subData
}

// handleInvoicePaid handles invoice.paid events
func (h *StripeWebhookHandler) handleInvoicePaid(ctx interface{}, data json.RawMessage) {
	var invoiceData struct {
		ID               string  `json:"id"`
		Customer         string  `json:"customer"`
		AmountPaid       float64 `json:"amount_paid"`
		Currency         string  `json:"currency"`
		PeriodStart      int64   `json:"period_start"`
		PeriodEnd        int64   `json:"period_end"`
		HostedInvoiceURL string  `json:"hosted_invoice_url"`
	}

	if err := json.Unmarshal(data, &invoiceData); err != nil {
		return
	}

	// In production, find customer by Stripe ID and create billing record
	_ = invoiceData
}

// handleInvoicePaymentFailed handles invoice.payment_failed events
func (h *StripeWebhookHandler) handleInvoicePaymentFailed(ctx interface{}, data json.RawMessage) {
	var invoiceData struct {
		ID        string  `json:"id"`
		Customer  string  `json:"customer"`
		AmountDue float64 `json:"amount_due"`
		Currency  string  `json:"currency"`
	}

	if err := json.Unmarshal(data, &invoiceData); err != nil {
		return
	}

	// In production, handle failed payment
	_ = invoiceData
}

// handlePaymentIntentSucceeded handles payment_intent.succeeded events
func (h *StripeWebhookHandler) handlePaymentIntentSucceeded(ctx interface{}, data json.RawMessage) {
	var paymentData struct {
		ID       string  `json:"id"`
		Customer string  `json:"customer"`
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}

	if err := json.Unmarshal(data, &paymentData); err != nil {
		return
	}

	// In production, handle successful payment
	_ = paymentData
}

// handlePaymentIntentFailed handles payment_intent.payment_failed events
func (h *StripeWebhookHandler) handlePaymentIntentFailed(ctx interface{}, data json.RawMessage) {
	var paymentData struct {
		ID       string  `json:"id"`
		Customer string  `json:"customer"`
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}

	if err := json.Unmarshal(data, &paymentData); err != nil {
		return
	}

	// In production, handle failed payment
	_ = paymentData
}

// CreateBillingRecordFromInvoice creates a billing record from a paid invoice
func (h *StripeWebhookHandler) CreateBillingRecordFromInvoice(ctx gin.Context, customerID uuid.UUID, invoiceID string, amount float64, currency string) {
	now := time.Now()
	billingRecord := &master.BillingRecord{
		CustomerID:  customerID,
		Amount:      amount,
		Currency:    currency,
		Status:      master.BillingRecordStatusPaid,
		Type:        "subscription",
		Description: "Invoice payment",
		InvoiceID:   invoiceID,
		PaidAt:      &now,
	}

	h.billingRepo.Create(ctx.Request.Context(), billingRecord)
}
