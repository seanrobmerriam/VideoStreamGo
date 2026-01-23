package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"videostreamgo/internal/config"
	"videostreamgo/internal/database"
	"videostreamgo/internal/middleware"
	"videostreamgo/internal/routes/instance"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize instance database manager
	dbManager := database.NewTenantDBManager(cfg)

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
	router.Use(middleware.RateLimit())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"service":   "instance-api",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Setup instance routes
	instance.SetupRoutes(router, dbManager, cfg)

	// Start server on port 8081
	addr := fmt.Sprintf(":%d", cfg.App.Port+1) // Use Port + 1 for instance API
	log.Printf("Starting Instance API server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
