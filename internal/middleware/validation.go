package middleware

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"videostreamgo/internal/types"
)

// Validator wraps the validator library
var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateRequest validates the request body against the provided struct
func ValidateRequest(dto interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindJSON(dto); err != nil {
			var errorMessages []string
			for _, err := range err.(validator.ValidationErrors) {
				field := err.Field()
				tag := err.Tag()
				var message string

				switch tag {
				case "required":
					message = field + " is required"
				case "email":
					message = field + " must be a valid email"
				case "min":
					message = field + " must be at least " + err.Param() + " characters"
				case "max":
					message = field + " must be at most " + err.Param() + " characters"
				case "url":
					message = field + " must be a valid URL"
				case "uuid":
					message = field + " must be a valid UUID"
				case "alphanum":
					message = field + " must contain only alphanumeric characters"
				case "gte":
					message = field + " must be greater than or equal to " + err.Param()
				case "lte":
					message = field + " must be less than or equal to " + err.Param()
				default:
					message = field + " is invalid"
				}

				errorMessages = append(errorMessages, message)
			}

			c.AbortWithStatusJSON(http.StatusBadRequest, types.ErrorResponse(
				"VALIDATION_ERROR",
				"Request validation failed",
				strings.Join(errorMessages, "; "),
			))
			return
		}

		c.Set("validated_dto", dto)
		c.Next()
	}
}

// ValidateQuery validates query parameters
func ValidateQuery(dto interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindQuery(dto); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, types.ErrorResponse(
				"VALIDATION_ERROR",
				"Query parameter validation failed",
				err.Error(),
			))
			return
		}
		c.Next()
	}
}

// Custom validators

// IsValidEmail validates email format
func IsValidEmail(fl validator.FieldLevel) bool {
	email := fl.Field().String()
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// IsValidSubdomain validates subdomain format
func IsValidSubdomain(fl validator.FieldLevel) bool {
	subdomain := fl.Field().String()
	if len(subdomain) < 3 || len(subdomain) > 63 {
		return false
	}
	pattern := `^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`
	matched, _ := regexp.MatchString(pattern, subdomain)
	return matched
}

// IsValidSlug validates URL slug format
func IsValidSlug(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	if len(slug) < 3 || len(slug) > 255 {
		return false
	}
	pattern := `^[a-z0-9]+(-[a-z0-9]+)*$`
	matched, _ := regexp.MatchString(pattern, slug)
	return matched
}

// RegisterCustomValidators registers custom validators with the validator instance
func RegisterCustomValidators() {
	validate.RegisterValidation("email_format", IsValidEmail)
	validate.RegisterValidation("subdomain", IsValidSubdomain)
	validate.RegisterValidation("slug", IsValidSlug)
}

// PaginationValidator validates pagination parameters
type PaginationValidator struct {
	Page    int `form:"page" binding:"min=1"`
	PerPage int `form:"per_page" binding:"min=1,max=100"`
}

// ValidatePagination returns a middleware that validates and sets defaults for pagination
func ValidatePagination() gin.HandlerFunc {
	return func(c *gin.Context) {
		var pv PaginationValidator
		if err := c.ShouldBindQuery(&pv); err != nil {
			pv.Page = 1
			pv.PerPage = 20
		}
		if pv.Page < 1 {
			pv.Page = 1
		}
		if pv.PerPage < 1 {
			pv.PerPage = 20
		}
		if pv.PerPage > 100 {
			pv.PerPage = 100
		}

		c.Set("page", pv.Page)
		c.Set("per_page", pv.PerPage)
		c.Next()
	}
}

// GetPage retrieves the page number from context
func GetPage(c *gin.Context) int {
	if page, exists := c.Get("page"); exists {
		return page.(int)
	}
	return 1
}

// GetPerPage retrieves the per page value from context
func GetPerPage(c *gin.Context) int {
	if perPage, exists := c.Get("per_page"); exists {
		return perPage.(int)
	}
	return 20
}
