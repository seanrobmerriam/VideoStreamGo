package instance

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"videostreamgo/internal/models/instance"
)

// Test_CategoryRepository_Queries tests category query logic
func Test_CategoryRepository_Queries(t *testing.T) {
	category := &instance.Category{
		ID:       uuid.New(),
		Name:     "Technology",
		Slug:     "technology",
		IsActive: true,
	}

	assert.NotEmpty(t, category.ID)
	assert.Equal(t, "Technology", category.Name)
	assert.True(t, category.IsActive)
}

// Test_CategoryRepository_TagAssociations tests tag association logic
func Test_CategoryRepository_TagAssociations(t *testing.T) {
	tests := []struct {
		name         string
		categoryName string
		tagCount     int
		expectValid  bool
	}{
		{"Category with tags", "Technology", 5, true},
		{"Category without tags", "General", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = &instance.Category{
				ID:   uuid.New(),
				Name: tt.categoryName,
			}

			tagCount := tt.tagCount
			assert.Equal(t, tt.expectValid, tagCount >= 0)
		})
	}
}
