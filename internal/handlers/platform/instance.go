package platform

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"videostreamgo/internal/config"
	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
	"videostreamgo/internal/services/platform"
	"videostreamgo/internal/types"
)

// InstanceHandler handles instance endpoints
type InstanceHandler struct {
	instanceRepo        *masterRepo.InstanceRepository
	customerRepo        *masterRepo.CustomerRepository
	subscriptionRepo    *masterRepo.SubscriptionRepository
	instanceProvisioner platform.InstanceProvisionerInterface
	cfg                 *config.Config
}

// NewInstanceHandler creates a new InstanceHandler
func NewInstanceHandler(
	instanceRepo *masterRepo.InstanceRepository,
	customerRepo *masterRepo.CustomerRepository,
	subscriptionRepo *masterRepo.SubscriptionRepository,
	instanceProvisioner platform.InstanceProvisionerInterface,
	cfg *config.Config,
) *InstanceHandler {
	return &InstanceHandler{
		instanceRepo:        instanceRepo,
		customerRepo:        customerRepo,
		subscriptionRepo:    subscriptionRepo,
		instanceProvisioner: instanceProvisioner,
		cfg:                 cfg,
	}
}

// List returns all instances
func (h *InstanceHandler) List(c *gin.Context) {
	page := getIntParam(c, "page", 1)
	perPage := getIntParam(c, "per_page", 20)
	status := c.Query("status")

	instances, total, err := h.instanceRepo.List(c.Request.Context(), (page-1)*perPage, perPage, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("LIST_ERROR", "Failed to list instances", err.Error()))
		return
	}

	result := make([]map[string]interface{}, len(instances))
	for i, instance := range instances {
		customer, _ := h.customerRepo.GetByID(c.Request.Context(), instance.CustomerID)
		customerName := ""
		if customer != nil {
			customerName = customer.CompanyName
		}

		result[i] = map[string]interface{}{
			"id":            instance.ID,
			"customer_id":   instance.CustomerID,
			"customer_name": customerName,
			"name":          instance.Name,
			"subdomain":     instance.Subdomain,
			"status":        instance.Status,
			"database_name": instance.DatabaseName,
			"created_at":    instance.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"instances": result,
		"total":     total,
		"page":      page,
		"per_page":  perPage,
	}, ""))
}

// Create creates a new instance
func (h *InstanceHandler) Create(c *gin.Context) {
	var req struct {
		CustomerID string `json:"customer_id" binding:"required,uuid"`
		Name       string `json:"name" binding:"required,min=3"`
		Subdomain  string `json:"subdomain" binding:"required,min=3"`
		PlanID     string `json:"plan_id" binding:"required,uuid"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	customerID, _ := uuid.Parse(req.CustomerID)
	planID, _ := uuid.Parse(req.PlanID)

	// Generate database and storage names
	instanceID := uuid.New()
	databaseName := "instance_" + instanceID.String()[:8]
	storageBucket := "instance-" + instanceID.String()[:8]

	instance := &master.Instance{
		CustomerID:    customerID,
		Name:          req.Name,
		Subdomain:     req.Subdomain,
		Status:        master.InstanceStatusPending,
		PlanID:        &planID,
		DatabaseName:  databaseName,
		StorageBucket: storageBucket,
	}

	if err := h.instanceRepo.Create(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("CREATE_ERROR", "Failed to create instance", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.SuccessResponse(map[string]interface{}{
		"id":             instance.ID,
		"customer_id":    instance.CustomerID,
		"name":           instance.Name,
		"subdomain":      instance.Subdomain,
		"status":         instance.Status,
		"database_name":  instance.DatabaseName,
		"storage_bucket": instance.StorageBucket,
		"created_at":     instance.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, "Instance created successfully"))
}

// Get returns an instance by ID
func (h *InstanceHandler) Get(c *gin.Context) {
	id := c.Param("id")
	instanceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid instance ID", ""))
		return
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Instance not found", ""))
		return
	}

	customer, _ := h.customerRepo.GetByID(c.Request.Context(), instance.CustomerID)
	customerName := ""
	if customer != nil {
		customerName = customer.CompanyName
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":             instance.ID,
		"customer_id":    instance.CustomerID,
		"customer_name":  customerName,
		"name":           instance.Name,
		"subdomain":      instance.Subdomain,
		"custom_domains": instance.CustomDomains,
		"status":         instance.Status,
		"database_name":  instance.DatabaseName,
		"storage_bucket": instance.StorageBucket,
		"created_at":     instance.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":     instance.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, ""))
}

// Update updates an instance
func (h *InstanceHandler) Update(c *gin.Context) {
	id := c.Param("id")
	instanceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid instance ID", ""))
		return
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Instance not found", ""))
		return
	}

	var req struct {
		Name          *string  `json:"name"`
		Subdomain     *string  `json:"subdomain"`
		CustomDomains []string `json:"custom_domains"`
		Status        *string  `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	if req.Name != nil {
		instance.Name = *req.Name
	}
	if req.Subdomain != nil {
		instance.Subdomain = *req.Subdomain
	}
	if req.CustomDomains != nil {
		instance.CustomDomains = req.CustomDomains
	}
	if req.Status != nil {
		instance.Status = master.InstanceStatus(*req.Status)
	}

	if err := h.instanceRepo.Update(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to update instance", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":         instance.ID,
		"name":       instance.Name,
		"subdomain":  instance.Subdomain,
		"status":     instance.Status,
		"updated_at": instance.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, "Instance updated successfully"))
}

// Delete soft deletes an instance
func (h *InstanceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	instanceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid instance ID", ""))
		return
	}

	if err := h.instanceRepo.Delete(c.Request.Context(), instanceID); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("DELETE_ERROR", "Failed to delete instance", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(nil, "Instance deleted successfully"))
}

// Provision provisions an instance with all resources
func (h *InstanceHandler) Provision(c *gin.Context) {
	id := c.Param("id")
	instanceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid instance ID", ""))
		return
	}

	// Check if instance exists
	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Instance not found", ""))
		return
	}

	// Only pending instances can be provisioned
	if instance.Status != master.InstanceStatusPending {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_STATUS", "Only pending instances can be provisioned", ""))
		return
	}

	// Start provisioning asynchronously
	go func() {
		_, err := h.instanceProvisioner.ProvisionInstance(c.Request.Context(), instanceID)
		if err != nil {
			// Log the error - in production, you'd want to handle this better
			println("Provisioning failed:", err.Error())
		}
	}()

	c.JSON(http.StatusAccepted, types.SuccessResponse(map[string]interface{}{
		"id":     instance.ID,
		"status": master.InstanceStatusProvisioning,
	}, "Instance provisioning started"))
}

// Deprovision deprovisions an instance
func (h *InstanceHandler) Deprovision(c *gin.Context) {
	id := c.Param("id")
	instanceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid instance ID", ""))
		return
	}

	// Check if instance exists
	_, err = h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Instance not found", ""))
		return
	}

	// Start deprovisioning asynchronously
	go func() {
		err := h.instanceProvisioner.DeprovisionInstance(c.Request.Context(), instanceID)
		if err != nil {
			println("Deprovisioning failed:", err.Error())
		}
	}()

	c.JSON(http.StatusAccepted, types.SuccessResponse(map[string]interface{}{
		"id":     instanceID,
		"status": master.InstanceStatusTerminated,
	}, "Instance deprovisioning started"))
}

// Status returns the provisioning status of an instance
func (h *InstanceHandler) Status(c *gin.Context) {
	id := c.Param("id")
	instanceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid instance ID", ""))
		return
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Instance not found", ""))
		return
	}

	progress, err := h.instanceProvisioner.GetProvisioningProgress(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("STATUS_ERROR", "Failed to get provisioning status", err.Error()))
		return
	}

	// Get metrics if instance is active
	var metrics map[string]interface{}
	if instance.Status == master.InstanceStatusActive {
		metrics, _ = h.instanceProvisioner.GetInstanceMetrics(c.Request.Context(), instanceID)
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":         instance.ID,
		"name":       instance.Name,
		"subdomain":  instance.Subdomain,
		"status":     instance.Status,
		"progress":   progress,
		"metrics":    metrics,
		"created_at": instance.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"activated_at": func() *string {
			if instance.ActivatedAt != nil {
				s := instance.ActivatedAt.Format("2006-01-02T15:04:05Z07:00")
				return &s
			}
			return nil
		}(),
	}, ""))
}

// CustomDomains adds a custom domain to an instance
func (h *InstanceHandler) CustomDomains(c *gin.Context) {
	id := c.Param("id")
	instanceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid instance ID", ""))
		return
	}

	var req struct {
		Domain string `json:"domain" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	// Add custom domain
	err = h.instanceProvisioner.AddCustomDomain(c.Request.Context(), instanceID, req.Domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("DOMAIN_ERROR", "Failed to add custom domain", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":     instanceID,
		"domain": req.Domain,
	}, "Custom domain added successfully"))
}

// Metrics returns usage metrics for an instance
func (h *InstanceHandler) Metrics(c *gin.Context) {
	id := c.Param("id")
	instanceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid instance ID", ""))
		return
	}

	metrics, err := h.instanceProvisioner.GetInstanceMetrics(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("METRICS_ERROR", "Failed to get metrics", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(metrics, ""))
}

// Suspend suspends an instance
func (h *InstanceHandler) Suspend(c *gin.Context) {
	id := c.Param("id")
	instanceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid instance ID", ""))
		return
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Instance not found", ""))
		return
	}

	instance.Status = master.InstanceStatusSuspended
	if err := h.instanceRepo.Update(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to suspend instance", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":     instance.ID,
		"status": instance.Status,
	}, "Instance suspended successfully"))
}

// Activate activates an instance
func (h *InstanceHandler) Activate(c *gin.Context) {
	id := c.Param("id")
	instanceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid instance ID", ""))
		return
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Instance not found", ""))
		return
	}

	instance.Status = master.InstanceStatusActive
	if err := h.instanceRepo.Update(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to activate instance", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":     instance.ID,
		"status": instance.Status,
	}, "Instance activated successfully"))
}
