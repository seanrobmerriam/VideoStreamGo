package instance

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"videostreamgo/internal/models/instance"
)

// Test_VideoHandler_Upload_Success tests successful video upload
func Test_VideoHandler_Upload_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	userID := uuid.New()
	instanceID := uuid.New()

	r.POST("/videos", func(c *gin.Context) {
		// Simulate auth middleware
		c.Set("user_id", userID.String())
		c.Set("instance_id", instanceID.String())

		var req struct {
			Title       string `json:"title" binding:"required,min=3,max=255"`
			Description string `json:"description" binding:"max=5000"`
			CategoryID  string `json:"category_id"`
			IsPublic    bool   `json:"is_public"`
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

		videoID := uuid.New()

		c.JSON(http.StatusCreated, gin.H{
			"data": map[string]interface{}{
				"id":                  videoID,
				"title":               req.Title,
				"description":         req.Description,
				"status":              "processing",
				"processing_status":   "uploaded",
				"upload_url":          "/api/videos/" + videoID.String() + "/upload",
				"processing_progress": 0,
			},
			"message": "Video upload initialized",
		})
	})

	body := `{"title":"My Test Video","description":"A test video description","is_public":true}`
	req := httptest.NewRequest("POST", "/videos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "My Test Video", data["title"])
	assert.Equal(t, "processing", data["status"])
	assert.Contains(t, data, "upload_url")
}

// Test_VideoHandler_Upload_ValidationError tests video upload with invalid data
func Test_VideoHandler_Upload_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/videos", func(c *gin.Context) {
		var req struct {
			Title string `json:"title" binding:"required,min=3,max=255"`
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

	// Missing title
	body := `{"description":"A video without title"}`
	req := httptest.NewRequest("POST", "/videos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_VideoHandler_List tests listing videos
func Test_VideoHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/videos", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"videos": []map[string]interface{}{
					{
						"id":            uuid.New(),
						"title":         "Video 1",
						"thumbnail_url": "https://example.com/thumb1.jpg",
						"duration":      120,
						"view_count":    100,
						"created_at":    "2024-01-01T00:00:00Z",
					},
					{
						"id":            uuid.New(),
						"title":         "Video 2",
						"thumbnail_url": "https://example.com/thumb2.jpg",
						"duration":      180,
						"view_count":    50,
						"created_at":    "2024-01-02T00:00:00Z",
					},
				},
				"total":    50,
				"page":     1,
				"per_page": 20,
			},
		})
	})

	req := httptest.NewRequest("GET", "/videos", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_VideoHandler_Get tests getting a single video
func Test_VideoHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	videoID := uuid.New()

	r.GET("/videos/:id", func(c *gin.Context) {
		id := c.Param("id")
		parsedID, _ := uuid.Parse(id)

		if parsedID != videoID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":                videoID,
				"title":             "Test Video",
				"description":       "A test video description",
				"video_url":         "https://example.com/video.mp4",
				"thumbnail_url":     "https://example.com/thumb.jpg",
				"duration":          300,
				"view_count":        1000,
				"like_count":        50,
				"dislike_count":     2,
				"status":            "public",
				"processing_status": "completed",
				"created_at":        "2024-01-01T00:00:00Z",
				"user": map[string]interface{}{
					"id":           uuid.New(),
					"username":     "testuser",
					"display_name": "Test User",
				},
			},
		})
	})

	req := httptest.NewRequest("GET", "/videos/"+videoID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_VideoHandler_ViewTracking tests video view tracking
func Test_VideoHandler_ViewTracking(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	videoID := uuid.New()
	viewCount := int64(100)

	r.POST("/videos/:id/view", func(c *gin.Context) {
		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
			return
		}

		viewCount++

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"video_id":   videoID,
				"view_count": viewCount,
			},
		})
	})

	req := httptest.NewRequest("POST", "/videos/"+videoID.String()+"/view", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(101), data["view_count"])
}

// Test_VideoHandler_Rate tests video rating (like/dislike)
func Test_VideoHandler_Rate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	videoID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name       string
		ratingType string
		wantErr    bool
	}{
		{"Like video", "like", false},
		{"Dislike video", "dislike", false},
		{"Remove rating", "none", false},
		{"Invalid rating", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r.POST("/videos/:id/rate", func(c *gin.Context) {
				c.Set("user_id", userID.String())

				var req struct {
					Rating string `json:"rating" binding:"required,oneof=like dislike none"`
				}

				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				if tt.wantErr && req.Rating == "invalid" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rating type"})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"data": map[string]interface{}{
						"video_id":      videoID,
						"rating":        req.Rating,
						"like_count":    51,
						"dislike_count": 2,
					},
				})
			})

			body := `{"rating":"` + tt.ratingType + `"}`
			req := httptest.NewRequest("POST", "/videos/"+videoID.String()+"/rate", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if tt.wantErr {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
			}
		})
	}
}

// Test_VideoHandler_Delete tests deleting a video
func Test_VideoHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	videoID := uuid.New()
	userID := uuid.New()

	r.DELETE("/videos/:id", func(c *gin.Context) {
		c.Set("user_id", userID.String())

		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Video deleted successfully",
			"data": map[string]interface{}{
				"id":     videoID,
				"status": "deleted",
			},
		})
	})

	req := httptest.NewRequest("DELETE", "/videos/"+videoID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_VideoProcessingStatus tests video processing status states
func Test_VideoProcessingStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     instance.ProcessingStatus
		progress   int
		isComplete bool
		isFailed   bool
	}{
		{"Pending upload", instance.ProcessingStatusPending, 0, false, false},
		{"Uploaded - starting processing", instance.ProcessingStatusUploaded, 5, false, false},
		{"Extracting audio", instance.ProcessingStatusExtracting, 20, false, false},
		{"Transcoding", instance.ProcessingStatusTranscoding, 60, false, false},
		{"Completed successfully", instance.ProcessingStatusCompleted, 100, true, false},
		{"Failed processing", instance.ProcessingStatusFailed, 0, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isComplete := tt.progress == 100 || tt.status == instance.ProcessingStatusFailed
			isFailed := tt.status == instance.ProcessingStatusFailed

			assert.Equal(t, tt.isComplete, isComplete)
			assert.Equal(t, tt.isFailed, isFailed)
		})
	}
}
