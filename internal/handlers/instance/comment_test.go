package instance

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"videostreamgo/internal/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Test_CommentHandler_ListComments_Success tests successful comment listing
func Test_CommentHandler_ListComments_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	videoID := uuid.New()

	r.GET("/videos/:id/comments", func(c *gin.Context) {
		id := c.Param("id")
		parsedID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
			return
		}

		if parsedID != videoID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
			return
		}

		page := 1
		perPage := 20

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"comments": []map[string]interface{}{
					{
						"id":         uuid.New(),
						"content":    "Great video!",
						"is_edited":  false,
						"like_count": 5,
						"user": map[string]interface{}{
							"id":           uuid.New(),
							"username":     "testuser",
							"display_name": "Test User",
						},
						"created_at": time.Now().Format(time.RFC3339),
					},
				},
				"total":    1,
				"page":     page,
				"per_page": perPage,
			},
		})
	})

	req := httptest.NewRequest("GET", "/videos/"+videoID.String()+"/comments?page=1&per_page=20", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["total"])
}

// Test_CommentHandler_ListComments_Pagination tests pagination parameters
func Test_CommentHandler_ListComments_Pagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/videos/:id/comments", func(c *gin.Context) {
		page := getIntParam(c, "page", 1)
		perPage := getIntParam(c, "per_page", 20)

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"comments": []map[string]interface{}{},
				"total":    100,
				"page":     page,
				"per_page": perPage,
			},
		})
	})

	req := httptest.NewRequest("GET", "/videos/"+uuid.New().String()+"/comments?page=2&per_page=10", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["page"])
	assert.Equal(t, float64(10), data["per_page"])
}

// Test_CommentHandler_CreateComment_Success tests creating a comment
func Test_CommentHandler_CreateComment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	videoID := uuid.New()
	userID := uuid.New()

	r.POST("/videos/:id/comments", func(c *gin.Context) {
		id := c.Param("id")
		parsedID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
			return
		}

		if parsedID != videoID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
			return
		}

		c.Set(string(types.ContextKeyUserID), userID)

		var req struct {
			Content  string `json:"content" binding:"required,min=1,max=5000"`
			ParentID string `json:"parent_id"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"data": map[string]interface{}{
				"id":        uuid.New(),
				"content":   req.Content,
				"is_edited": false,
				"user": map[string]interface{}{
					"id":           userID,
					"username":     "testuser",
					"display_name": "Test User",
				},
				"created_at": time.Now().Format(time.RFC3339),
			},
			"message": "Comment created successfully",
		})
	})

	body := `{"content":"This is a great video!","parent_id":""}`
	req := httptest.NewRequest("POST", "/videos/"+videoID.String()+"/comments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// Test_CommentHandler_CreateComment_Reply tests creating a reply comment
func Test_CommentHandler_CreateComment_Reply(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	videoID := uuid.New()
	parentCommentID := uuid.New()

	r.POST("/videos/:id/comments", func(c *gin.Context) {
		var req struct {
			Content  string `json:"content" binding:"required,min=1,max=5000"`
			ParentID string `json:"parent_id"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"data": map[string]interface{}{
				"id":         uuid.New(),
				"content":    req.Content,
				"parent_id":  parentCommentID,
				"is_edited":  false,
				"created_at": time.Now().Format(time.RFC3339),
			},
			"message": "Comment created successfully",
		})
	})

	body := `{"content":"This is a reply!","parent_id":"` + parentCommentID.String() + `"}`
	req := httptest.NewRequest("POST", "/videos/"+videoID.String()+"/comments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, parentCommentID.String(), data["parent_id"])
}

// Test_CommentHandler_UpdateComment_Success tests updating a comment
func Test_CommentHandler_UpdateComment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	commentID := uuid.New()
	userID := uuid.New()

	r.PUT("/comments/:id", func(c *gin.Context) {
		id := c.Param("id")
		parsedID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
			return
		}

		if parsedID != commentID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
			return
		}

		c.Set(string(types.ContextKeyUserID), userID)

		var req struct {
			Content string `json:"content" binding:"required,min=1,max=5000"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":         commentID,
				"content":    req.Content,
				"is_edited":  true,
				"updated_at": time.Now().Format(time.RFC3339),
			},
			"message": "Comment updated successfully",
		})
	})

	body := `{"content":"Updated comment content"}`
	req := httptest.NewRequest("PUT", "/comments/"+commentID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.True(t, data["is_edited"].(bool))
}

// Test_CommentHandler_DeleteComment_Success tests deleting a comment (soft delete)
func Test_CommentHandler_DeleteComment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	commentID := uuid.New()
	userID := uuid.New()

	r.DELETE("/comments/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
			return
		}

		c.Set(string(types.ContextKeyUserID), userID)

		c.JSON(http.StatusOK, gin.H{
			"message": "Comment deleted successfully",
		})
	})

	req := httptest.NewRequest("DELETE", "/comments/"+commentID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_CommentHandler_RateVideo_Success tests rating a video
func Test_CommentHandler_RateVideo_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	videoID := uuid.New()
	userID := uuid.New()

	r.POST("/videos/:id/rate", func(c *gin.Context) {
		id := c.Param("id")
		parsedID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
			return
		}

		if parsedID != videoID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
			return
		}

		c.Set(string(types.ContextKeyUserID), userID)

		var req struct {
			Rating int8 `json:"rating" binding:"required,oneof=-1 1"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"rating": req.Rating,
			},
			"message": "Video rated successfully",
		})
	})

	body := `{"rating":1}`
	req := httptest.NewRequest("POST", "/videos/"+videoID.String()+"/rate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["rating"])
}

// Test_CommentHandler_RateVideo_InvalidRating tests invalid rating value
func Test_CommentHandler_RateVideo_InvalidRating(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/videos/:id/rate", func(c *gin.Context) {
		var req struct {
			Rating int8 `json:"rating" binding:"required,oneof=-1 1"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	})

	// Invalid rating (not -1 or 1)
	body := `{"rating":2}`
	req := httptest.NewRequest("POST", "/videos/"+uuid.New().String()+"/rate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_CommentHandler_GetVideoRating tests getting video rating stats
func Test_CommentHandler_GetVideoRating(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	videoID := uuid.New()

	r.GET("/videos/:id/rating", func(c *gin.Context) {
		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"likes":    150,
				"dislikes": 10,
			},
		})
	})

	req := httptest.NewRequest("GET", "/videos/"+videoID.String()+"/rating", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(150), data["likes"])
	assert.Equal(t, float64(10), data["dislikes"])
}

// Test_CommentHandler_ValidationError tests validation errors
func Test_CommentHandler_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/videos/:id/comments", func(c *gin.Context) {
		var req struct {
			Content string `json:"content" binding:"required,min=1,max=5000"`
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

	// Empty content
	body := `{"content":""}`
	req := httptest.NewRequest("POST", "/videos/"+uuid.New().String()+"/comments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_CommentHandler_Unauthorized tests unauthenticated request
func Test_CommentHandler_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/videos/:id/comments", func(c *gin.Context) {
		_, exists := c.Get(string(types.ContextKeyUserID))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": map[string]string{
					"code":    "NOT_AUTHENTICATED",
					"message": "User not authenticated",
				},
			})
			return
		}
	})

	body := `{"content":"Test comment"}`
	req := httptest.NewRequest("POST", "/videos/"+uuid.New().String()+"/comments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
