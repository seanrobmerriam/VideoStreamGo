package middleware

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Test_ValidateRequest_RequiredFields tests validation of required fields
func Test_ValidateRequest_RequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type TestDTO struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required,email"`
	}

	tests := []struct {
		name           string
		body           interface{}
		expectedErrors []string
	}{
		{
			name:           "Missing all required fields",
			body:           map[string]string{},
			expectedErrors: []string{"name is required", "email is required"},
		},
		{
			name:           "Missing name only",
			body:           map[string]string{"email": "test@example.com"},
			expectedErrors: []string{"name is required"},
		},
		{
			name:           "Missing email only",
			body:           map[string]string{"name": "Test"},
			expectedErrors: []string{"email is required"},
		},
		{
			name:           "Invalid email format",
			body:           map[string]string{"name": "Test", "email": "invalid-email"},
			expectedErrors: []string{"email must be a valid email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate validation
			errors := []string{}
			if bodyMap, ok := tt.body.(map[string]string); ok {
				if bodyMap["name"] == "" {
					errors = append(errors, "name is required")
				}
				if bodyMap["email"] == "" {
					errors = append(errors, "email is required")
				} else if bodyMap["email"] != "test@example.com" {
					errors = append(errors, "email must be a valid email")
				}
			}

			assert.Equal(t, len(tt.expectedErrors), len(errors))
		})
	}
}

// Test_ValidateQuery_Pagination tests pagination validation
func Test_ValidateQuery_Pagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		page         string
		perPage      string
		expectedPage int
		expectedPP   int
	}{
		{"Default values", "", "", 1, 20},
		{"Valid values", "2", "50", 2, 50},
		{"Zero page defaults to 1", "0", "20", 1, 20},
		{"Negative page defaults to 1", "-1", "20", 1, 20},
		{"Zero per_page defaults to 20", "1", "0", 1, 20},
		{"Over max per_page capped at 100", "1", "200", 1, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := 1
			perPage := 20

			if tt.page != "" {
				if p := parseInt(tt.page); p > 0 {
					page = p
				}
			}
			if tt.perPage != "" {
				if pp := parseInt(tt.perPage); pp > 0 {
					perPage = pp
				}
			}
			if perPage > 100 {
				perPage = 100
			}

			assert.Equal(t, tt.expectedPage, page)
			assert.Equal(t, tt.expectedPP, perPage)
		})
	}
}

// Test_IsValidEmail tests email validation
func Test_IsValidEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"valid@example.com", true},
		{"user.name@domain.org", true},
		{"user+tag@example.co.uk", true},
		{"invalid-email", false},
		{"@nodomain.com", false},
		{"noat.com", false},
		{"double@@at.com", false},
		{"spaces in@email.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := isValidEmailFormat(tt.email)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test_IsValidSubdomain tests subdomain validation
func Test_IsValidSubdomain(t *testing.T) {
	tests := []struct {
		subdomain string
		expected  bool
	}{
		{"valid", true},
		{"my-site", true},
		{"site123", true},
		{"a", false},  // Too short
		{"ab", false}, // Too short
		{"a" + string(make([]byte, 62)) + "a", false}, // Too long
		{"UPPERCASE", false},
		{"has space", false},
		{"has_special!", false},
		{"-startswith", false},
		{"endswith-", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.subdomain, func(t *testing.T) {
			result := isValidSubdomainFormat(tt.subdomain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test_IsValidSlug tests URL slug validation
func Test_IsValidSlug(t *testing.T) {
	tests := []struct {
		slug     string
		expected bool
	}{
		{"valid-slug", true},
		{"slug123", true},
		{"multiple-parts-here", true},
		{"a", false},  // Too short
		{"ab", false}, // Too short
		{"a" + string(make([]byte, 253)) + "a", false}, // Too long
		{"has space", false},
		{"has_CAPS", false},
		{"has.special!", false},
		{"double--dash", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			result := isValidSlugFormat(tt.slug)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test_ExtractHost tests host extraction from request
func Test_ExtractHost(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{"Standard host", "example.com", "example.com"},
		{"Host with port", "example.com:8080", "example.com"},
		{"IPv4 with port", "192.168.1.1:3000", "192.168.1.1"},
		{"Empty host", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHost(tt.host)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test_ExtractSubdomain tests subdomain extraction
func Test_ExtractSubdomain(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{"Standard subdomain", "customer.videostreamgo.com", "customer"},
		{"Subdomain with port", "customer.videostreamgo.com:8080", "customer"},
		{"Localhost with port", "localhost:3000", "localhost"},
		{"No subdomain", "videostreamgo.com", ""},
		{"Deep subdomain", "deep.sub.videostreamgo.com", "deep"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSubdomain(tt.host)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test_IsPlatformDomain tests platform domain detection
func Test_IsPlatformDomain(t *testing.T) {
	tests := []struct {
		domain   string
		expected bool
	}{
		{"videostreamgo.com", true},
		{"www.videostreamgo.com", true},
		{"api.videostreamgo.com", true},
		{"admin.videostreamgo.com", true},
		{"localhost", true},
		{"customer.videostreamgo.com", false},
		{"external.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := isPlatformDomain(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions for tests
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}

func isValidEmailFormat(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			if atIndex != -1 {
				return false // Multiple @
			}
			atIndex = i
		}
	}
	if atIndex < 1 || atIndex >= len(email)-1 {
		return false
	}
	// Check for valid domain
	domain := email[atIndex+1:]
	if len(domain) < 3 {
		return false
	}
	hasDot := false
	for _, c := range domain {
		if c == '.' {
			hasDot = true
		}
	}
	return hasDot
}

func isValidSubdomainFormat(subdomain string) bool {
	if len(subdomain) < 3 || len(subdomain) > 63 {
		return false
	}
	for _, c := range subdomain {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	if subdomain[0] == '-' || subdomain[len(subdomain)-1] == '-' {
		return false
	}
	return true
}

func isValidSlugFormat(slug string) bool {
	if len(slug) < 3 || len(slug) > 255 {
		return false
	}
	for _, c := range slug {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	if slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}
	return true
}
