package instance

import (
	"github.com/google/uuid"

	"videostreamgo/internal/models/instance"
)

// CreateCategoryRequest represents a request to create a new category
type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Slug        string `json:"slug" binding:"required,slug"`
	Description string `json:"description" binding:"max=500"`
	ParentID    string `json:"parent_id" binding:"omitempty,uuid"`
}

// UpdateCategoryRequest represents a request to update a category
type UpdateCategoryRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=100"`
	Slug        *string `json:"slug" binding:"omitempty,slug"`
	Description *string `json:"description" binding:"omitempty,max=500"`
	ParentID    *string `json:"parent_id" binding:"omitempty,uuid"`
}

// CategoryResponse represents a category in API responses
type CategoryResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	ParentName  string     `json:"parent_name,omitempty"`
	VideoCount  int        `json:"video_count"`
	CreatedAt   string     `json:"created_at"`
}

// ToCategoryResponse converts a Category to CategoryResponse
func ToCategoryResponse(category *instance.Category, parentName string, videoCount int) CategoryResponse {
	return CategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Slug:        category.Slug,
		Description: category.Description,
		ParentID:    category.ParentID,
		ParentName:  parentName,
		VideoCount:  videoCount,
		CreatedAt:   category.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// CategoryListResponse represents a list of categories
type CategoryListResponse struct {
	Categories []CategoryResponse `json:"categories"`
}
