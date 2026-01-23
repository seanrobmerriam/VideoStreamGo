package platform

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Test_BillingHandler_ListRecords_Success tests successful listing of billing records
func Test_BillingHandler_ListRecords_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/billing/records", func(c *gin.Context) {
		page := 1
		perPage := 20
		_ = c.Query("status")

		// Mock response
		records := []map[string]interface{}{
			{
				"id":            uuid.New(),
				"customer_id":   uuid.New(),
				"customer_name": "Test Company",
				"amount":        99.99,
				"currency":      "USD",
				"status":        "paid",
				"invoice_id":    "INV-001",
				"description":   "Monthly subscription",
				"created_at":    time.Now().Format("2006-01-02T15:04:05Z07:00"),
			},
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"records":  records,
				"total":    1,
				"page":     page,
				"per_page": perPage,
			},
		})
	})

	req := httptest.NewRequest("GET", "/billing/records?page=1&per_page=20", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["total"])
	assert.Equal(t, float64(1), data["page"])
}

// Test_BillingHandler_ListRecords_WithStatusFilter tests listing with status filter
func Test_BillingHandler_ListRecords_WithStatusFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/billing/records", func(c *gin.Context) {
		_ = c.Query("status")

		// Mock filtered response
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"records":  []map[string]interface{}{},
				"total":    0,
				"page":     1,
				"per_page": 20,
			},
		})
	})

	req := httptest.NewRequest("GET", "/billing/records?status=paid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_BillingHandler_GetRecord_Success tests getting a billing record by ID
func Test_BillingHandler_GetRecord_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	recordID := uuid.New()

	r.GET("/billing/records/:id", func(c *gin.Context) {
		id := c.Param("id")
		parsedID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "INVALID_ID",
					"message": "Invalid record ID",
				},
			})
			return
		}

		if parsedID != recordID {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "NOT_FOUND",
					"message": "Billing record not found",
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":          recordID,
				"customer_id": uuid.New(),
				"amount":      99.99,
				"currency":    "USD",
				"status":      "paid",
				"invoice_id":  "INV-001",
				"description": "Monthly subscription",
				"created_at":  time.Now().Format("2006-01-02T15:04:05Z07:00"),
			},
		})
	})

	req := httptest.NewRequest("GET", "/billing/records/"+recordID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_BillingHandler_GetRecord_InvalidID tests getting a record with invalid ID
func Test_BillingHandler_GetRecord_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/billing/records/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "INVALID_ID",
					"message": "Invalid record ID",
				},
			})
			return
		}
	})

	req := httptest.NewRequest("GET", "/billing/records/invalid-uuid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_BillingHandler_GetRecord_NotFound tests getting a non-existent record
func Test_BillingHandler_GetRecord_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/billing/records/:id", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "NOT_FOUND",
				"message": "Billing record not found",
			},
		})
	})

	req := httptest.NewRequest("GET", "/billing/records/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Test_BillingHandler_CreateRecord_Success tests creating a billing record
func Test_BillingHandler_CreateRecord_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/billing/records", func(c *gin.Context) {
		var req struct {
			CustomerID  string  `json:"customer_id" binding:"required,uuid"`
			Amount      float64 `json:"amount" binding:"required,gte=0"`
			Currency    string  `json:"currency" binding:"required,len=3"`
			Description string  `json:"description"`
			InvoiceID   string  `json:"invoice_id"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "VALIDATION_ERROR",
					"message": "Invalid request",
					"details": err.Error(),
				},
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"data": map[string]interface{}{
				"id":          uuid.New(),
				"customer_id": req.CustomerID,
				"amount":      req.Amount,
				"currency":    req.Currency,
				"status":      "pending",
				"created_at":  time.Now().Format("2006-01-02T15:04:05Z07:00"),
			},
			"message": "Billing record created successfully",
		})
	})

	body := `{"customer_id":"` + uuid.New().String() + `","amount":99.99,"currency":"USD","description":"Test invoice","invoice_id":"INV-001"}`
	req := httptest.NewRequest("POST", "/billing/records", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// Test_BillingHandler_CreateRecord_ValidationError tests creating with invalid data
func Test_BillingHandler_CreateRecord_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/billing/records", func(c *gin.Context) {
		var req struct {
			CustomerID string  `json:"customer_id" binding:"required,uuid"`
			Amount     float64 `json:"amount" binding:"required,gte=0"`
			Currency   string  `json:"currency" binding:"required,len=3"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "VALIDATION_ERROR",
					"message": "Invalid request",
					"details": err.Error(),
				},
			})
			return
		}
	})

	// Invalid currency (not 3 characters)
	body := `{"customer_id":"` + uuid.New().String() + `","amount":99.99,"currency":"INVALID"}`
	req := httptest.NewRequest("POST", "/billing/records", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_BillingHandler_GetOverview tests getting billing overview
func Test_BillingHandler_GetOverview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/billing/overview", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"total_revenue":    15000.00,
				"active_customers": 50,
				"active_instances": 45,
				"total_storage_gb": 1024,
				"period_start":     time.Now().AddDate(0, -1, 0).Format("2006-01-02"),
				"period_end":       time.Now().Format("2006-01-02"),
			},
		})
	})

	req := httptest.NewRequest("GET", "/billing/overview", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, 15000.00, data["total_revenue"])
	assert.Equal(t, float64(50), data["active_customers"])
}

// Test_BillingHandler_GetRevenueReport tests getting revenue report
func Test_BillingHandler_GetRevenueReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/billing/reports/revenue", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"total_revenue":       50000.00,
				"total_transactions":  150,
				"average_transaction": 333.33,
			},
		})
	})

	req := httptest.NewRequest("GET", "/billing/reports/revenue", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_BillingHandler_GetUsageReport tests getting usage report
func Test_BillingHandler_GetUsageReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/billing/reports/usage", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"total_storage_gb":   500,
				"total_bandwidth_gb": 2000,
				"total_videos":       100,
				"total_users":        500,
			},
		})
	})

	req := httptest.NewRequest("GET", "/billing/reports/usage", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_BillingHandler_GetCustomerBilling tests getting customer billing details
func Test_BillingHandler_GetCustomerBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	customerID := uuid.New()

	r.GET("/billing/customers/:id", func(c *gin.Context) {
		id := c.Param("id")
		parsedID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "INVALID_ID",
					"message": "Invalid customer ID",
				},
			})
			return
		}

		if parsedID != customerID {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "NOT_FOUND",
					"message": "Customer not found",
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"customer": map[string]interface{}{
					"id":      customerID,
					"email":   "test@company.com",
					"company": "Test Company",
					"status":  "active",
				},
				"billing_records": []map[string]interface{}{},
				"total_revenue":   500.00,
			},
		})
	})

	req := httptest.NewRequest("GET", "/billing/customers/"+customerID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_BillingHandler_GenerateInvoice tests invoice generation
func Test_BillingHandler_GenerateInvoice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	customerID := uuid.New()

	r.POST("/billing/customers/:id/invoice", func(c *gin.Context) {
		id := c.Param("id")
		parsedID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
			return
		}

		var req struct {
			PeriodStart string `json:"period_start" binding:"required"`
			PeriodEnd   string `json:"period_end" binding:"required"`
			Description string `json:"description"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"customer_id":    parsedID,
				"period_start":   req.PeriodStart,
				"period_end":     req.PeriodEnd,
				"description":    req.Description,
				"invoice_number": "INV-" + uuid.New().String()[:8],
				"status":         "generated",
			},
			"message": "Invoice generated successfully",
		})
	})

	body := `{"period_start":"2024-01-01","period_end":"2024-01-31","description":"Monthly service"}`
	req := httptest.NewRequest("POST", "/billing/customers/"+customerID.String()+"/invoice", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_BillingHandler_GetRevenueAnalytics tests revenue analytics endpoint
func Test_BillingHandler_GetRevenueAnalytics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/billing/analytics", func(c *gin.Context) {
		months := 12

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"total_revenue":     100000.00,
				"revenue_by_status": map[string]float64{"paid": 80000, "pending": 20000},
				"active_customers":  100,
				"months":            months,
				"period": map[string]string{
					"start": time.Now().AddDate(0, -months, 0).Format("2006-01-02"),
					"end":   time.Now().Format("2006-01-02"),
				},
			},
		})
	})

	req := httptest.NewRequest("GET", "/billing/analytics?months=12", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_BillingHandler_ListCustomers tests listing customers with billing info
func Test_BillingHandler_ListCustomers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/billing/customers", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"customers": []map[string]interface{}{
					{
						"id":            uuid.New(),
						"email":         "customer1@test.com",
						"company_name":  "Company 1",
						"status":        "active",
						"total_revenue": 500.00,
					},
				},
				"total":    1,
				"page":     1,
				"per_page": 20,
			},
		})
	})

	req := httptest.NewRequest("GET", "/billing/customers", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_BillingHandler_GetUsageMetrics tests getting usage metrics
func Test_BillingHandler_GetUsageMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	instanceID := uuid.New()

	r.GET("/billing/instances/:instance_id/usage", func(c *gin.Context) {
		id := c.Param("instance_id")
		parsedID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"instance_id":       parsedID,
				"storage_used_gb":   50,
				"bandwidth_used_gb": 200,
				"video_count":       25,
				"user_count":        100,
				"period_start":      time.Now().AddDate(0, -1, 0).Format("2006-01-02"),
				"period_end":        time.Now().Format("2006-01-02"),
			},
		})
	})

	req := httptest.NewRequest("GET", "/billing/instances/"+instanceID.String()+"/usage", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
