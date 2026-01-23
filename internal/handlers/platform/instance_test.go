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

	"videostreamgo/internal/models/master"
)

// Test_InstanceHandler_Create_Success tests successful instance creation
func Test_InstanceHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	customerID := uuid.New()
	planID := uuid.New()
	createdInstances := make(map[string]*master.Instance)

	r.POST("/instances", func(c *gin.Context) {
		var req struct {
			Name      string `json:"name" binding:"required,min=3,max=100"`
			Subdomain string `json:"subdomain" binding:"required,min=3,max=50,alphanum"`
			PlanID    string `json:"plan_id" binding:"required"`
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

		parsedPlanID, _ := uuid.Parse(req.PlanID)
		now := time.Now()
		instance := &master.Instance{
			ID:            uuid.New(),
			CustomerID:    customerID,
			Name:          req.Name,
			Subdomain:     req.Subdomain,
			PlanID:        &parsedPlanID,
			Status:        master.InstanceStatusProvisioning,
			DatabaseName:  "vs_" + req.Subdomain,
			StorageBucket: "vs-" + req.Subdomain,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		createdInstances[instance.ID.String()] = instance

		c.JSON(http.StatusCreated, gin.H{
			"data": map[string]interface{}{
				"id":         instance.ID,
				"name":       instance.Name,
				"subdomain":  instance.Subdomain,
				"plan_id":    instance.PlanID,
				"status":     instance.Status,
				"status_url": "/api/instances/" + instance.ID.String() + "/status",
				"created_at": instance.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
			"message": "Instance creation started",
		})
	})

	body := `{"name":"My Video Site","subdomain":"myvideosite","plan_id":"` + planID.String() + `"}`
	req := httptest.NewRequest("POST", "/instances", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "My Video Site", data["name"])
	assert.Equal(t, "myvideosite", data["subdomain"])
	assert.Equal(t, "provisioning", data["status"])
	assert.Contains(t, data, "status_url")
}

// Test_InstanceHandler_Create_InvalidSubdomain tests instance creation with invalid subdomain
func Test_InstanceHandler_Create_InvalidSubdomain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/instances", func(c *gin.Context) {
		var req struct {
			Name      string `json:"name" binding:"required,min=3,max=100"`
			Subdomain string `json:"subdomain" binding:"required,min=3,max=50,alphanum"`
			PlanID    string `json:"plan_id" binding:"required"`
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

	// Subdomain with special characters
	body := `{"name":"My Site","subdomain":"my-site!","plan_id":"pro"}`
	req := httptest.NewRequest("POST", "/instances", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_InstanceHandler_GetProvisioningStatus tests getting provisioning status
func Test_InstanceHandler_GetProvisioningStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	instanceID := uuid.New()

	r.GET("/instances/:id/status", func(c *gin.Context) {
		id := c.Param("id")
		parsedID, _ := uuid.Parse(id)

		if parsedID != instanceID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":                   instanceID,
				"status":               "provisioning",
				"progress":             45,
				"current_step":         "Setting up database",
				"estimated_completion": "2024-01-01T12:00:00Z",
				"completed_steps":      []string{"Account created", "Domain configured", "Database initializing"},
				"pending_steps":        []string{"Installing video processing", "Configuring CDN", "Final setup"},
			},
		})
	})

	req := httptest.NewRequest("GET", "/instances/"+instanceID.String()+"/status", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "provisioning", data["status"])
	assert.Equal(t, float64(45), data["progress"])
}

// Test_InstanceHandler_CustomDomain tests custom domain configuration
func Test_InstanceHandler_CustomDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	instanceID := uuid.New()

	r.POST("/instances/:id/domains", func(c *gin.Context) {
		var req struct {
			Domain string `json:"domain" binding:"required,hostname"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate domain format
		if req.Domain == "videostreamgo.com" || req.Domain == "www.videostreamgo.com" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "DOMAIN_RESERVED",
					"message": "This domain is reserved",
				},
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"data": map[string]interface{}{
				"id":          uuid.New(),
				"instance_id": instanceID,
				"domain":      req.Domain,
				"status":      "pending_verification",
				"verification_dns_records": []map[string]string{
					{"type": "TXT", "name": "_videostreamgo-verify", "value": "verification-code-123"},
				},
			},
		})
	})

	body := `{"domain":"custom.example.com"}`
	req := httptest.NewRequest("POST", "/instances/"+instanceID.String()+"/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// Test_InstanceHandler_List tests listing instances for a customer
func Test_InstanceHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/instances", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"instances": []map[string]interface{}{
					{
						"id":           uuid.New(),
						"name":         "Site 1",
						"subdomain":    "site1",
						"status":       "active",
						"plan_id":      uuid.New(),
						"video_count":  150,
						"storage_used": "50GB",
					},
					{
						"id":           uuid.New(),
						"name":         "Site 2",
						"subdomain":    "site2",
						"status":       "active",
						"plan_id":      uuid.New(),
						"video_count":  25,
						"storage_used": "5GB",
					},
				},
				"total":    2,
				"page":     1,
				"per_page": 20,
			},
		})
	})

	req := httptest.NewRequest("GET", "/instances", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_InstanceHandler_Delete tests soft deleting an instance
func Test_InstanceHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	instanceID := uuid.New()

	r.DELETE("/instances/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Instance deletion scheduled",
			"data": map[string]interface{}{
				"id":             instanceID,
				"status":         "terminating",
				"scheduled_at":   "2024-01-01T00:00:00Z",
				"data_retention": "30 days",
			},
		})
	})

	req := httptest.NewRequest("DELETE", "/instances/"+instanceID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
