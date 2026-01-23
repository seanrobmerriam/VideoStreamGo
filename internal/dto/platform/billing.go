package platform

import (
	"github.com/google/uuid"

	"videostreamgo/internal/models/master"
)

// CreateBillingRecordRequest represents a request to create a billing record
type CreateBillingRecordRequest struct {
	CustomerID  string  `json:"customer_id" binding:"required,uuid"`
	Amount      float64 `json:"amount" binding:"required,gte=0"`
	Type        string  `json:"type" binding:"required,oneof=subscription usage overage refund"`
	Description string  `json:"description"`
	InvoiceID   string  `json:"invoice_id"`
}

// BillingRecordResponse represents a billing record in API responses
type BillingRecordResponse struct {
	ID           uuid.UUID                  `json:"id"`
	CustomerID   uuid.UUID                  `json:"customer_id"`
	CustomerName string                     `json:"customer_name"`
	Amount       float64                    `json:"amount"`
	Currency     string                     `json:"currency"`
	Description  string                     `json:"description"`
	InvoiceID    string                     `json:"invoice_id,omitempty"`
	Status       master.BillingRecordStatus `json:"status"`
	PeriodStart  *string                    `json:"period_start,omitempty"`
	PeriodEnd    *string                    `json:"period_end,omitempty"`
	CreatedAt    string                     `json:"created_at"`
}

// ToBillingRecordResponse converts a BillingRecord to BillingRecordResponse
func ToBillingRecordResponse(record *master.BillingRecord, customerName string) BillingRecordResponse {
	resp := BillingRecordResponse{
		ID:           record.ID,
		CustomerID:   record.CustomerID,
		CustomerName: customerName,
		Amount:       record.Amount,
		Currency:     record.Currency,
		Description:  record.Description,
		InvoiceID:    record.InvoiceID,
		Status:       record.Status,
		CreatedAt:    record.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if record.PeriodStart != nil {
		start := record.PeriodStart.Format("2006-01-02")
		resp.PeriodStart = &start
	}
	if record.PeriodEnd != nil {
		end := record.PeriodEnd.Format("2006-01-02")
		resp.PeriodEnd = &end
	}
	return resp
}

// BillingRecordListResponse represents a list of billing records with pagination
type BillingRecordListResponse struct {
	Records []BillingRecordResponse `json:"records"`
	Total   int64                   `json:"total"`
	Page    int                     `json:"page"`
	PerPage int                     `json:"per_page"`
}

// UsageMetricsResponse represents usage metrics in API responses
type UsageMetricsResponse struct {
	InstanceID       uuid.UUID `json:"instance_id"`
	InstanceName     string    `json:"instance_name"`
	PeriodStart      string    `json:"period_start"`
	PeriodEnd        string    `json:"period_end"`
	StorageUsedGB    float64   `json:"storage_used_gb"`
	StorageLimitGB   int       `json:"storage_limit_gb"`
	BandwidthUsedGB  float64   `json:"bandwidth_used_gb"`
	BandwidthLimitGB int       `json:"bandwidth_limit_gb"`
	VideoCount       int       `json:"video_count"`
	VideoLimit       int       `json:"video_limit"`
	UserCount        int       `json:"user_count"`
	UserLimit        int       `json:"user_limit"`
}

// GenerateInvoiceRequest represents a request to generate an invoice
type GenerateInvoiceRequest struct {
	CustomerID  string `json:"customer_id" binding:"required,uuid"`
	PeriodStart string `json:"period_start" binding:"required"`
	PeriodEnd   string `json:"period_end" binding:"required"`
}

// InvoiceResponse represents an invoice in API responses
type InvoiceResponse struct {
	ID            uuid.UUID `json:"id"`
	CustomerID    uuid.UUID `json:"customer_id"`
	CustomerName  string    `json:"customer_name"`
	InvoiceNumber string    `json:"invoice_number"`
	Amount        float64   `json:"amount"`
	TaxAmount     float64   `json:"tax_amount"`
	TotalAmount   float64   `json:"total_amount"`
	Status        string    `json:"status"`
	PeriodStart   string    `json:"period_start"`
	PeriodEnd     string    `json:"period_end"`
	DueDate       string    `json:"due_date"`
	CreatedAt     string    `json:"created_at"`
}

// PaymentRequest represents a payment processing request
type PaymentRequest struct {
	CustomerID    string  `json:"customer_id" binding:"required,uuid"`
	Amount        float64 `json:"amount" binding:"required,gte=0.01"`
	Currency      string  `json:"currency" binding:"required,len=3"`
	PaymentMethod string  `json:"payment_method" binding:"required"`
	InvoiceID     string  `json:"invoice_id" binding:"omitempty,uuid"`
}

// PaymentResponse represents a payment processing response
type PaymentResponse struct {
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"`
	ProcessedAt   string  `json:"processed_at"`
}
