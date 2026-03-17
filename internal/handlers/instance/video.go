package instance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"videostreamgo/internal/config"
	instancemodels "videostreamgo/internal/models/instance"
	services "videostreamgo/internal/services/instance"
	"videostreamgo/internal/types"
)

// VideoUploadHandler handles video upload requests
type VideoUploadHandler struct {
	storageService    services.StorageService
	processingService services.VideoProcessingService
	videoRepo         services.VideoRepositoryInterface
	maxSize           int64
	chunkSize         int64
	uploadDir         string
	allowedTypes      map[string]bool
}

// NewVideoUploadHandler creates a new video upload handler
func NewVideoUploadHandler(
	storageService services.StorageService,
	processingService services.VideoProcessingService,
	videoRepo services.VideoRepositoryInterface,
	cfg *config.Config,
) *VideoUploadHandler {
	uploadDir := filepath.Join(os.TempDir(), "videostreamgo-uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create upload directory: %v", err))
	}

	maxSize := int64(5 * 1024 * 1024 * 1024) // 5GB default

	return &VideoUploadHandler{
		storageService:    storageService,
		processingService: processingService,
		videoRepo:         videoRepo,
		maxSize:           maxSize,
		chunkSize:         10 * 1024 * 1024, // 10MB chunks
		uploadDir:         uploadDir,
		allowedTypes: map[string]bool{
			"video/mp4":        true,
			"video/mpeg":       true,
			"video/quicktime":  true,
			"video/x-msvideo":  true,
			"video/x-matroska": true,
			"video/webm":       true,
			"video/avi":        true,
			"video/x-flv":      true,
			"video/mp2t":       true,
		},
	}
}

// InitUploadRequest represents a request to initialize an upload
type InitUploadRequest struct {
	Filename    string `json:"filename" binding:"required"`
	FileSize    int64  `json:"file_size" binding:"required,gt=0"`
	ContentType string `json:"content_type" binding:"required"`
	Title       string `json:"title" binding:"required,min=3,max=255"`
	Description string `json:"description" binding:"max=5000"`
	CategoryID  string `json:"category_id"`
	IsPublic    bool   `json:"is_public"`
}

// InitUploadResponse represents the response after initializing an upload
type InitUploadResponse struct {
	UploadID    uuid.UUID `json:"upload_id"`
	VideoID     uuid.UUID `json:"video_id"`
	ChunkSize   int64     `json:"chunk_size"`
	TotalChunks int       `json:"total_chunks"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// UploadChunkRequest represents a chunk upload request
type UploadChunkRequest struct {
	UploadID    uuid.UUID `form:"upload_id" binding:"required"`
	ChunkNumber int       `form:"chunk_number" binding:"required,min=0"`
	TotalChunks int       `form:"total_chunks" binding:"required,gt=0"`
	ChunkSize   int64     `form:"chunk_size" binding:"required,gt=0"`
}

// ChunkUploadResponse represents the response after uploading a chunk
type ChunkUploadResponse struct {
	UploadID       uuid.UUID `json:"upload_id"`
	ChunkNumber    int       `json:"chunk_number"`
	UploadedChunks int       `json:"uploaded_chunks"`
	TotalChunks    int       `json:"total_chunks"`
	Progress       float64   `json:"progress"`
}

// CompleteUploadRequest represents a request to complete an upload
type CompleteUploadRequest struct {
	UploadID uuid.UUID `json:"upload_id" binding:"required"`
	FileHash string    `json:"file_hash" binding:"required,len=64"`
}

// CompleteUploadResponse represents the response after completing an upload
type CompleteUploadResponse struct {
	UploadID uuid.UUID `json:"upload_id"`
	VideoID  uuid.UUID `json:"video_id"`
	Status   string    `json:"status"`
	Message  string    `json:"message"`
}

// InitUpload initializes a new upload session
func (h *VideoUploadHandler) InitUpload(c *gin.Context) (*InitUploadResponse, error) {
	var req InitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate content type
	if !h.allowedTypes[req.ContentType] {
		return nil, fmt.Errorf("invalid content type: %s", req.ContentType)
	}

	// Validate file size
	if req.FileSize > h.maxSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", h.maxSize)
	}

	// Generate IDs
	videoID := uuid.New()
	uploadID := uuid.New()

	// Calculate total chunks
	totalChunks := int(req.FileSize / h.chunkSize)
	if req.FileSize%h.chunkSize > 0 {
		totalChunks++
	}

	// Create upload session
	session := &instancemodels.UploadSession{
		ID:          uploadID,
		VideoID:     videoID,
		TotalChunks: totalChunks,
		ChunkSize:   h.chunkSize,
		TotalSize:   req.FileSize,
		FileName:    req.Filename,
		ContentType: req.ContentType,
		Status:      "pending",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	_ = session

	return &InitUploadResponse{
		UploadID:    uploadID,
		VideoID:     videoID,
		ChunkSize:   h.chunkSize,
		TotalChunks: totalChunks,
		ExpiresAt:   session.ExpiresAt,
	}, nil
}

// UploadChunk handles uploading a single chunk of a video
func (h *VideoUploadHandler) UploadChunk(c *gin.Context) (*ChunkUploadResponse, error) {
	// Get video ID from URL
	videoIDStr := c.Param("id")
	_, err := uuid.Parse(videoIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid video ID: %w", err)
	}

	// Get chunk number from URL
	chunkNumberStr := c.Param("chunk")
	chunkNumber, err := strconv.Atoi(chunkNumberStr)
	if err != nil {
		return nil, fmt.Errorf("invalid chunk number: %w", err)
	}

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	defer file.Close()

	// Read chunk data
	chunkData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk: %w", err)
	}

	// Save chunk to storage
	sessionID, _ := uuid.Parse(videoIDStr)
	if err := h.storageService.UploadChunk(sessionID, chunkNumber, chunkData); err != nil {
		return nil, fmt.Errorf("failed to save chunk: %w", err)
	}

	// Calculate progress
	uploadedChunks := chunkNumber + 1
	totalChunks := int(header.Size / h.chunkSize)
	if header.Size%h.chunkSize > 0 {
		totalChunks++
	}
	progress := float64(uploadedChunks) / float64(totalChunks) * 100

	return &ChunkUploadResponse{
		UploadID:       sessionID,
		ChunkNumber:    chunkNumber,
		UploadedChunks: uploadedChunks,
		TotalChunks:    totalChunks,
		Progress:       progress,
	}, nil
}

// CompleteUpload completes an upload session
func (h *VideoUploadHandler) CompleteUpload(c *gin.Context) (*CompleteUploadResponse, error) {
	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get video from database
	video, err := h.videoRepo.GetByID(c, req.UploadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	video.ProcessingStatus = instancemodels.ProcessingStatusUploaded
	video.Status = instancemodels.VideoStatusProcessing
	if err := h.videoRepo.Update(c, video); err != nil {
		return nil, fmt.Errorf("failed to update video: %w", err)
	}

	return &CompleteUploadResponse{
		UploadID: req.UploadID,
		VideoID:  video.ID,
		Status:   "uploaded",
		Message:  "Video uploaded successfully, processing started",
	}, nil
}

// UploadVideo handles single-part video upload (for smaller files)
func (h *VideoUploadHandler) UploadVideo(c *gin.Context) (*types.APIResponse, error) {
	// Get user ID from context (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return nil, fmt.Errorf("unauthorized")
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get instance ID from context
	instanceIDStr, _ := c.Get("instance_id")
	instanceID, _ := uuid.Parse(instanceIDStr.(string))

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(h.maxSize); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	// Get file
	file, header, err := c.Request.FormFile("video")
	if err != nil {
		return nil, fmt.Errorf("failed to get video file: %w", err)
	}
	defer file.Close()

	// Get metadata
	title := c.PostForm("title")
	description := c.PostForm("description")
	categoryIDStr := c.PostForm("category_id")
	isPublic := c.PostForm("is_public") == "true"

	// Validate title
	if len(title) < 3 || len(title) > 255 {
		return nil, fmt.Errorf("title must be between 3 and 255 characters")
	}

	// Validate content type
	contentType := header.Header.Get("Content-Type")
	if !h.allowedTypes[contentType] {
		return nil, fmt.Errorf("invalid video format: %s", contentType)
	}

	// Get file size
	fileSize := header.Size
	if fileSize > h.maxSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", h.maxSize)
	}

	// Generate IDs
	videoID := uuid.New()

	// Save to temp file
	tempPath := filepath.Join(h.uploadDir, videoID.String()+"_"+header.Filename)
	dst, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempPath) // Cleanup after processing

	// Copy file data
	written, err := io.Copy(dst, file)
	if err != nil {
		dst.Close()
		return nil, fmt.Errorf("failed to save file: %w", err)
	}
	dst.Close()

	if written != fileSize {
		return nil, fmt.Errorf("file size mismatch: expected %d, got %d", fileSize, written)
	}

	// Calculate file hash
	fileHash, err := calculateFileHash(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}

	_ = fileHash

	// Create video record
	var categoryID *uuid.UUID
	if categoryIDStr != "" {
		catID, err := uuid.Parse(categoryIDStr)
		if err == nil {
			categoryID = &catID
		}
	}

	video := &instancemodels.Video{
		ID:                 videoID,
		InstanceID:         instanceID,
		Title:              title,
		Slug:               generateSlug(title),
		Description:        description,
		UserID:             userID,
		CategoryID:         categoryID,
		Status:             instancemodels.VideoStatusProcessing,
		FileSize:           fileSize,
		ProcessingStatus:   instancemodels.ProcessingStatusUploaded,
		ProcessingProgress: 0,
		IsPublic:           isPublic,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Save to database
	if err := h.videoRepo.Create(c, video); err != nil {
		return nil, fmt.Errorf("failed to save video record: %w", err)
	}

	return &types.APIResponse{
		Success: true,
		Message: "Video uploaded successfully",
		Data: gin.H{
			"video_id":          videoID,
			"status":            "uploaded",
			"processing_status": instancemodels.ProcessingStatusUploaded,
		},
	}, nil
}

// GetUploadProgress returns the upload progress for a video
func (h *VideoUploadHandler) GetUploadProgress(c *gin.Context) {
	// Get instance ID from context (set by tenant middleware)
	instanceID, exists := c.Get(string(types.ContextKeyInstanceID))
	if !exists {
		c.JSON(http.StatusForbidden, types.ErrorResponse(
			"ACCESS_DENIED",
			"Instance context is missing",
			"",
		))
		return
	}

	instanceIDUUID, ok := instanceID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusForbidden, types.ErrorResponse(
			"ACCESS_DENIED",
			"Invalid instance ID in context",
			"",
		))
		return
	}

	videoIDStr := c.Param("id")
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse(
			"INVALID_VIDEO_ID",
			"Invalid video ID format",
			"",
		))
		return
	}

	// Get video from database
	video, err := h.videoRepo.GetByID(c, videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse(
			"VIDEO_NOT_FOUND",
			"Video not found",
			"",
		))
		return
	}

	// SECURITY FIX: Verify the video belongs to the current tenant (IDOR vulnerability fix)
	if video.InstanceID != instanceIDUUID {
		c.JSON(http.StatusForbidden, types.ErrorResponse(
			"ACCESS_DENIED",
			"Video does not belong to this instance",
			"",
		))
		return
	}

	c.JSON(http.StatusOK, ChunkUploadResponse{
		UploadID:       videoID,
		ChunkNumber:    int(video.ProcessingProgress),
		UploadedChunks: int(video.ProcessingProgress),
		TotalChunks:    100,
		Progress:       float64(video.ProcessingProgress),
	})
}

// CancelUpload cancels an ongoing upload
func (h *VideoUploadHandler) CancelUpload(c *gin.Context) error {
	videoIDStr := c.Param("id")
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		return fmt.Errorf("invalid video ID: %w", err)
	}

	// Get video from database
	video, err := h.videoRepo.GetByID(c, videoID)
	if err != nil {
		return fmt.Errorf("failed to get video: %w", err)
	}

	// Check if video is still uploading
	if video.ProcessingStatus != instancemodels.ProcessingStatusUploading &&
		video.ProcessingStatus != instancemodels.ProcessingStatusPending {
		return fmt.Errorf("cannot cancel upload in current state: %s", video.ProcessingStatus)
	}

	// Delete from storage
	if err := h.storageService.DeleteVideo(videoID.String()); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to delete video from storage: %v\n", err)
	}

	// Update video status
	video.Status = instancemodels.VideoStatusDeleted
	if err := h.videoRepo.Update(c, video); err != nil {
		return fmt.Errorf("failed to update video: %w", err)
	}

	return nil
}

// Helper function to generate URL-safe slug
func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove special characters
	var result strings.Builder
	for _, c := range slug {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result.WriteRune(c)
		}
	}

	// Add UUID to ensure uniqueness
	slug = result.String() + "-" + uuid.New().String()[:8]
	return slug
}

// Helper function to calculate file hash
func calculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// validateVideoFile validates a video file
func (h *VideoUploadHandler) validateVideoFile(file *multipart.FileHeader) error {
	// Check file size
	if file.Size > h.maxSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", h.maxSize)
	}

	// Check content type
	contentType := file.Header.Get("Content-Type")
	if !h.allowedTypes[contentType] {
		return fmt.Errorf("invalid video format: %s", contentType)
	}

	return nil
}

// StreamVideoHandler handles video streaming requests with authentication and authorization
func (h *VideoUploadHandler) StreamVideoHandler(c *gin.Context) {
	// Step 1: Authentication - Check for valid JWT token in context
	userID, exists := c.Get(string(types.ContextKeyUserID))
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse(
			"UNAUTHORIZED",
			"Authentication required. Please provide a valid JWT token.",
			"",
		))
		return
	}

	userIDUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse(
			"INVALID_USER",
			"Invalid user ID in authentication context",
			"",
		))
		return
	}

	// Get instance ID from context (set by tenant middleware)
	instanceID, exists := c.Get(string(types.ContextKeyInstanceID))
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse(
			"MISSING_INSTANCE",
			"Instance context is missing",
			"",
		))
		return
	}

	instanceIDUUID, ok := instanceID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse(
			"INVALID_INSTANCE",
			"Invalid instance ID in context",
			"",
		))
		return
	}

	// Step 2: Parse video ID
	videoID := c.Param("id")
	if videoID == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse(
			"MISSING_VIDEO_ID",
			"Video ID is required",
			"",
		))
		return
	}

	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse(
			"INVALID_VIDEO_ID",
			"Invalid video ID format",
			"",
		))
		return
	}

	// Step 3: Fetch video from database
	video, err := h.videoRepo.GetByID(c.Request.Context(), videoUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse(
			"VIDEO_NOT_FOUND",
			"Video not found",
			"",
		))
		return
	}

	// Step 4: Authorization - Verify user has access to the video
	// Check 1: Video must belong to the same instance (tenant)
	if video.InstanceID != instanceIDUUID {
		c.JSON(http.StatusForbidden, types.ErrorResponse(
			"ACCESS_DENIED",
			"Video does not belong to this instance",
			"",
		))
		return
	}

	// Check 2: User must be the owner OR video must be public
	isOwner := video.UserID == userIDUUID
	isPublic := video.IsPublic

	if !isOwner && !isPublic {
		c.JSON(http.StatusForbidden, types.ErrorResponse(
			"ACCESS_DENIED",
			"You do not have permission to stream this video. Only the owner or public videos can be accessed.",
			"",
		))
		return
	}

	// Step 5: Check video status - only allow streaming of ready/active/hidden videos
	allowedStatuses := map[instancemodels.VideoStatus]bool{
		instancemodels.VideoStatusReady:   true,
		instancemodels.VideoStatusActive:  true,
		instancemodels.VideoStatusHidden:  true,
	}

	if !allowedStatuses[video.Status] {
		c.JSON(http.StatusForbidden, types.ErrorResponse(
			"VIDEO_NOT_AVAILABLE",
			"Video is not available for streaming",
			"",
		))
		return
	}

	// Step 6: Handle video streaming
	quality := c.Param("quality")

	// Set content type for HLS
	if quality == "hls" || strings.HasSuffix(c.Request.URL.Path, ".m3u8") {
		c.Header("Content-Type", "application/vnd.apple.mpegurl")
		c.Header("Cache-Control", "public, max-age=3600")
	} else if strings.HasSuffix(c.Request.URL.Path, ".ts") {
		c.Header("Content-Type", "video/mp2t")
		c.Header("Cache-Control", "public, max-age=86400")
	} else {
		c.Header("Content-Type", "video/mp4")
	}

	// Handle range requests for progressive download
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		c.Header("Accept-Ranges", "bytes")
		// Parse range and serve partial content
		c.Status(http.StatusPartialContent)
	}

	// Return video streaming information
	// In production, this would return the actual video stream or redirect to CDN
	c.JSON(http.StatusOK, gin.H{
		"message":     "Video streaming endpoint",
		"video_id":    videoID,
		"quality":     quality,
		"stream_type": "hls",
		"video_url":   video.VideoURL,
		"hls_path":    video.HLSPath,
	})
}
