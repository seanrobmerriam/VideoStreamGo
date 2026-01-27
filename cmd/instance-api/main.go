package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"videostreamgo/internal/cache"
	"videostreamgo/internal/config"
	"videostreamgo/internal/database"
	"videostreamgo/internal/health"
	"videostreamgo/internal/metrics"
	"videostreamgo/internal/middleware"
	"videostreamgo/internal/routes/instance"
	"videostreamgo/internal/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize instance database manager
	dbManager := database.NewTenantDBManager(cfg)

	// Initialize Redis client
	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}

	// Connect to Redis with retry
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := redisClient.Connect(ctx); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v (continuing without Redis)", err)
	} else {
		log.Printf("Redis connection established")
	}

	// Initialize MinIO client
	minioClient, err := storage.NewMinioClient(cfg)
	if err != nil {
		log.Printf("Warning: Failed to create MinIO client: %v (continuing without MinIO)", err)
		minioClient = nil
	} else {
		// Connect to MinIO and create buckets
		if err := minioClient.Connect(ctx); err != nil {
			log.Printf("Warning: Failed to connect to MinIO: %v (continuing without MinIO)", err)
			minioClient = nil
		} else {
			log.Printf("MinIO connection established")
		}
	}

	// Initialize health checker (instance-api doesn't use master DB, only Redis and MinIO)
	healthChecker := health.NewChecker(nil, redisClient, minioClient, cfg, "1.0.0")

	// Initialize metrics (blank identifier since Handler uses global prometheus registry)
	_ = metrics.NewMetrics("instance-api", "1.0.0")

	// Set Gin mode
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin router
	router := gin.New()

	// Add global middleware
	router.Use(middleware.RecoveryLogger())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())

	// Initialize rate limiter with Redis
	rateLimiter := middleware.NewRedisRateLimiter(redisClient.GetClient(), cfg)
	router.Use(middleware.RedisRateLimitMiddleware(rateLimiter))

	// Health check endpoints using health checker
	router.GET("/health", func(c *gin.Context) {
		// Simple check for load balancers - returns 200 if all critical dependencies healthy
		if err := healthChecker.SimpleCheck(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "instance-api",
		})
	})

	router.GET("/health/detailed", func(c *gin.Context) {
		result := healthChecker.Check(c.Request.Context())
		statusCode := http.StatusOK
		if result.Status == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}
		c.JSON(statusCode, result)
	})

	// Prometheus metrics endpoint
	router.GET("/metrics", metrics.Handler())

	// Setup instance routes
	instance.SetupRoutes(router, dbManager, cfg)

	// Create server with graceful shutdown
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.App.Port),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting Instance API server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}

	// Close MinIO client
	if minioClient != nil {
		if err := minioClient.Close(); err != nil {
			log.Printf("Error closing MinIO connection: %v", err)
		}
	}

	log.Println("Server exited gracefully")
}
