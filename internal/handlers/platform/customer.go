package platform

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"videostreamgo/internal/config"
	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
	"videostreamgo/internal/types"
)

// CustomerHandler handles customer endpoints
type CustomerHandler struct {
	customerRepo *masterRepo.CustomerRepository
	instanceRepo *masterRepo.InstanceRepository
	cfg          *config.Config
}

// NewCustomerHandler creates a new CustomerHandler
func NewCustomerHandler(customerRepo *masterRepo.CustomerRepository, instanceRepo *masterRepo.InstanceRepository, cfg *config.Config) *CustomerHandler {
	return &CustomerHandler{
		customerRepo: customerRepo,
		instanceRepo: instanceRepo,
		cfg:          cfg,
	}
}

// List returns all customers
func (h *CustomerHandler) List(c *gin.Context) {
	page := getIntParam(c, "page", 1)
	perPage := getIntParam(c, "per_page", 20)
	status := c.Query("status")

	customers, total, err := h.customerRepo.List(c.Request.Context(), (page-1)*perPage, perPage, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("LIST_ERROR", "Failed to list customers", err.Error()))
		return
	}

	// Get instance counts for each customer
	result := make([]map[string]interface{}, len(customers))
	for i, customer := range customers {
		instances, _ := h.instanceRepo.GetByCustomerID(c.Request.Context(), customer.ID)
		result[i] = map[string]interface{}{
			"id":             customer.ID,
			"email":          customer.Email,
			"company_name":   customer.CompanyName,
			"contact_name":   customer.ContactName,
			"phone":          customer.Phone,
			"status":         customer.Status,
			"instance_count": len(instances),
			"created_at":     customer.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"updated_at":     customer.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"customers": result,
		"total":     total,
		"page":      page,
		"per_page":  perPage,
	}, ""))
}

// Create creates a new customer
func (h *CustomerHandler) Create(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required,min=8"`
		CompanyName string `json:"company_name" binding:"required,min=2"`
		ContactName string `json:"contact_name" binding:"required,min=2"`
		Phone       string `json:"phone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	// Check if customer exists
	_, err := h.customerRepo.GetByEmail(c.Request.Context(), req.Email)
	if err == nil {
		c.JSON(http.StatusConflict, types.ErrorResponse("EMAIL_EXISTS", "Email already registered", ""))
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	customer := &master.Customer{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		CompanyName:  req.CompanyName,
		ContactName:  req.ContactName,
		Phone:        req.Phone,
		Status:       master.CustomerStatusActive,
	}

	if err := h.customerRepo.Create(c.Request.Context(), customer); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("CREATE_ERROR", "Failed to create customer", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.SuccessResponse(map[string]interface{}{
		"id":           customer.ID,
		"email":        customer.Email,
		"company_name": customer.CompanyName,
		"contact_name": customer.ContactName,
		"status":       customer.Status,
		"created_at":   customer.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, "Customer created successfully"))
}

// Get returns a customer by ID
func (h *CustomerHandler) Get(c *gin.Context) {
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

	instances, _ := h.instanceRepo.GetByCustomerID(c.Request.Context(), customer.ID)

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":             customer.ID,
		"email":          customer.Email,
		"company_name":   customer.CompanyName,
		"contact_name":   customer.ContactName,
		"phone":          customer.Phone,
		"status":         customer.Status,
		"instance_count": len(instances),
		"created_at":     customer.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":     customer.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, ""))
}

// Update updates a customer
func (h *CustomerHandler) Update(c *gin.Context) {
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

	var req struct {
		Email       *string `json:"email"`
		CompanyName *string `json:"company_name"`
		ContactName *string `json:"contact_name"`
		Phone       *string `json:"phone"`
		Status      *string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	if req.Email != nil {
		customer.Email = *req.Email
	}
	if req.CompanyName != nil {
		customer.CompanyName = *req.CompanyName
	}
	if req.ContactName != nil {
		customer.ContactName = *req.ContactName
	}
	if req.Phone != nil {
		customer.Phone = *req.Phone
	}
	if req.Status != nil {
		customer.Status = master.CustomerStatus(*req.Status)
	}

	if err := h.customerRepo.Update(c.Request.Context(), customer); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to update customer", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":           customer.ID,
		"email":        customer.Email,
		"company_name": customer.CompanyName,
		"status":       customer.Status,
		"updated_at":   customer.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, "Customer updated successfully"))
}

// Delete soft deletes a customer
func (h *CustomerHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	customerID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid customer ID", ""))
		return
	}

	if err := h.customerRepo.Delete(c.Request.Context(), customerID); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("DELETE_ERROR", "Failed to delete customer", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(nil, "Customer deleted successfully"))
}

func getIntParam(c *gin.Context, key string, defaultValue int) int {
	value := c.GetInt(key)
	if value == 0 {
		return defaultValue
	}
	return value
}
