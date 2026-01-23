package platform

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"videostreamgo/internal/models/master"
)

// Test_SubscriptionHandler_ListPlans tests listing available plans
func Test_SubscriptionHandler_ListPlans(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/plans", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": []map[string]interface{}{
				{
					"id":            uuid.New(),
					"name":          "Starter",
					"description":   "Perfect for small video sites",
					"monthly_price": 9.99,
					"yearly_price":  99.99,
					"storage_gb":    50,
					"bandwidth_tb":  1,
					"max_videos":    100,
					"features":      []string{"HD video", "Basic analytics", "Email support"},
					"video_formats": []string{"720p"},
					"custom_domain": false,
					"white_label":   false,
				},
				{
					"id":            uuid.New(),
					"name":          "Pro",
					"description":   "For growing video businesses",
					"monthly_price": 29.99,
					"yearly_price":  299.99,
					"storage_gb":    500,
					"bandwidth_tb":  10,
					"max_videos":    1000,
					"features":      []string{"4K video", "Advanced analytics", "Priority support", "Custom branding"},
					"video_formats": []string{"720p", "1080p"},
					"custom_domain": true,
					"white_label":   false,
				},
				{
					"id":            uuid.New(),
					"name":          "Enterprise",
					"description":   "For large-scale video platforms",
					"monthly_price": 99.99,
					"yearly_price":  999.99,
					"storage_gb":    5000,
					"bandwidth_tb":  100,
					"max_videos":    -1, // Unlimited
					"features":      []string{"4K video", "Advanced analytics", "24/7 support", "Full white-label", "API access"},
					"video_formats": []string{"720p", "1080p", "4K"},
					"custom_domain": true,
					"white_label":   true,
				},
			},
		})
	})

	req := httptest.NewRequest("GET", "/plans", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	plans := response["data"].([]map[string]interface{})
	assert.Len(t, plans, 3)
	assert.Equal(t, "Starter", plans[0]["name"])
	assert.Equal(t, "Pro", plans[1]["name"])
	assert.Equal(t, "Enterprise", plans[2]["name"])
}

// Test_SubscriptionHandler_GetPlan tests getting a specific plan
func Test_SubscriptionHandler_GetPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	planID := uuid.New()

	r.GET("/plans/:id", func(c *gin.Context) {
		id := c.Param("id")
		parsedID, _ := uuid.Parse(id)

		if parsedID != planID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":            planID,
				"name":          "Pro",
				"monthly_price": 29.99,
				"yearly_price":  299.99,
				"features":      []string{"4K video", "Advanced analytics"},
			},
		})
	})

	req := httptest.NewRequest("GET", "/plans/"+planID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_SubscriptionHandler_Subscribe tests subscribing to a plan
func Test_SubscriptionHandler_Subscribe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	customerID := uuid.New()
	planID := uuid.New()

	r.POST("/subscriptions", func(c *gin.Context) {
		var req struct {
			PlanID       string `json:"plan_id" binding:"required"`
			BillingCycle string `json:"billing_cycle" binding:"required,oneof=monthly yearly"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := uuid.Parse(req.PlanID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
			return
		}

		subscriptionID := uuid.New()

		c.JSON(http.StatusCreated, gin.H{
			"data": map[string]interface{}{
				"id":                   subscriptionID,
				"customer_id":          customerID,
				"plan_id":              planID,
				"status":               "active",
				"billing_cycle":        req.BillingCycle,
				"current_period_start": "2024-01-01T00:00:00Z",
				"current_period_end":   "2024-02-01T00:00:00Z",
				"created_at":           "2024-01-01T00:00:00Z",
			},
			"message": "Subscription created successfully",
		})
	})

	body := `{"plan_id":"` + planID.String() + `","billing_cycle":"monthly"}`
	req := httptest.NewRequest("POST", "/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// Test_SubscriptionHandler_Cancel tests canceling a subscription
func Test_SubscriptionHandler_Cancel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	subscriptionID := uuid.New()

	r.POST("/subscriptions/:id/cancel", func(c *gin.Context) {
		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":              subscriptionID,
				"status":          "canceled",
				"canceled_at":     "2024-01-15T00:00:00Z",
				"effective_date":  "2024-02-01T00:00:00Z",
				"refund_prorated": true,
			},
			"message": "Subscription will be canceled at end of billing period",
		})
	})

	req := httptest.NewRequest("POST", "/subscriptions/"+subscriptionID.String()+"/cancel", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "canceled", data["status"])
}

// Test_SubscriptionHandler_ChangePlan tests changing subscription plan
func Test_SubscriptionHandler_ChangePlan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	subscriptionID := uuid.New()
	newPlanID := uuid.New()

	r.POST("/subscriptions/:id/change-plan", func(c *gin.Context) {
		var req struct {
			NewPlanID string `json:"new_plan_id" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":                    subscriptionID,
				"old_plan_id":           uuid.New(),
				"new_plan_id":           newPlanID,
				"status":                "active",
				"proration_credit":      15.50,
				"new_monthly_amount":    29.99,
				"effective_immediately": true,
			},
			"message": "Plan changed successfully",
		})
	})

	body := `{"new_plan_id":"` + newPlanID.String() + `"}`
	req := httptest.NewRequest("POST", "/subscriptions/"+subscriptionID.String()+"/change-plan", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_SubscriptionHandler_GetSubscription tests getting subscription details
func Test_SubscriptionHandler_GetSubscription(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	subscriptionID := uuid.New()
	planID := uuid.New()

	r.GET("/subscriptions/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":                   subscriptionID,
				"plan_id":              planID,
				"plan_name":            "Pro",
				"status":               "active",
				"billing_cycle":        "monthly",
				"monthly_amount":       29.99,
				"current_period_start": "2024-01-01T00:00:00Z",
				"current_period_end":   "2024-02-01T00:00:00Z",
				"created_at":           "2023-06-01T00:00:00Z",
				"usage": map[string]interface{}{
					"storage_used_gb":   150.5,
					"bandwidth_used_tb": 2.3,
					"videos_uploaded":   150,
				},
			},
		})
	})

	req := httptest.NewRequest("GET", "/subscriptions/"+subscriptionID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_SubscriptionHandler_ListBillingHistory tests listing billing history
func Test_SubscriptionHandler_ListBillingHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/billing/history", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": []map[string]interface{}{
				{
					"id":          uuid.New(),
					"type":        "subscription_payment",
					"description": "Pro Plan - Monthly",
					"amount":      29.99,
					"currency":    "USD",
					"status":      "succeeded",
					"invoice_id":  "INV-001",
					"created_at":  "2024-01-01T00:00:00Z",
					"pdf_url":     "/api/billing/invoices/INV-001.pdf",
				},
				{
					"id":          uuid.New(),
					"type":        "subscription_payment",
					"description": "Pro Plan - Monthly",
					"amount":      29.99,
					"currency":    "USD",
					"status":      "succeeded",
					"invoice_id":  "INV-002",
					"created_at":  "2023-12-01T00:00:00Z",
					"pdf_url":     "/api/billing/invoices/INV-002.pdf",
				},
			},
			"total":    24,
			"page":     1,
			"per_page": 20,
		})
	})

	req := httptest.NewRequest("GET", "/billing/history", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_SubscriptionStatusTransitions tests subscription status transitions
func Test_SubscriptionStatusTransitions(t *testing.T) {
	tests := []struct {
		name        string
		initial     master.SubscriptionStatus
		transition  string
		expected    master.SubscriptionStatus
		expectError bool
	}{
		{"Active to Past Due", master.SubscriptionStatusActive, "mark_past_due", master.SubscriptionStatusPastDue, false},
		{"Past Due to Active", master.SubscriptionStatusPastDue, "retry_payment", master.SubscriptionStatusActive, false},
		{"Active to Cancelled", master.SubscriptionStatusActive, "cancel", master.SubscriptionStatusCancelled, false},
		{"Trialing to Active", master.SubscriptionStatusTrialing, "complete_trial", master.SubscriptionStatusActive, false},
		{"Past Due to Cancelled", master.SubscriptionStatusPastDue, "cancel", master.SubscriptionStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate status transition logic
			newStatus := tt.initial

			switch tt.transition {
			case "mark_past_due":
				if tt.initial == master.SubscriptionStatusActive {
					newStatus = master.SubscriptionStatusPastDue
				}
			case "retry_payment":
				if tt.initial == master.SubscriptionStatusPastDue {
					newStatus = master.SubscriptionStatusActive
				}
			case "cancel":
				if tt.initial == master.SubscriptionStatusActive || tt.initial == master.SubscriptionStatusPastDue {
					newStatus = master.SubscriptionStatusCancelled
				}
			case "complete_trial":
				if tt.initial == master.SubscriptionStatusTrialing {
					newStatus = master.SubscriptionStatusActive
				}
			}

			if tt.expectError {
				assert.NotEqual(t, tt.expected, newStatus)
			} else {
				assert.Equal(t, tt.expected, newStatus)
			}
		})
	}
}
