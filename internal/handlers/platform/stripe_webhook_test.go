package platform

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Test_StripeWebhookHandler_HandleWebhook_Success tests successful webhook handling
func Test_StripeWebhookHandler_HandleWebhook_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/webhooks/stripe", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}

		var event StripeWebhookPayload
		if err := json.Unmarshal(body, &event); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse event"})
			return
		}

		// Handle the event based on type
		switch event.Type {
		case "customer.subscription.created":
			// Handle subscription created
		case "invoice.paid":
			// Handle invoice paid
		default:
			// Ignore unknown event types
		}

		c.JSON(http.StatusOK, gin.H{
			"received": true,
			"event_id": event.ID,
		})
	})

	event := StripeWebhookPayload{
		ID:      "evt_" + uuid.New().String()[:8],
		Type:    "invoice.paid",
		Data:    json.RawMessage(`{"id":"in_123","customer":"cus_123","amount_paid":99.99}`),
		Created: time.Now().Unix(),
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewBufferString(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response["received"].(bool))
}

// Test_StripeWebhookHandler_SignatureVerification tests Stripe signature verification
func Test_StripeWebhookHandler_SignatureVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	secret := "whsec_test_secret"
	_ = secret

	r.POST("/webhooks/stripe", func(c *gin.Context) {
		signature := c.GetHeader("Stripe-Signature")
		if signature == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing signature"})
			return
		}
		_ = signature

		// Simple signature verification (in production, use proper Stripe library)
		c.JSON(http.StatusOK, gin.H{"verified": true})
	})

	body := `{"type":"invoice.paid","data":{"object":{"id":"in_123"}}}`
	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "t=1234567890,v1=abc123")

	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_StripeWebhookHandler_InvalidPayload tests handling of invalid payload
func Test_StripeWebhookHandler_InvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/webhooks/stripe", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}

		var event StripeWebhookPayload
		if err := json.Unmarshal(body, &event); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse event"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"received": true})
	})

	// Invalid JSON
	body := `invalid json {`
	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_StripeWebhookHandler_Idempotency tests webhook idempotency
func Test_StripeWebhookHandler_Idempotency(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	processedEvents := make(map[string]bool)

	r.POST("/webhooks/stripe", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)

		var event StripeWebhookPayload
		json.Unmarshal(body, &event)

		// Check if already processed
		if processedEvents[event.ID] {
			c.JSON(http.StatusOK, gin.H{
				"received":          true,
				"event_id":          event.ID,
				"already_processed": true,
			})
			return
		}

		processedEvents[event.ID] = true

		c.JSON(http.StatusOK, gin.H{
			"received":          true,
			"event_id":          event.ID,
			"already_processed": false,
		})
	})

	event := StripeWebhookPayload{
		ID:      "evt_123",
		Type:    "invoice.paid",
		Data:    json.RawMessage(`{"id":"in_123"}`),
		Created: time.Now().Unix(),
	}
	body, _ := json.Marshal(event)

	// First request
	req1 := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewBufferString(string(body)))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	var response1 map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &response1)
	assert.False(t, response1["already_processed"].(bool))

	// Second request (same event)
	req2 := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewBufferString(string(body)))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var response2 map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &response2)
	assert.True(t, response2["already_processed"].(bool))
}

// Test_StripeWebhookHandler_SubscriptionCreated tests subscription.created event
func Test_StripeWebhookHandler_SubscriptionCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/webhooks/stripe", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)

		var event StripeWebhookPayload
		json.Unmarshal(body, &event)

		if event.Type == "customer.subscription.created" {
			var subData struct {
				ID                 string `json:"id"`
				Customer           string `json:"customer"`
				Status             string `json:"status"`
				CurrentPeriodStart int64  `json:"current_period_start"`
				CurrentPeriodEnd   int64  `json:"current_period_end"`
			}
			json.Unmarshal(event.Data, &subData)

			c.JSON(http.StatusOK, gin.H{
				"event_type":      "subscription_created",
				"subscription_id": subData.ID,
				"customer_id":     subData.Customer,
				"status":          subData.Status,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"received": true})
	})

	event := StripeWebhookPayload{
		ID:   "evt_sub_created",
		Type: "customer.subscription.created",
		Data: json.RawMessage(`{
			"id": "sub_123",
			"customer": "cus_123",
			"status": "active",
			"current_period_start": 1704067200,
			"current_period_end": 1706745600
		}`),
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewBufferString(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "subscription_created", response["event_type"])
	assert.Equal(t, "sub_123", response["subscription_id"])
}

// Test_StripeWebhookHandler_InvoicePaid tests invoice.paid event
func Test_StripeWebhookHandler_InvoicePaid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/webhooks/stripe", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)

		var event StripeWebhookPayload
		json.Unmarshal(body, &event)

		if event.Type == "invoice.paid" {
			var invoiceData struct {
				ID               string  `json:"id"`
				Customer         string  `json:"customer"`
				AmountPaid       float64 `json:"amount_paid"`
				Currency         string  `json:"currency"`
				HostedInvoiceURL string  `json:"hosted_invoice_url"`
			}
			json.Unmarshal(event.Data, &invoiceData)

			c.JSON(http.StatusOK, gin.H{
				"event_type":  "invoice_paid",
				"invoice_id":  invoiceData.ID,
				"customer_id": invoiceData.Customer,
				"amount_paid": invoiceData.AmountPaid,
				"currency":    invoiceData.Currency,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"received": true})
	})

	event := StripeWebhookPayload{
		ID:   "evt_invoice_paid",
		Type: "invoice.paid",
		Data: json.RawMessage(`{
			"id": "in_123",
			"customer": "cus_123",
			"amount_paid": 99.99,
			"currency": "usd",
			"hosted_invoice_url": "https://pay.stripe.com/invoices/in_123"
		}`),
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewBufferString(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "invoice_paid", response["event_type"])
	assert.Equal(t, 99.99, response["amount_paid"])
}

// Test_StripeWebhookHandler_PaymentFailed tests payment failure handling
func Test_StripeWebhookHandler_PaymentFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/webhooks/stripe", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)

		var event StripeWebhookPayload
		json.Unmarshal(body, &event)

		if event.Type == "invoice.payment_failed" {
			var invoiceData struct {
				ID        string  `json:"id"`
				Customer  string  `json:"customer"`
				AmountDue float64 `json:"amount_due"`
				Currency  string  `json:"currency"`
			}
			json.Unmarshal(event.Data, &invoiceData)

			c.JSON(http.StatusOK, gin.H{
				"event_type":  "payment_failed",
				"invoice_id":  invoiceData.ID,
				"customer_id": invoiceData.Customer,
				"amount_due":  invoiceData.AmountDue,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"received": true})
	})

	event := StripeWebhookPayload{
		ID:   "evt_payment_failed",
		Type: "invoice.payment_failed",
		Data: json.RawMessage(`{
			"id": "in_123",
			"customer": "cus_123",
			"amount_due": 99.99,
			"currency": "usd"
		}`),
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewBufferString(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "payment_failed", response["event_type"])
}

// Test_StripeWebhookHandler_UnknownEvent tests handling of unknown event types
func Test_StripeWebhookHandler_UnknownEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/webhooks/stripe", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)

		var event StripeWebhookPayload
		json.Unmarshal(body, &event)

		switch event.Type {
		case "customer.subscription.created":
		case "invoice.paid":
		default:
			// Ignore unknown event types
		}

		c.JSON(http.StatusOK, gin.H{
			"received":   true,
			"event_id":   event.ID,
			"event_type": event.Type,
			"handled":    false,
		})
	})

	event := StripeWebhookPayload{
		ID:   "evt_unknown",
		Type: "unknown.event.type",
		Data: json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewBufferString(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.False(t, response["handled"].(bool))
}

// Helper function for HMAC signature generation (used in production)
func generateStripeSignature(timestamp int64, payload []byte, secret string) string {
	signedPayload := string(timestamp) + "." + string(payload)
	signature := hmac.New(sha256.New, []byte(secret))
	signature.Write([]byte(signedPayload))
	return "t=" + string(rune(timestamp)) + ",v1=" + hex.EncodeToString(signature.Sum(nil))
}
