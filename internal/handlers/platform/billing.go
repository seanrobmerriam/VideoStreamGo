package platform

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"videostreamgo/internal/config"
	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
	"videostreamgo/internal/types"
)

// BillingHandler handles billing endpoints
type BillingHandler struct {
	billingRepo  *masterRepo.BillingRecordRepository
	customerRepo *masterRepo.CustomerRepository
	cfg          *config.Config
}

// NewBillingHandler creates a new BillingHandler
func NewBillingHandler(billingRepo *masterRepo.BillingRecordRepository, customerRepo *masterRepo.CustomerRepository, cfg *config.Config) *BillingHandler {
	return &BillingHandler{
		billingRepo:  billingRepo,
		customerRepo: customerRepo,
		cfg:          cfg,
	}
}

// ListRecords returns all billing records
func (h *BillingHandler) ListRecords(c *gin.Context) {
	page := getIntParam(c, "page", 1)
	perPage := getIntParam(c, "per_page", 20)
	status := c.Query("status")

	records, total, err := h.billingRepo.List(c.Request.Context(), (page-1)*perPage, perPage, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("LIST_ERROR", "Failed to list billing records", err.Error()))
		return
	}

	result := make([]map[string]interface{}, len(records))
	for i, record := range records {
		customer, _ := h.customerRepo.GetByID(c.Request.Context(), record.CustomerID)
		customerName := ""
		if customer != nil {
			customerName = customer.CompanyName
		}

		result[i] = map[string]interface{}{
			"id":            record.ID,
			"customer_id":   record.CustomerID,
			"customer_name": customerName,
			"amount":        record.Amount,
			"currency":      record.Currency,
			"status":        record.Status,
			"invoice_id":    record.InvoiceID,
			"description":   record.Description,
			"created_at":    record.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"records":  result,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	}, ""))
}

// GetRecord returns a billing record by ID
func (h *BillingHandler) GetRecord(c *gin.Context) {
	id := c.Param("id")
	recordID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid record ID", ""))
		return
	}

	record, err := h.billingRepo.GetByID(c.Request.Context(), recordID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Billing record not found", ""))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":          record.ID,
		"customer_id": record.CustomerID,
		"amount":      record.Amount,
		"currency":    record.Currency,
		"status":      record.Status,
		"invoice_id":  record.InvoiceID,
		"description": record.Description,
		"created_at":  record.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, ""))
}

// CreateRecord creates a new billing record
func (h *BillingHandler) CreateRecord(c *gin.Context) {
	var req struct {
		CustomerID  string  `json:"customer_id" binding:"required,uuid"`
		Amount      float64 `json:"amount" binding:"required,gte=0"`
		Currency    string  `json:"currency" binding:"required,len=3"`
		Description string  `json:"description"`
		InvoiceID   string  `json:"invoice_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	customerID, _ := uuid.Parse(req.CustomerID)

	record := &master.BillingRecord{
		CustomerID:  customerID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Status:      master.BillingRecordStatusPending,
		Description: req.Description,
		InvoiceID:   req.InvoiceID,
	}

	if err := h.billingRepo.Create(c.Request.Context(), record); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("CREATE_ERROR", "Failed to create billing record", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.SuccessResponse(map[string]interface{}{
		"id":          record.ID,
		"customer_id": record.CustomerID,
		"amount":      record.Amount,
		"currency":    record.Currency,
		"status":      record.Status,
		"created_at":  record.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, "Billing record created successfully"))
}

// GetUsageMetrics returns usage metrics for an instance
func (h *BillingHandler) GetUsageMetrics(c *gin.Context) {
	instanceID := c.Param("instance_id")
	id, err := uuid.Parse(instanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid instance ID", ""))
		return
	}

	// This would typically call the instance-specific database
	// For now, return empty metrics
	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"instance_id":       id,
		"storage_used_gb":   0,
		"bandwidth_used_gb": 0,
		"video_count":       0,
		"user_count":        0,
		"period_start":      time.Now().AddDate(0, -1, 0).Format("2006-01-02"),
		"period_end":        time.Now().Format("2006-01-02"),
	}, ""))
}

// GetOverview returns platform analytics overview
func (h *BillingHandler) GetOverview(c *gin.Context) {
	revenue, _ := h.billingRepo.GetTotalRevenue(c.Request.Context())

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"total_revenue":    revenue,
		"active_customers": 0,
		"active_instances": 0,
		"total_storage_gb": 0,
		"period_start":     time.Now().AddDate(0, -1, 0).Format("2006-01-02"),
		"period_end":       time.Now().Format("2006-01-02"),
	}, ""))
}

// GetRevenueReport returns revenue analytics
func (h *BillingHandler) GetRevenueReport(c *gin.Context) {
	revenue, _ := h.billingRepo.GetTotalRevenue(c.Request.Context())

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"total_revenue":       revenue,
		"total_transactions":  0,
		"average_transaction": 0,
	}, ""))
}

// GetUsageReport returns usage analytics
func (h *BillingHandler) GetUsageReport(c *gin.Context) {
	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"total_storage_gb":   0,
		"total_bandwidth_gb": 0,
		"total_videos":       0,
		"total_users":        0,
	}, ""))
}

// ListCustomers lists all customers with billing info
func (h *BillingHandler) ListCustomers(c *gin.Context) {
	page := getIntParam(c, "page", 1)
	perPage := getIntParam(c, "per_page", 20)
	status := c.Query("status")

	customers, total, err := h.customerRepo.List(c.Request.Context(), (page-1)*perPage, perPage, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("LIST_ERROR", "Failed to list customers", err.Error()))
		return
	}

	result := make([]map[string]interface{}, len(customers))
	for i, customer := range customers {
		revenue, _ := h.billingRepo.GetRevenueByPeriod(c.Request.Context(), time.Now().AddDate(0, -1, 0), time.Now())

		result[i] = map[string]interface{}{
			"id":                 customer.ID,
			"email":              customer.Email,
			"company_name":       customer.CompanyName,
			"status":             customer.Status,
			"stripe_customer_id": customer.StripeCustomerID,
			"total_revenue":      revenue,
			"created_at":         customer.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"customers": result,
		"total":     total,
		"page":      page,
		"per_page":  perPage,
	}, ""))
}

// GetCustomerBilling returns billing details for a specific customer
func (h *BillingHandler) GetCustomerBilling(c *gin.Context) {
	id := c.Param("id")
	customerID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid customer ID", ""))
		return
	}

	customer, err := h.customerRepo.GetByID(c.Request.Context(), customerID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Customer not found", ""))
		return
	}

	// Get billing records
	records, _ := h.billingRepo.GetByCustomerID(c.Request.Context(), customerID)
	totalRevenue, _ := h.billingRepo.GetTotalRevenue(c.Request.Context())

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"customer": map[string]interface{}{
			"id":                 customer.ID,
			"email":              customer.Email,
			"company_name":       customer.CompanyName,
			"status":             customer.Status,
			"stripe_customer_id": customer.StripeCustomerID,
			"created_at":         customer.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		"billing_records": records,
		"total_revenue":   totalRevenue,
	}, ""))
}

// GenerateInvoice generates an invoice for a customer
func (h *BillingHandler) GenerateInvoice(c *gin.Context) {
	id := c.Param("id")
	customerID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid customer ID", ""))
		return
	}

	var req struct {
		PeriodStart string `json:"period_start" binding:"required"`
		PeriodEnd   string `json:"period_end" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	// In production, this would generate a proper invoice
	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"customer_id":    customerID,
		"period_start":   req.PeriodStart,
		"period_end":     req.PeriodEnd,
		"description":    req.Description,
		"invoice_number": "INV-" + uuid.New().String()[:8],
		"status":         "generated",
	}, "Invoice generated successfully"))
}

// GetBillingReports returns billing reports
func (h *BillingHandler) GetBillingReports(c *gin.Context) {
	revenueByStatus, _ := h.billingRepo.GetRevenueByStatus(c.Request.Context())

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"revenue_by_status": revenueByStatus,
		"report_period": map[string]string{
			"start": time.Now().AddDate(0, -1, 0).Format("2006-01-02"),
			"end":   time.Now().Format("2006-01-02"),
		},
	}, ""))
}

// GetRevenueAnalytics returns revenue analytics
func (h *BillingHandler) GetRevenueAnalytics(c *gin.Context) {
	months := getIntParam(c, "months", 12)

	totalRevenue, _ := h.billingRepo.GetTotalRevenue(c.Request.Context())
	revenueByStatus, _ := h.billingRepo.GetRevenueByStatus(c.Request.Context())
	activeCustomers, _ := h.customerRepo.GetActiveCount(c.Request.Context())

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"total_revenue":     totalRevenue,
		"revenue_by_status": revenueByStatus,
		"active_customers":  activeCustomers,
		"months":            months,
		"period": map[string]string{
			"start": time.Now().AddDate(0, -months, 0).Format("2006-01-02"),
			"end":   time.Now().Format("2006-01-02"),
		},
	}, ""))
}
