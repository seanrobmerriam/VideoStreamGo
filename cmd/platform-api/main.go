package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"videostreamgo/internal/config"
	"videostreamgo/internal/database"
	"videostreamgo/internal/middleware"
	"videostreamgo/internal/routes/platform"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize master database
	masterDB, err := database.NewMasterDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to master database: %v", err)
	}
	defer masterDB.Close()

	// Run migrations
	if err := masterDB.Migrate(); err != nil {
		log.Printf("Warning: Failed to run migrations: %v", err)
	}

	// Set Gin mode
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin router
	router := gin.New()

	// Add global middleware
	router.Use(middleware.RecoveryLogger())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.CORS(cfg.App.AllowedOrigins))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RateLimit())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"service":   "platform-api",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Setup platform routes
	platform.SetupRoutes(router, masterDB.GetDB(), cfg)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.App.Port)
	log.Printf("Starting Platform API server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
