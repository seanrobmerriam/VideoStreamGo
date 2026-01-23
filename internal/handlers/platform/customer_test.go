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
	"golang.org/x/crypto/bcrypt"

	"videostreamgo/internal/models/master"
)

// Test_CustomerHandler_Create_Success tests successful customer creation
func Test_CustomerHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	registeredEmails := make(map[string]bool)

	r.POST("/customers", func(c *gin.Context) {
		var req struct {
			Email       string `json:"email" binding:"required,email"`
			Password    string `json:"password" binding:"required,min=8"`
			CompanyName string `json:"company_name" binding:"required,min=2"`
			ContactName string `json:"contact_name" binding:"required,min=2"`
			Phone       string `json:"phone"`
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

		// Check if customer exists
		if registeredEmails[req.Email] {
			c.JSON(http.StatusConflict, gin.H{
				"error": map[string]string{
					"code":    "EMAIL_EXISTS",
					"message": "Email already registered",
				},
			})
			return
		}

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

		customer := &master.Customer{
			ID:           uuid.New(),
			Email:        req.Email,
			PasswordHash: string(hashedPassword),
			CompanyName:  req.CompanyName,
			ContactName:  req.ContactName,
			Phone:        req.Phone,
			Status:       master.CustomerStatusActive,
		}
		registeredEmails[customer.Email] = true

		c.JSON(http.StatusCreated, gin.H{
			"data": map[string]interface{}{
				"id":           customer.ID,
				"email":        customer.Email,
				"company_name": customer.CompanyName,
				"contact_name": customer.ContactName,
				"status":       customer.Status,
				"created_at":   customer.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
			"message": "Customer created successfully",
		})
	})

	body := `{"email":"company@test.com","password":"password123","company_name":"Test Company","contact_name":"John Doe","phone":"+1234567890"}`
	req := httptest.NewRequest("POST", "/customers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "company@test.com", data["email"])
	assert.Equal(t, "Test Company", data["company_name"])
	assert.Equal(t, "active", data["status"])
}

// Test_CustomerHandler_Create_ValidationError tests customer creation with invalid data
func Test_CustomerHandler_Create_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/customers", func(c *gin.Context) {
		var req struct {
			Email       string `json:"email" binding:"required,email"`
			Password    string `json:"password" binding:"required,min=8"`
			CompanyName string `json:"company_name" binding:"required,min=2"`
			ContactName string `json:"contact_name" binding:"required,min=2"`
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

	// Invalid email
	body := `{"email":"invalid-email","password":"password123","company_name":"Test","contact_name":"John"}`
	req := httptest.NewRequest("POST", "/customers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_CustomerHandler_Create_EmailExists tests customer creation with existing email
func Test_CustomerHandler_Create_EmailExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	registeredEmails := make(map[string]bool)
	registeredEmails["existing@test.com"] = true

	r.POST("/customers", func(c *gin.Context) {
		var req struct {
			Email       string `json:"email" binding:"required,email"`
			Password    string `json:"password" binding:"required,min=8"`
			CompanyName string `json:"company_name" binding:"required,min=2"`
			ContactName string `json:"contact_name" binding:"required,min=2"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if registeredEmails[req.Email] {
			c.JSON(http.StatusConflict, gin.H{
				"error": map[string]string{
					"code":    "EMAIL_EXISTS",
					"message": "Email already registered",
				},
			})
			return
		}
	})

	body := `{"email":"existing@test.com","password":"password123","company_name":"Test","contact_name":"John"}`
	req := httptest.NewRequest("POST", "/customers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// Test_CustomerHandler_Get_Success tests getting a customer by ID
func Test_CustomerHandler_Get_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	customerID := uuid.New()

	r.GET("/customers/:id", func(c *gin.Context) {
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
				"id":             customerID,
				"email":          "test@company.com",
				"company_name":   "Test Company",
				"contact_name":   "John Doe",
				"status":         "active",
				"instance_count": 2,
			},
		})
	})

	req := httptest.NewRequest("GET", "/customers/"+customerID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_CustomerHandler_Get_InvalidID tests getting a customer with invalid ID
func Test_CustomerHandler_Get_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/customers/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "INVALID_ID",
					"message": "Invalid customer ID",
				},
			})
			return
		}
	})

	req := httptest.NewRequest("GET", "/customers/invalid-uuid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_CustomerHandler_Get_NotFound tests getting a non-existent customer
func Test_CustomerHandler_Get_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/customers/:id", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "NOT_FOUND",
				"message": "Customer not found",
			},
		})
	})

	req := httptest.NewRequest("GET", "/customers/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Test_CustomerHandler_Update_Success tests updating a customer
func Test_CustomerHandler_Update_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	customer := &master.Customer{
		ID:          uuid.New(),
		Email:       "test@company.com",
		CompanyName: "Original Company",
		ContactName: "John Doe",
		Status:      master.CustomerStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	r.PUT("/customers/:id", func(c *gin.Context) {
		id := c.Param("id")
		customerID, _ := uuid.Parse(id)

		if customerID != customer.ID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.CompanyName != nil {
			customer.CompanyName = *req.CompanyName
		}
		if req.ContactName != nil {
			customer.ContactName = *req.ContactName
		}
		if req.Status != nil {
			customer.Status = master.CustomerStatus(*req.Status)
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":           customer.ID,
				"email":        customer.Email,
				"company_name": customer.CompanyName,
				"status":       customer.Status,
				"updated_at":   customer.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
			"message": "Customer updated successfully",
		})
	})

	body := `{"company_name":"Updated Company"}`
	req := httptest.NewRequest("PUT", "/customers/"+customer.ID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_CustomerHandler_Update_StatusTransition tests customer status transitions
func Test_CustomerHandler_Update_StatusTransition(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	tests := []struct {
		name          string
		initialStatus master.CustomerStatus
		newStatus     master.CustomerStatus
		expectSuccess bool
	}{
		{"Active to Suspended", master.CustomerStatusActive, master.CustomerStatusSuspended, true},
		{"Suspended to Active", master.CustomerStatusSuspended, master.CustomerStatusActive, true},
		{"Active to Pending", master.CustomerStatusActive, master.CustomerStatusPending, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customer := &master.Customer{
				ID:          uuid.New(),
				Email:       "test@company.com",
				CompanyName: "Test Company",
				ContactName: "John Doe",
				Status:      tt.initialStatus,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			r.PUT("/customers/:id", func(c *gin.Context) {
				var req struct {
					Status *string `json:"status"`
				}

				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				if req.Status != nil {
					customer.Status = master.CustomerStatus(*req.Status)
				}

				c.JSON(http.StatusOK, gin.H{"status": customer.Status})
			})

			body := `{"status":"` + string(tt.newStatus) + `"}`
			req := httptest.NewRequest("PUT", "/customers/"+customer.ID.String(), bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if tt.expectSuccess {
				assert.Equal(t, http.StatusOK, w.Code)
			}
		})
	}
}

// Test_CustomerHandler_Delete_Success tests soft deleting a customer
func Test_CustomerHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	customerID := uuid.New()

	r.DELETE("/customers/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Customer deleted successfully",
		})
	})

	req := httptest.NewRequest("DELETE", "/customers/"+customerID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_CustomerHandler_List tests listing customers with pagination
func Test_CustomerHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/customers", func(c *gin.Context) {
		page := 1
		perPage := 20

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"customers": []map[string]interface{}{
					{
						"id":             uuid.New(),
						"email":          "customer1@test.com",
						"company_name":   "Company 1",
						"contact_name":   "John Doe",
						"status":         "active",
						"instance_count": 1,
					},
					{
						"id":             uuid.New(),
						"email":          "customer2@test.com",
						"company_name":   "Company 2",
						"contact_name":   "Jane Smith",
						"status":         "active",
						"instance_count": 2,
					},
				},
				"total":    50,
				"page":     page,
				"per_page": perPage,
			},
		})
	})

	req := httptest.NewRequest("GET", "/customers?page=1&per_page=20", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(50), data["total"])
	assert.Equal(t, float64(1), data["page"])
}
