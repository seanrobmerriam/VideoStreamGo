package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"videostreamgo/internal/types"
)

// Context key for correlation ID
type contextKey string

const (
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey contextKey = "correlation_id"
	// TraceIDKey is the context key for distributed tracing ID
	TraceIDKey contextKey = "trace_id"
)

// CorrelationMiddleware generates and propagates correlation IDs for request tracing
func CorrelationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get existing correlation ID from header
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = c.GetHeader("X-Request-ID")
		}
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Set correlation ID in context
		ctx := context.WithValue(c.Request.Context(), CorrelationIDKey, correlationID)
		c.Request = c.Request.WithContext(ctx)

		// Set correlation ID in gin context
		c.Set(string(CorrelationIDKey), correlationID)

		// Add to response headers
		c.Header("X-Correlation-ID", correlationID)

		// Process request
		c.Next()
	}
}

// GetCorrelationID extracts correlation ID from context or gin context
func GetCorrelationID(c *gin.Context) string {
	// Try gin context first
	if id, exists := c.Get(string(CorrelationIDKey)); exists {
		if correlationID, ok := id.(string); ok {
			return correlationID
		}
	}

	// Try request context
	if id := c.Request.Context().Value(CorrelationIDKey); id != nil {
		if correlationID, ok := id.(string); ok {
			return correlationID
		}
	}

	return ""
}

// StructuredLogger provides JSON-structured logging with correlation IDs
type StructuredLogger struct {
	serviceName string
	version     string
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp     string                 `json:"timestamp"`
	Level         string                 `json:"level"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	TraceID       string                 `json:"trace_id,omitempty"`
	SpanID        string                 `json:"span_id,omitempty"`
	Service       string                 `json:"service"`
	Version       string                 `json:"version"`
	Endpoint      string                 `json:"endpoint,omitempty"`
	Method        string                 `json:"method,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	TenantID      string                 `json:"tenant_id,omitempty"`
	InstanceID    string                 `json:"instance_id,omitempty"`
	StatusCode    int                    `json:"status_code,omitempty"`
	Duration      float64                `json:"duration_ms,omitempty"`
	Message       string                 `json:"message"`
	Error         string                 `json:"error,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(serviceName, version string) *StructuredLogger {
	return &StructuredLogger{
		serviceName: serviceName,
		version:     version,
	}
}

// RequestLogger returns a middleware that logs requests in structured format
func (l *StructuredLogger) RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		correlationID := GetCorrelationID(c)

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)
		durationMs := float64(duration.Nanoseconds()) / 1e6

		// Determine log level based on status code
		logLevel := "info"
		if c.Writer.Status() >= 500 {
			logLevel = "error"
		} else if c.Writer.Status() >= 400 {
			logLevel = "warn"
		}

		// Get user and tenant info if available
		userID := ""
		if id, exists := c.Get(string(types.ContextKeyUserID)); exists {
			userID = fmt.Sprintf("%v", id)
		}

		tenantID := ""
		if id, exists := c.Get(string(types.ContextKeyTenantID)); exists {
			tenantID = fmt.Sprintf("%v", id)
		}

		// Create log entry
		entry := LogEntry{
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
			Level:         logLevel,
			CorrelationID: correlationID,
			Service:       l.serviceName,
			Version:       l.version,
			Endpoint:      c.Request.URL.Path,
			Method:        c.Request.Method,
			UserID:        userID,
			TenantID:      tenantID,
			StatusCode:    c.Writer.Status(),
			Duration:      durationMs,
			Message:       fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
		}

		// Add error if present
		if len(c.Errors) > 0 {
			entry.Error = c.Errors.String()
		}

		// Log in JSON format
		logJSON(entry)
	}
}

// logJSON outputs a log entry as JSON
func logJSON(entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal log entry: %v", err)
		return
	}
	log.Println(string(data))
}

// Info logs an info level message
func (l *StructuredLogger) Info(ctx context.Context, message string, fields map[string]interface{}) {
	l.log(ctx, "info", message, fields)
}

// Warn logs a warning level message
func (l *StructuredLogger) Warn(ctx context.Context, message string, fields map[string]interface{}) {
	l.log(ctx, "warn", message, fields)
}

// Error logs an error level message
func (l *StructuredLogger) Error(ctx context.Context, message string, fields map[string]interface{}) {
	l.log(ctx, "error", message, fields)
}

// Debug logs a debug level message
func (l *StructuredLogger) Debug(ctx context.Context, message string, fields map[string]interface{}) {
	l.log(ctx, "debug", message, fields)
}

// Fatal logs a fatal level message and exits
func (l *StructuredLogger) Fatal(ctx context.Context, message string, fields map[string]interface{}) {
	l.log(ctx, "fatal", message, fields)
	log.Fatal(message)
}

// log outputs a structured log entry
func (l *StructuredLogger) log(ctx context.Context, level, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Service:   l.serviceName,
		Version:   l.version,
		Message:   message,
		Fields:    fields,
	}

	// Add correlation ID if available
	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		entry.CorrelationID = correlationID.(string)
	}

	// Add trace ID if available
	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		entry.TraceID = traceID.(string)
	}

	logJSON(entry)
}

// WithCorrelation creates a child logger with the given correlation ID
func (l *StructuredLogger) WithCorrelation(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// WithTrace creates a child logger with the given trace ID
func (l *StructuredLogger) WithTrace(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// RecoveryLogger recovers from panics and logs with correlation ID
func (l *StructuredLogger) RecoveryLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				correlationID := GetCorrelationID(c)

				entry := LogEntry{
					Timestamp:     time.Now().UTC().Format(time.RFC3339),
					Level:         "error",
					CorrelationID: correlationID,
					Service:       l.serviceName,
					Version:       l.version,
					Endpoint:      c.Request.URL.Path,
					Method:        c.Request.Method,
					StatusCode:    http.StatusInternalServerError,
					Message:       "Panic recovered",
					Error:         fmt.Sprintf("%v", err),
				}

				logJSON(entry)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  "INTERNAL_ERROR",
				})
			}
		}()
		c.Next()
	}
}
