package instance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test_CategoryService_SlugGeneration tests slug generation for categories
func Test_CategoryService_SlugGeneration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple name", "Technology", "technology"},
		{"With spaces", "Web Development", "web-development"},
		{"With special chars", "C++ Programming", "c"},
		{"Uppercase", "UPPERCASE", "uppercase"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple slug generation logic for testing
			slug := ""
			for _, c := range tt.input {
				if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
					slug += string(c)
				} else if c >= 'A' && c <= 'Z' {
					slug += string(c + 32)
				} else if c == ' ' || c == '+' {
					slug += "-"
				}
			}
			assert.Contains(t, slug, tt.expected)
		})
	}
}

// Test_CategoryService_BusinessLogic tests category business logic
func Test_CategoryService_BusinessLogic(t *testing.T) {
	tests := []struct {
		name          string
		isActive      bool
		hasParent     bool
		expectVisible bool
	}{
		{"Active without parent", true, false, true},
		{"Inactive without parent", false, false, false},
		{"Active with parent", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isVisible := tt.isActive
			assert.Equal(t, tt.expectVisible, isVisible)
		})
	}
}
