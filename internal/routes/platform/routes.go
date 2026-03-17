package platform

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"videostreamgo/internal/config"
	handlers "videostreamgo/internal/handlers/platform"
	"videostreamgo/internal/middleware"
	masterModels "videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
	platformsvc "videostreamgo/internal/services/platform"
)

// SetupRoutes configures all platform API routes
func SetupRoutes(router *gin.Engine, db *gorm.DB, cfg *config.Config) {
	// Initialize repositories
	adminRepo := masterRepo.NewAdminRepository(db)
	customerRepo := masterRepo.NewCustomerRepository(db)
	instanceRepo := masterRepo.NewInstanceRepository(db)
	subscriptionRepo := masterRepo.NewSubscriptionRepository(db)
	planRepo := masterRepo.NewPlanRepository(db)
	billingRepo := masterRepo.NewBillingRecordRepository(db)

	// Initialize services
	instanceProvisioner, err := platformsvc.NewInstanceProvisioner(db, instanceRepo, cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create instance provisioner: %v", err))
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(adminRepo, cfg)
	customerHandler := handlers.NewCustomerHandler(customerRepo, instanceRepo, cfg)
	instanceHandler := handlers.NewInstanceHandler(instanceRepo, customerRepo, subscriptionRepo, instanceProvisioner, cfg)
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionRepo, planRepo, customerRepo, cfg)
	billingHandler := handlers.NewBillingHandler(billingRepo, customerRepo, cfg)

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// Public routes (no authentication required)
		auth := v1.Group("/auth")
		{
			auth.POST("/admin/login", authHandler.Login)
			auth.POST("/admin/register", authHandler.Register)
		}

		// Protected routes (require admin authentication)
		admin := v1.Group("/admin")
		admin.Use(middleware.AdminAuthMiddleware(db, cfg))
		{
			// Admin management
			admin.GET("/me", authHandler.GetCurrentAdmin)
			admin.PUT("/password", authHandler.ChangePassword)

			// Customer management
			customers := admin.Group("/customers")
			{
				customers.GET("", customerHandler.List)
				customers.POST("", customerHandler.Create)
				customers.GET("/:id", customerHandler.Get)
				customers.PUT("/:id", customerHandler.Update)
				customers.DELETE("/:id", customerHandler.Delete)
			}

			// Instance management
			instances := admin.Group("/instances")
			{
				instances.GET("", instanceHandler.List)
				instances.POST("", instanceHandler.Create)
				instances.GET("/:id", instanceHandler.Get)
				instances.PUT("/:id", instanceHandler.Update)
				instances.DELETE("/:id", instanceHandler.Delete)
				instances.POST("/:id/provision", instanceHandler.Provision)
				instances.POST("/:id/deprovision", instanceHandler.Deprovision)
				instances.GET("/:id/status", instanceHandler.Status)
				instances.POST("/:id/custom-domains", instanceHandler.CustomDomains)
				instances.GET("/:id/metrics", instanceHandler.Metrics)
				instances.POST("/:id/suspend", instanceHandler.Suspend)
				instances.POST("/:id/activate", instanceHandler.Activate)
			}

			// Subscription management
			subscriptions := admin.Group("/subscriptions")
			{
				subscriptions.GET("", subscriptionHandler.List)
				subscriptions.POST("", subscriptionHandler.Create)
				subscriptions.GET("/:id", subscriptionHandler.Get)
				subscriptions.PUT("/:id", subscriptionHandler.Update)
				subscriptions.DELETE("/:id", subscriptionHandler.Cancel)
			}

			// Plan management
			plans := admin.Group("/plans")
			{
				plans.GET("", subscriptionHandler.ListPlans)
				plans.POST("", subscriptionHandler.CreatePlan)
				plans.GET("/:id", subscriptionHandler.GetPlan)
				plans.PUT("/:id", subscriptionHandler.UpdatePlan)
				plans.DELETE("/:id", subscriptionHandler.DeletePlan)
			}

			// Billing management - requires super_admin role
			billing := admin.Group("/billing")
			billing.Use(middleware.RequireRole(masterModels.AdminRoleSuperAdmin))
			{
				// Billing records
				billing.GET("/records", billingHandler.ListRecords)
				billing.GET("/records/:id", billingHandler.GetRecord)
				billing.POST("/records", billingHandler.CreateRecord)

				// Customer billing
				billing.GET("/customers", billingHandler.ListCustomers)
				billing.GET("/customers/:id", billingHandler.GetCustomerBilling)
				billing.POST("/customers/:id/invoice", billingHandler.GenerateInvoice)

				// Reports and analytics
				billing.GET("/reports", billingHandler.GetBillingReports)
				billing.GET("/revenue", billingHandler.GetRevenueAnalytics)

				// Usage metrics
				billing.GET("/usage/:instance_id", billingHandler.GetUsageMetrics)
			}

			// Analytics - requires super_admin role (shares billing handlers)
			analytics := admin.Group("/analytics")
			analytics.Use(middleware.RequireRole(masterModels.AdminRoleSuperAdmin))
			{
				analytics.GET("/overview", billingHandler.GetOverview)
				analytics.GET("/revenue", billingHandler.GetRevenueReport)
				analytics.GET("/usage", billingHandler.GetUsageReport)
			}
		}

		// Stripe webhook (no auth required, would be handled by webhook handler)
		v1.POST("/webhooks/stripe", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "webhook endpoint ready"})
		})
	}
}
