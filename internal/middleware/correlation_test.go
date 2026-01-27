package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"videostreamgo/internal/types"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCorrelationMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(CorrelationMiddleware())
	router.GET("/test", func(c *gin.Context) {
		correlationID := GetCorrelationID(c)
		c.JSON(http.StatusOK, gin.H{"correlation_id": correlationID})
	})

	t.Run("generates new correlation ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		headerID := w.Header().Get("X-Correlation-ID")
		assert.True(t, headerID != "", "X-Correlation-ID should not be empty")

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["correlation_id"] != "", "correlation_id should not be empty")
	})

	t.Run("uses existing correlation ID from header", func(t *testing.T) {
		existingID := "existing-correlation-id-123"
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Correlation-ID", existingID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, existingID, w.Header().Get("X-Correlation-ID"))

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, existingID, response["correlation_id"])
	})

	t.Run("uses X-Request-ID as fallback", func(t *testing.T) {
		requestID := "request-id-456"
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", requestID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, requestID, w.Header().Get("X-Correlation-ID"))
	})
}

func TestGetCorrelationIDFromContext(t *testing.T) {
	router := gin.New()
	router.Use(CorrelationMiddleware())
	router.GET("/test", func(c *gin.Context) {
		correlationID := GetCorrelationID(c)
		c.JSON(http.StatusOK, gin.H{"correlation_id": correlationID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["correlation_id"] != "", "correlation_id should not be empty")
}

func TestNewStructuredLogger(t *testing.T) {
	logger := NewStructuredLogger("test-service", "1.0.0")

	assert.NotNil(t, logger)
	assert.Equal(t, "test-service", logger.serviceName)
	assert.Equal(t, "1.0.0", logger.version)
}

func TestStructuredLoggerInfo(t *testing.T) {
	logger := NewStructuredLogger("test-service", "1.0.0")

	// This should not panic
	logger.Info(context.Background(), "test message", map[string]interface{}{
		"key": "value",
	})
}

func TestStructuredLoggerWarn(t *testing.T) {
	logger := NewStructuredLogger("test-service", "1.0.0")

	logger.Warn(context.Background(), "warning message", nil)
}

func TestStructuredLoggerError(t *testing.T) {
	logger := NewStructuredLogger("test-service", "1.0.0")

	logger.Error(context.Background(), "error message", map[string]interface{}{
		"error": "test error",
	})
}

func TestStructuredLoggerDebug(t *testing.T) {
	logger := NewStructuredLogger("test-service", "1.0.0")

	logger.Debug(context.Background(), "debug message", nil)
}

func TestStructuredLoggerWithCorrelation(t *testing.T) {
	logger := NewStructuredLogger("test-service", "1.0.0")

	ctx := context.Background()
	correlatedCtx := logger.WithCorrelation(ctx, "test-correlation-id")

	// The correlation ID should be in the context
	id := correlatedCtx.Value(CorrelationIDKey)
	assert.Equal(t, "test-correlation-id", id)
}

func TestStructuredLoggerWithTrace(t *testing.T) {
	logger := NewStructuredLogger("test-service", "1.0.0")

	ctx := context.Background()
	tracedCtx := logger.WithTrace(ctx, "test-trace-id")

	// The trace ID should be in the context
	id := tracedCtx.Value(TraceIDKey)
	assert.Equal(t, "test-trace-id", id)
}

func TestRecoveryLogger(t *testing.T) {
	logger := NewStructuredLogger("test-service", "1.0.0")

	router := gin.New()
	router.Use(logger.RecoveryLogger())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()

	// This should not panic
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRequestLoggerMiddleware(t *testing.T) {
	logger := NewStructuredLogger("test-service", "1.0.0")

	router := gin.New()
	router.Use(logger.RequestLogger())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogEntry(t *testing.T) {
	entry := LogEntry{
		Timestamp:     "2025-01-26T12:00:00Z",
		Level:         "info",
		CorrelationID: "test-correlation-id",
		Service:       "test-service",
		Version:       "1.0.0",
		Endpoint:      "/test",
		Method:        "GET",
		StatusCode:    200,
		Duration:      100.5,
		Message:       "test message",
	}

	assert.Equal(t, "2025-01-26T12:00:00Z", entry.Timestamp)
	assert.Equal(t, "info", entry.Level)
	assert.Equal(t, "test-correlation-id", entry.CorrelationID)
	assert.Equal(t, "test-service", entry.Service)
	assert.Equal(t, "1.0.0", entry.Version)
	assert.Equal(t, "/test", entry.Endpoint)
	assert.Equal(t, "GET", entry.Method)
	assert.Equal(t, 200, entry.StatusCode)
	assert.Equal(t, 100.5, entry.Duration)
	assert.Equal(t, "test message", entry.Message)
}

func TestContextKeys(t *testing.T) {
	assert.Equal(t, contextKey("correlation_id"), CorrelationIDKey)
	assert.Equal(t, contextKey("trace_id"), TraceIDKey)
}

func TestStructuredLoggerWithUserContext(t *testing.T) {
	logger := NewStructuredLogger("test-service", "1.0.0")

	router := gin.New()
	router.Use(CorrelationMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set(string(types.ContextKeyUserID), "user-123")
		c.Set(string(types.ContextKeyTenantID), "tenant-456")
		c.Next()
	})
	router.Use(logger.RequestLogger())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
