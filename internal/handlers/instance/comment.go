package instance

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"videostreamgo/internal/middleware"
	instancemodels "videostreamgo/internal/models/instance"
	instanceRepo "videostreamgo/internal/repository/instance"
	"videostreamgo/internal/types"
)

// CommentHandler handles comment endpoints for instance API
type CommentHandler struct {
	commentRepo *instanceRepo.CommentRepository
	ratingRepo  *instanceRepo.RatingRepository
	videoRepo   *instanceRepo.VideoRepository
	userRepo    *instanceRepo.UserRepository
}

// NewCommentHandler creates a new CommentHandler
func NewCommentHandler(commentRepo *instanceRepo.CommentRepository, ratingRepo *instanceRepo.RatingRepository, videoRepo *instanceRepo.VideoRepository, userRepo *instanceRepo.UserRepository) *CommentHandler {
	return &CommentHandler{
		commentRepo: commentRepo,
		ratingRepo:  ratingRepo,
		videoRepo:   videoRepo,
		userRepo:    userRepo,
	}
}

// ListComments lists comments for a video
func (h *CommentHandler) ListComments(c *gin.Context) {
	videoID := c.Param("id")
	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid video ID", ""))
		return
	}

	page := getIntParam(c, "page", 1)
	perPage := getIntParam(c, "per_page", 20)

	comments, total, err := h.commentRepo.GetTopLevelByVideoID(c.Request.Context(), videoUUID, (page-1)*perPage, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("LIST_ERROR", "Failed to list comments", err.Error()))
		return
	}

	result := make([]map[string]interface{}, len(comments))
	for i, comment := range comments {
		user, _ := h.userRepo.GetByID(c.Request.Context(), comment.UserID)
		replies, _ := h.commentRepo.GetReplies(c.Request.Context(), comment.ID)

		replyData := make([]map[string]interface{}, len(replies))
		for j, reply := range replies {
			replyUser, _ := h.userRepo.GetByID(c.Request.Context(), reply.UserID)
			replyData[j] = h.toCommentResponse(&reply, replyUser)
		}

		result[i] = h.toCommentResponse(&comment, user)
		result[i]["replies"] = replyData
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"comments": result,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	}, ""))
}

// CreateComment creates a new comment
func (h *CommentHandler) CreateComment(c *gin.Context) {
	videoID := c.Param("id")
	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid video ID", ""))
		return
	}

	userID, exists := c.Get(string(types.ContextKeyUserID))
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("NOT_AUTHENTICATED", "User not authenticated", ""))
		return
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("TYPE_ERROR", "Invalid user ID type", ""))
		return
	}

	var req struct {
		Content  string `json:"content" binding:"required,min=1,max=5000"`
		ParentID string `json:"parent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	// Verify video exists
	_, err = h.videoRepo.GetByID(c.Request.Context(), videoUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Video not found", ""))
		return
	}

	instanceID := middleware.GetTenantID(c)

	comment := &instancemodels.Comment{
		InstanceID: instanceID,
		VideoID:    videoUUID,
		UserID:     id,
		Content:    req.Content,
		IsEdited:   false,
		IsDeleted:  false,
		LikeCount:  0,
	}

	if req.ParentID != "" {
		parentUUID, err := uuid.Parse(req.ParentID)
		if err == nil {
			comment.ParentID = &parentUUID
		}
	}

	if err := h.commentRepo.Create(c.Request.Context(), comment); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("CREATE_ERROR", "Failed to create comment", err.Error()))
		return
	}

	user, _ := h.userRepo.GetByID(c.Request.Context(), id)

	c.JSON(http.StatusCreated, types.SuccessResponse(h.toCommentResponse(comment, user), "Comment created successfully"))
}

// UpdateComment updates a comment
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	id := c.Param("id")
	commentUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid comment ID", ""))
		return
	}

	userID, exists := c.Get(string(types.ContextKeyUserID))
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("NOT_AUTHENTICATED", "User not authenticated", ""))
		return
	}

	currentUserID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("TYPE_ERROR", "Invalid user ID type", ""))
		return
	}

	comment, err := h.commentRepo.GetByID(c.Request.Context(), commentUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Comment not found", ""))
		return
	}

	// Only the author can edit their comment
	if comment.UserID != currentUserID {
		c.JSON(http.StatusForbidden, types.ErrorResponse("FORBIDDEN", "You can only edit your own comments", ""))
		return
	}

	var req struct {
		Content string `json:"content" binding:"required,min=1,max=5000"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	comment.Content = req.Content
	comment.IsEdited = true

	if err := h.commentRepo.Update(c.Request.Context(), comment); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to update comment", err.Error()))
		return
	}

	user, _ := h.userRepo.GetByID(c.Request.Context(), comment.UserID)
	c.JSON(http.StatusOK, types.SuccessResponse(h.toCommentResponse(comment, user), "Comment updated successfully"))
}

// DeleteComment deletes a comment
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	id := c.Param("id")
	commentUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid comment ID", ""))
		return
	}

	userID, exists := c.Get(string(types.ContextKeyUserID))
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("NOT_AUTHENTICATED", "User not authenticated", ""))
		return
	}

	currentUserID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("TYPE_ERROR", "Invalid user ID type", ""))
		return
	}

	comment, err := h.commentRepo.GetByID(c.Request.Context(), commentUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Comment not found", ""))
		return
	}

	// Only the author can delete their comment
	if comment.UserID != currentUserID {
		c.JSON(http.StatusForbidden, types.ErrorResponse("FORBIDDEN", "You can only delete your own comments", ""))
		return
	}

	// Soft delete
	comment.IsDeleted = true
	comment.Content = "[deleted]"
	if err := h.commentRepo.Update(c.Request.Context(), comment); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("DELETE_ERROR", "Failed to delete comment", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(nil, "Comment deleted successfully"))
}

// RateVideo handles rating a video (like/dislike)
func (h *CommentHandler) RateVideo(c *gin.Context) {
	videoID := c.Param("id")
	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid video ID", ""))
		return
	}

	userID, exists := c.Get(string(types.ContextKeyUserID))
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("NOT_AUTHENTICATED", "User not authenticated", ""))
		return
	}

	currentUserID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("TYPE_ERROR", "Invalid user ID type", ""))
		return
	}

	var req struct {
		Rating int8 `json:"rating" binding:"required,oneof=-1 1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	// Check if user has already rated
	existingRating, err := h.ratingRepo.GetByVideoAndUser(c.Request.Context(), videoUUID, currentUserID)
	if err == nil && existingRating != nil {
		// User has already rated - update or remove
		if existingRating.Rating == req.Rating {
			// Same rating - remove it (toggle off)
			h.ratingRepo.Delete(c.Request.Context(), existingRating.ID)
			if req.Rating == 1 {
				h.videoRepo.DecrementLikeCount(c.Request.Context(), videoUUID)
			}
			c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
				"rating": 0,
			}, "Rating removed"))
			return
		}

		// Different rating - update
		if existingRating.Rating == 1 {
			h.videoRepo.DecrementLikeCount(c.Request.Context(), videoUUID)
		}

		existingRating.Rating = req.Rating
		h.ratingRepo.Update(c.Request.Context(), existingRating)

		if req.Rating == 1 {
			h.videoRepo.IncrementLikeCount(c.Request.Context(), videoUUID)
		}

		c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
			"rating": req.Rating,
		}, "Rating updated"))
		return
	}

	// Create new rating
	rating := &instancemodels.Rating{
		InstanceID: middleware.GetTenantID(c),
		VideoID:    videoUUID,
		UserID:     currentUserID,
		Rating:     req.Rating,
	}

	if err := h.ratingRepo.Create(c.Request.Context(), rating); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("CREATE_ERROR", "Failed to rate video", err.Error()))
		return
	}

	if req.Rating == 1 {
		h.videoRepo.IncrementLikeCount(c.Request.Context(), videoUUID)
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"rating": req.Rating,
	}, "Video rated successfully"))
}

// GetVideoRating returns the rating stats for a video
func (h *CommentHandler) GetVideoRating(c *gin.Context) {
	videoID := c.Param("id")
	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid video ID", ""))
		return
	}

	likes, dislikes, err := h.ratingRepo.GetVideoRatingStats(c.Request.Context(), videoUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("LIST_ERROR", "Failed to get rating stats", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"likes":    likes,
		"dislikes": dislikes,
	}, ""))
}

// toCommentResponse converts a Comment to response format
func (h *CommentHandler) toCommentResponse(comment *instancemodels.Comment, user *instancemodels.User) map[string]interface{} {
	response := map[string]interface{}{
		"id":         comment.ID,
		"content":    comment.Content,
		"is_edited":  comment.IsEdited,
		"like_count": comment.LikeCount,
		"created_at": comment.CreatedAt.Format(time.RFC3339),
		"updated_at": comment.UpdatedAt.Format(time.RFC3339),
	}

	if user != nil {
		response["user"] = map[string]interface{}{
			"id":           user.ID,
			"username":     user.Username,
			"display_name": user.DisplayName,
			"avatar_url":   user.AvatarURL,
		}
	}

	if comment.ParentID != nil {
		response["parent_id"] = *comment.ParentID
	}

	return response
}
