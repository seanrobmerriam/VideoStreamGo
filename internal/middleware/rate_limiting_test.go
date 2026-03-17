package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

// Test_RateLimiter_BasicFunctionality tests basic rate limiting functionality
func Test_RateLimiter_BasicFunctionality(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(rate.Limit(60), 10)

	r := gin.New()
	r.Use(RateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First request should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_RateLimiter_ExceedsLimit tests that requests exceeding limit are rejected
func Test_RateLimiter_ExceedsLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Use a fresh limiter with very low limit (allows 2 requests per IP)
	limiter := NewRateLimiter(rate.Limit(2), 2)

	r := gin.New()
	r.Use(RateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make requests from the same IP
	// First 2 should succeed (burst), 3rd should be rate limited
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if i < 2 {
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, w.Code, "Request %d should be rate limited", i+1)
		}
	}
}

// Test_RateLimiter_DifferentTenants tests that different tenants have separate limits
func Test_RateLimiter_DifferentTenants(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(rate.Limit(10), 5)

	// Tenant 1 router with tenant middleware
	r1 := gin.New()
	tenant1ID := uuid.New()
	r1.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenant1ID)
		c.Next()
	})
	r1.Use(RateLimitMiddleware(limiter))
	r1.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Tenant 1 makes requests - should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r1.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Tenant 2 router with different tenant
	r2 := gin.New()
	tenant2ID := uuid.New()
	r2.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenant2ID)
		c.Next()
	})
	r2.Use(RateLimitMiddleware(limiter))
	r2.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Tenant 2 should have its own limit - should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r2.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_RateLimiter_PerTenantIsolation tests that one tenant cannot affect another
func Test_RateLimiter_PerTenantIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Very restrictive limiter - only allows 2 requests
	limiter := NewRateLimiter(rate.Limit(2), 2)

	r := gin.New()
	r.Use(RateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tenant1ID := uuid.New()

	// Tenant 1 uses up the limit - we need to simulate requests with tenant context
	// by adding a middleware that sets the tenant
	r2 := gin.New()
	r2.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenant1ID)
		c.Next()
	})
	r2.Use(RateLimitMiddleware(limiter))
	r2.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r2.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Tenant 1 should now be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r2.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Tenant 2 (different UUID) should still be able to make requests
	r3 := gin.New()
	tenant2ID := uuid.New()
	r3.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenant2ID)
		c.Next()
	})
	r3.Use(RateLimitMiddleware(limiter))
	r3.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r3.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Tenant 2 should not be rate limited by Tenant 1's usage")
}

// Test_RateLimiter_FallbackToIP tests fallback to IP when tenant ID is not available
func Test_RateLimiter_FallbackToIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(rate.Limit(2), 2)

	r := gin.New()
	r.Use(RateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First request without tenant ID - uses IP
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Should be rate limited (IP-based)
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Different IP should work
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.2:1234"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Different IP should have separate limit")
}

// Test_RateLimiter_ConcurrentAccess tests rate limiting under concurrent access
func Test_RateLimiter_ConcurrentAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Use a higher limit to allow more concurrent requests
	limiter := NewRateLimiter(rate.Limit(100), 50)

	r := gin.New()
	r.Use(RateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	var wg sync.WaitGroup
	results := make(chan int, 50)

	// Launch 50 concurrent requests
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			results <- w.Code
		}()
	}

	wg.Wait()
	close(results)

	// Count results
	successCount := 0
	rateLimitedCount := 0
	for code := range results {
		if code == http.StatusOK {
			successCount++
		} else if code == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// Most requests should succeed (within rate limit)
	// With 100 rps and 50 burst, 50 concurrent requests should all succeed
	assert.Greater(t, successCount, 45)
}

// Test_RateLimiter_ResetAfterWindow tests that limit resets after time window
func Test_RateLimiter_ResetAfterWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a limiter with 1 second window
	limiter := NewRateLimiter(rate.Limit(2), 1)

	r := gin.New()
	r.Use(RateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Use up the limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}

	// Should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Wait for rate limit window to reset
	time.Sleep(time.Second)

	// Should succeed after reset
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_GetRateLimitKey_TenantID tests that tenant ID is used when available
func Test_GetRateLimitKey_TenantID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Test with tenant ID
	tenantID := uuid.New()
	c.Set("tenant_id", tenantID)

	key := getRateLimitKey(c)
	assert.Equal(t, "tenant:"+tenantID.String(), key)
}

// Test_GetRateLimitKey_FallbackToIP tests fallback to IP when tenant ID is not available
func Test_GetRateLimitKey_FallbackToIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test without tenant ID - should fall back to IP
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:8080"
	c.Request = req

	key := getRateLimitKey(c)
	assert.Equal(t, "ip:192.168.1.100", key)
}

// Test_GetRateLimitKey_NilTenant tests fallback to IP when tenant ID is nil
func Test_GetRateLimitKey_NilTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set tenant_id to nil UUID
	c.Set("tenant_id", uuid.Nil)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:8080"
	c.Request = req

	key := getRateLimitKey(c)
	// Should fall back to IP when tenant ID is nil
	assert.Equal(t, "ip:192.168.1.100", key)
}
