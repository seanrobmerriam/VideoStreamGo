package middleware

import (
	"bytes"
	"io"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"videostreamgo/internal/types"
)

// RequestLogger logs all incoming requests and their responses
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID
		requestID := uuid.New().String()
		c.Set(string(types.ContextKeyRequestID), requestID)
		c.Header("X-Request-ID", requestID)

		// Log request
		startTime := time.Now()

		// Read request body for logging (if present)
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Log request details
		log.Printf("[%s] %s %s | Status: %d | Duration: %v | Client: %s | User-Agent: %s",
			requestID,
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
			c.ClientIP(),
			c.Request.UserAgent(),
		)

		// Log slow requests
		if duration > 5*time.Second {
			log.Printf("[%s] SLOW REQUEST: %s %s took %v", requestID, c.Request.Method, c.Request.URL.Path, duration)
		}
	}
}

// RecoveryLogger recovers from panics and logs the error
func RecoveryLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := uuid.New().String()
				c.Header("X-Request-ID", requestID)

				log.Printf("[%s] PANIC RECOVERY: %v", requestID, err)

				c.AbortWithStatusJSON(500, types.ErrorResponse(
					"INTERNAL_ERROR",
					"An unexpected error occurred",
					"",
				))
			}
		}()
		c.Next()
	}
}

// isOriginAllowed checks if the given origin is in the list of allowed origins
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range allowedOrigins {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}

// CORS adds CORS headers to responses
// allowedOrigins parameter should be a list of permitted origins (e.g., from config)
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Only allow configured origins (not wildcard)
		if isOriginAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			// If origin is not allowed, still return 204 but without Allow-Credentials
			if !isOriginAllowed(origin, allowedOrigins) && origin != "" {
				log.Printf("[CORS] Rejected origin: %s", origin)
			}
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// SecurityHeaders adds security headers to responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}
