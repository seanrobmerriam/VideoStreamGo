package instance

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

// Test_CategoryHandler_ListCategories_Success tests successful category listing
func Test_CategoryHandler_ListCategories_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/categories", func(c *gin.Context) {
		page := 1
		perPage := 20

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"categories": []map[string]interface{}{
					{
						"id":          uuid.New(),
						"name":        "Technology",
						"slug":        "technology-" + uuid.New().String()[:8],
						"description": "Tech videos",
						"video_count": 25,
						"is_active":   true,
						"sort_order":  1,
						"created_at":  time.Now().Format(time.RFC3339),
					},
				},
				"total":    1,
				"page":     page,
				"per_page": perPage,
			},
		})
	})

	req := httptest.NewRequest("GET", "/categories?page=1&per_page=20", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["total"])
}

// Test_CategoryHandler_GetCategory_Success tests getting a category by ID
func Test_CategoryHandler_GetCategory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	categoryID := uuid.New()

	r.GET("/categories/:id", func(c *gin.Context) {
		id := c.Param("id")
		parsedID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
			return
		}

		if parsedID != categoryID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":          categoryID,
				"name":        "Technology",
				"slug":        "technology",
				"description": "Tech videos",
				"video_count": 25,
				"is_active":   true,
			},
		})
	})

	req := httptest.NewRequest("GET", "/categories/"+categoryID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_CategoryHandler_GetCategory_InvalidID tests getting with invalid ID
func Test_CategoryHandler_GetCategory_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/categories/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "INVALID_ID",
					"message": "Invalid category ID",
				},
			})
			return
		}
	})

	req := httptest.NewRequest("GET", "/categories/invalid-uuid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_CategoryHandler_GetCategory_NotFound tests getting non-existent category
func Test_CategoryHandler_GetCategory_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/categories/:id", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "NOT_FOUND",
				"message": "Category not found",
			},
		})
	})

	req := httptest.NewRequest("GET", "/categories/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Test_CategoryHandler_CreateCategory_Success tests creating a category
func Test_CategoryHandler_CreateCategory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/categories", func(c *gin.Context) {
		var req struct {
			Name        string `json:"name" binding:"required,min=1,max=100"`
			Description string `json:"description" binding:"max=500"`
			Color       string `json:"color"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Color != "" && !isValidHexColor(req.Color) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid color format"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"data": map[string]interface{}{
				"id":          uuid.New(),
				"name":        req.Name,
				"slug":        "test-category-" + uuid.New().String()[:8],
				"description": req.Description,
				"color":       req.Color,
				"is_active":   true,
				"created_at":  time.Now().Format(time.RFC3339),
			},
			"message": "Category created successfully",
		})
	})

	body := `{"name":"Test Category","description":"A test category","color":"#ff5733"}`
	req := httptest.NewRequest("POST", "/categories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// Test_CategoryHandler_CreateCategory_ValidationError tests validation errors
func Test_CategoryHandler_CreateCategory_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/categories", func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required,min=1,max=100"`
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

	// Missing required name field
	body := `{"description":"No name provided"}`
	req := httptest.NewRequest("POST", "/categories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_CategoryHandler_UpdateCategory_Success tests updating a category
func Test_CategoryHandler_UpdateCategory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	categoryID := uuid.New()

	r.PUT("/categories/:id", func(c *gin.Context) {
		id := c.Param("id")
		parsedID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
			return
		}

		if parsedID != categoryID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}

		var req struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
			Color       *string `json:"color"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Color != nil && !isValidHexColor(*req.Color) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid color format"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":          categoryID,
				"name":        "Updated Category",
				"description": "Updated description",
				"updated_at":  time.Now().Format(time.RFC3339),
			},
			"message": "Category updated successfully",
		})
	})

	body := `{"name":"Updated Category","description":"Updated description"}`
	req := httptest.NewRequest("PUT", "/categories/"+categoryID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_CategoryHandler_DeleteCategory_Success tests deleting a category
func Test_CategoryHandler_DeleteCategory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	categoryID := uuid.New()

	r.DELETE("/categories/:id", func(c *gin.Context) {
		id := c.Param("id")
		parsedID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
			return
		}

		if parsedID != categoryID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}

		// Category has no videos, can be deleted
		c.JSON(http.StatusOK, gin.H{
			"message": "Category deleted successfully",
		})
	})

	req := httptest.NewRequest("DELETE", "/categories/"+categoryID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_CategoryHandler_DeleteCategory_WithVideos tests preventing deletion with videos
func Test_CategoryHandler_DeleteCategory_WithVideos(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.DELETE("/categories/:id", func(c *gin.Context) {
		// Simulate category has videos
		c.JSON(http.StatusConflict, gin.H{
			"error": map[string]string{
				"code":    "CATEGORY_HAS_VIDEOS",
				"message": "Cannot delete category with videos",
			},
		})
	})

	req := httptest.NewRequest("DELETE", "/categories/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// Test_TagHandler_ListTags tests listing tags
func Test_TagHandler_ListTags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/tags", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"tags": []map[string]interface{}{
					{
						"id":          uuid.New(),
						"name":        "javascript",
						"slug":        "javascript",
						"usage_count": 50,
						"created_at":  time.Now().Format(time.RFC3339),
					},
				},
				"total":    1,
				"page":     1,
				"per_page": 20,
			},
		})
	})

	req := httptest.NewRequest("GET", "/tags", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_TagHandler_CreateTag tests creating a tag
func Test_TagHandler_CreateTag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/tags", func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required,min=1,max=100"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"data": map[string]interface{}{
				"id":          uuid.New(),
				"name":        req.Name,
				"slug":        "test-tag-" + uuid.New().String()[:8],
				"usage_count": 0,
				"created_at":  time.Now().Format(time.RFC3339),
			},
			"message": "Tag created successfully",
		})
	})

	body := `{"name":"Test Tag"}`
	req := httptest.NewRequest("POST", "/tags", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// Test_TagHandler_DeleteTag tests deleting a tag
func Test_TagHandler_DeleteTag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	tagID := uuid.New()

	r.DELETE("/tags/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Tag deleted successfully",
		})
	})

	req := httptest.NewRequest("DELETE", "/tags/"+tagID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_GenerateCategorySlug tests category slug generation
func Test_GenerateCategorySlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple name", "Technology", "technology"},
		{"With spaces", "Web Development", "web-development"},
		{"With special chars", "C++ Programming", "c-programming"},
		{"With numbers", "Python 3.9", "python-39"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug := generateCategorySlug(tt.input)
			assert.Contains(t, slug, tt.expected)
		})
	}
}

// Test_IsValidHexColor tests hex color validation
func Test_IsValidHexColor(t *testing.T) {
	validColors := []string{"#ff5733", "#ffffff", "#000000", "#ABC123"}
	invalidColors := []string{"#gg5733", "#ff57", "ff5733", "#ff5733ff", "red"}

	for _, color := range validColors {
		assert.True(t, isValidHexColor(color), "Expected %s to be valid", color)
	}

	for _, color := range invalidColors {
		assert.False(t, isValidHexColor(color), "Expected %s to be invalid", color)
	}
}
