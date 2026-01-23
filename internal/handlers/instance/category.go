package instance

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"videostreamgo/internal/middleware"
	instancemodels "videostreamgo/internal/models/instance"
	repo "videostreamgo/internal/repository/instance"
	"videostreamgo/internal/types"
)

// CategoryHandler handles category endpoints for instance API
type CategoryHandler struct {
	categoryRepo *repo.CategoryRepository
	videoRepo    *repo.VideoRepository
}

// NewCategoryHandler creates a new CategoryHandler
func NewCategoryHandler(categoryRepo *repo.CategoryRepository, videoRepo *repo.VideoRepository) *CategoryHandler {
	return &CategoryHandler{
		categoryRepo: categoryRepo,
		videoRepo:    videoRepo,
	}
}

// ListCategories lists all categories
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	page := getIntParam(c, "page", 1)
	perPage := getIntParam(c, "per_page", 20)

	categories, total, err := h.categoryRepo.List(c.Request.Context(), (page-1)*perPage, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("LIST_ERROR", "Failed to list categories", err.Error()))
		return
	}

	result := make([]map[string]interface{}, len(categories))
	for i, category := range categories {
		videoCount, _ := h.videoRepo.GetVideoCountByCategory(c.Request.Context(), category.ID)
		result[i] = h.toCategoryResponse(&category, videoCount)
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"categories": result,
		"total":      total,
		"page":       page,
		"per_page":   perPage,
	}, ""))
}

// GetCategory returns a category by ID
func (h *CategoryHandler) GetCategory(c *gin.Context) {
	id := c.Param("id")
	categoryID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid category ID", ""))
		return
	}

	category, err := h.categoryRepo.GetByID(c.Request.Context(), categoryID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Category not found", ""))
		return
	}

	videoCount, _ := h.videoRepo.GetVideoCountByCategory(c.Request.Context(), category.ID)

	c.JSON(http.StatusOK, types.SuccessResponse(h.toCategoryResponse(category, videoCount), ""))
}

// CreateCategory creates a new category (admin only)
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required,min=1,max=100"`
		Description string `json:"description" binding:"max=500"`
		IconURL     string `json:"icon_url"`
		Color       string `json:"color"`
		ParentID    string `json:"parent_id"`
		SortOrder   int    `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	// Validate color format
	if req.Color != "" && !isValidHexColor(req.Color) {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_COLOR", "Invalid color format", ""))
		return
	}

	instanceID := middleware.GetTenantID(c)

	category := &instancemodels.Category{
		InstanceID:  instanceID,
		Name:        req.Name,
		Slug:        generateCategorySlug(req.Name),
		Description: req.Description,
		IconURL:     req.IconURL,
		Color:       req.Color,
		SortOrder:   req.SortOrder,
		IsActive:    true,
	}

	if req.ParentID != "" {
		parentID, err := uuid.Parse(req.ParentID)
		if err == nil {
			category.ParentID = &parentID
		}
	}

	if err := h.categoryRepo.Create(c.Request.Context(), category); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("CREATE_ERROR", "Failed to create category", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.SuccessResponse(h.toCategoryResponse(category, 0), "Category created successfully"))
}

// UpdateCategory updates a category (admin only)
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	categoryID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid category ID", ""))
		return
	}

	category, err := h.categoryRepo.GetByID(c.Request.Context(), categoryID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Category not found", ""))
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		IconURL     *string `json:"icon_url"`
		Color       *string `json:"color"`
		ParentID    *string `json:"parent_id"`
		SortOrder   *int    `json:"sort_order"`
		IsActive    *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	if req.Name != nil {
		category.Name = *req.Name
		category.Slug = generateCategorySlug(*req.Name)
	}
	if req.Description != nil {
		category.Description = *req.Description
	}
	if req.IconURL != nil {
		category.IconURL = *req.IconURL
	}
	if req.Color != nil {
		if !isValidHexColor(*req.Color) {
			c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_COLOR", "Invalid color format", ""))
			return
		}
		category.Color = *req.Color
	}
	if req.SortOrder != nil {
		category.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}
	if req.ParentID != nil {
		if *req.ParentID == "" {
			category.ParentID = nil
		} else {
			parentID, err := uuid.Parse(*req.ParentID)
			if err == nil {
				category.ParentID = &parentID
			}
		}
	}

	if err := h.categoryRepo.Update(c.Request.Context(), category); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to update category", err.Error()))
		return
	}

	videoCount, _ := h.videoRepo.GetVideoCountByCategory(c.Request.Context(), category.ID)
	c.JSON(http.StatusOK, types.SuccessResponse(h.toCategoryResponse(category, videoCount), "Category updated successfully"))
}

// DeleteCategory deletes a category (admin only)
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	categoryID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid category ID", ""))
		return
	}

	// Check if category has videos
	videoCount, _ := h.videoRepo.GetVideoCountByCategory(c.Request.Context(), categoryID)
	if videoCount > 0 {
		c.JSON(http.StatusConflict, types.ErrorResponse("CATEGORY_HAS_VIDEOS", "Cannot delete category with videos", ""))
		return
	}

	if err := h.categoryRepo.Delete(c.Request.Context(), categoryID); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("DELETE_ERROR", "Failed to delete category", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(nil, "Category deleted successfully"))
}

// TagHandler handles tag endpoints for instance API
type TagHandler struct {
	tagRepo *repo.TagRepository
}

// NewTagHandler creates a new TagHandler
func NewTagHandler(tagRepo *repo.TagRepository) *TagHandler {
	return &TagHandler{
		tagRepo: tagRepo,
	}
}

// ListTags lists all tags
func (h *TagHandler) ListTags(c *gin.Context) {
	page := getIntParam(c, "page", 1)
	perPage := getIntParam(c, "per_page", 20)

	tags, total, err := h.tagRepo.List(c.Request.Context(), (page-1)*perPage, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("LIST_ERROR", "Failed to list tags", err.Error()))
		return
	}

	result := make([]map[string]interface{}, len(tags))
	for i, tag := range tags {
		result[i] = h.toTagResponse(&tag)
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"tags":     result,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	}, ""))
}

// CreateTag creates a new tag (admin only)
func (h *TagHandler) CreateTag(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required,min=1,max=100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	instanceID := middleware.GetTenantID(c)

	tag := &instancemodels.Tag{
		InstanceID: instanceID,
		Name:       req.Name,
		Slug:       generateTagSlug(req.Name),
	}

	if err := h.tagRepo.Create(c.Request.Context(), tag); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("CREATE_ERROR", "Failed to create tag", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.SuccessResponse(h.toTagResponse(tag), "Tag created successfully"))
}

// DeleteTag deletes a tag (admin only)
func (h *TagHandler) DeleteTag(c *gin.Context) {
	id := c.Param("id")
	tagID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid tag ID", ""))
		return
	}

	if err := h.tagRepo.Delete(c.Request.Context(), tagID); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("DELETE_ERROR", "Failed to delete tag", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(nil, "Tag deleted successfully"))
}

// Helper functions

func generateCategorySlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	var result strings.Builder
	for _, c := range slug {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result.WriteRune(c)
		}
	}

	return result.String() + "-" + uuid.New().String()[:8]
}

func generateTagSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	var result strings.Builder
	for _, c := range slug {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result.WriteRune(c)
		}
	}

	return result.String() + "-" + uuid.New().String()[:8]
}

func isValidHexColor(color string) bool {
	if len(color) != 7 || color[0] != '#' {
		return false
	}
	for _, c := range color[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// toCategoryResponse converts a Category to response format
func (h *CategoryHandler) toCategoryResponse(category *instancemodels.Category, videoCount int64) map[string]interface{} {
	response := map[string]interface{}{
		"id":          category.ID,
		"name":        category.Name,
		"slug":        category.Slug,
		"description": category.Description,
		"icon_url":    category.IconURL,
		"color":       category.Color,
		"sort_order":  category.SortOrder,
		"is_active":   category.IsActive,
		"video_count": videoCount,
		"created_at":  category.CreatedAt.Format(time.RFC3339),
		"updated_at":  category.UpdatedAt.Format(time.RFC3339),
	}

	if category.ParentID != nil {
		response["parent_id"] = *category.ParentID
	}

	return response
}

// toTagResponse converts a Tag to response format
func (h *TagHandler) toTagResponse(tag *instancemodels.Tag) map[string]interface{} {
	return map[string]interface{}{
		"id":          tag.ID,
		"name":        tag.Name,
		"slug":        tag.Slug,
		"usage_count": tag.UsageCount,
		"created_at":  tag.CreatedAt.Format(time.RFC3339),
	}
}
