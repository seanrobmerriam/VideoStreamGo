package migrations

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedData contains all seed data for the platform
type SeedData struct {
	SubscriptionPlans []SubscriptionPlanSeed
	PlatformSettings  []PlatformSettingSeed
	DefaultCategories []DefaultCategorySeed
}

// SubscriptionPlanSeed represents a subscription plan to seed
type SubscriptionPlanSeed struct {
	Name           string
	Description    string
	MonthlyPrice   float64
	YearlyPrice    float64
	MaxStorageGB   int
	MaxBandwidthGB int
	MaxVideos      int
	MaxUsers       int
	Features       string
	SortOrder      int
}

// PlatformSettingSeed represents a platform setting to seed
type PlatformSettingSeed struct {
	Key         string
	Value       string
	Description string
}

// DefaultCategorySeed represents a default category to seed for new instances
type DefaultCategorySeed struct {
	Name        string
	Slug        string
	Description string
	Color       string
	SortOrder   int
}

// DefaultSeedData contains the default seed data
var DefaultSeedData = SeedData{
	SubscriptionPlans: []SubscriptionPlanSeed{
		{
			Name:           "Free",
			Description:    "Perfect for getting started with your own video platform",
			MonthlyPrice:   0,
			YearlyPrice:    0,
			MaxStorageGB:   10,
			MaxBandwidthGB: 100,
			MaxVideos:      100,
			MaxUsers:       1000,
			Features:       `["basic_branding", "video_upload", "basic_analytics", "community_features"]`,
			SortOrder:      1,
		},
		{
			Name:           "Starter",
			Description:    "Great for growing communities and small businesses",
			MonthlyPrice:   29,
			YearlyPrice:    290,
			MaxStorageGB:   100,
			MaxBandwidthGB: 1000,
			MaxVideos:      5000,
			MaxUsers:       10000,
			Features:       `["custom_branding", "video_upload", "advanced_analytics", "priority_support", "no_watermark"]`,
			SortOrder:      2,
		},
		{
			Name:           "Professional",
			Description:    "For professional video content creators and businesses",
			MonthlyPrice:   99,
			YearlyPrice:    990,
			MaxStorageGB:   500,
			MaxBandwidthGB: 5000,
			MaxVideos:      25000,
			MaxUsers:       50000,
			Features:       `["custom_branding", "video_upload", "advanced_analytics", "priority_support", "no_watermark", "custom_domain", "api_access", "white_label"]`,
			SortOrder:      3,
		},
		{
			Name:           "Enterprise",
			Description:    "Maximum power and flexibility for large organizations",
			MonthlyPrice:   299,
			YearlyPrice:    2990,
			MaxStorageGB:   2000,
			MaxBandwidthGB: 20000,
			MaxVideos:      100000,
			MaxUsers:       100000,
			Features:       `["custom_branding", "video_upload", "advanced_analytics", "priority_support", "no_watermark", "custom_domain", "api_access", "white_label", "dedicated_support", "sla_guarantee", "advanced_security"]`,
			SortOrder:      4,
		},
	},
	PlatformSettings: []PlatformSettingSeed{
		{
			Key:         "maintenance_mode",
			Value:       "false",
			Description: "Enable maintenance mode to prevent new signups and show maintenance page",
		},
		{
			Key:         "allow_new_signups",
			Value:       "true",
			Description: "Allow new customers to sign up for the platform",
		},
		{
			Key:         "support_email",
			Value:       "support@videostreamgo.com",
			Description: "Support email address for customer inquiries",
		},
		{
			Key:         "tos_url",
			Value:       "https://videostreamgo.com/terms",
			Description: "URL to Terms of Service",
		},
		{
			Key:         "privacy_policy_url",
			Value:       "https://videostreamgo.com/privacy",
			Description: "URL to Privacy Policy",
		},
		{
			Key:         "default_plan_id",
			Value:       "",
			Description: "Default subscription plan ID for new signups (leave empty to require plan selection)",
		},
		{
			Key:         "max_instances_per_customer",
			Value:       "10",
			Description: "Maximum number of instances a customer can create",
		},
		{
			Key:         "trial_days",
			Value:       "14",
			Description: "Number of days for trial period",
		},
	},
	DefaultCategories: []DefaultCategorySeed{
		{
			Name:        "Entertainment",
			Slug:        "entertainment",
			Description: "Fun and entertaining videos",
			Color:       "#e91e63",
			SortOrder:   1,
		},
		{
			Name:        "Education",
			Slug:        "education",
			Description: "Educational and tutorial content",
			Color:       "#2196f3",
			SortOrder:   2,
		},
		{
			Name:        "Gaming",
			Slug:        "gaming",
			Description: "Gaming videos and streams",
			Color:       "#9c27b0",
			SortOrder:   3,
		},
		{
			Name:        "Music",
			Slug:        "music",
			Description: "Music videos and audio content",
			Color:       "#f44336",
			SortOrder:   4,
		},
		{
			Name:        "Sports",
			Slug:        "sports",
			Description: "Sports and athletic content",
			Color:       "#4caf50",
			SortOrder:   5,
		},
		{
			Name:        "Technology",
			Slug:        "technology",
			Description: "Tech reviews and tutorials",
			Color:       "#00bcd4",
			SortOrder:   6,
		},
		{
			Name:        "Lifestyle",
			Slug:        "lifestyle",
			Description: "Lifestyle and vlog content",
			Color:       "#ff9800",
			SortOrder:   7,
		},
		{
			Name:        "News",
			Slug:        "news",
			Description: "News and current events",
			Color:       "#607d8b",
			SortOrder:   8,
		},
	},
}

// SeedMasterDatabase seeds the master database with default data
func SeedMasterDatabase(db *gorm.DB) error {
	// Seed subscription plans
	if err := seedSubscriptionPlans(db); err != nil {
		return fmt.Errorf("failed to seed subscription plans: %w", err)
	}

	// Seed platform settings
	if err := seedPlatformSettings(db); err != nil {
		return fmt.Errorf("failed to seed platform settings: %w", err)
	}

	// Seed default admin user
	if err := seedDefaultAdminUser(db); err != nil {
		return fmt.Errorf("failed to seed default admin user: %w", err)
	}

	return nil
}

// seedSubscriptionPlans creates the default subscription plans
func seedSubscriptionPlans(db *gorm.DB) error {
	for _, plan := range DefaultSeedData.SubscriptionPlans {
		// Check if plan already exists
		var count int64
		db.Table("subscription_plans").Where("name = ?", plan.Name).Count(&count)
		if count > 0 {
			continue
		}

		sql := `
			INSERT INTO subscription_plans (id, name, description, monthly_price, yearly_price, max_storage_gb, max_bandwidth_gb, max_videos, max_users, features, sort_order, created_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		`
		if err := db.Exec(sql, plan.Name, plan.Description, plan.MonthlyPrice, plan.YearlyPrice,
			plan.MaxStorageGB, plan.MaxBandwidthGB, plan.MaxVideos, plan.MaxUsers,
			plan.Features, plan.SortOrder).Error; err != nil {
			return err
		}
	}
	return nil
}

// seedPlatformSettings creates the default platform settings
func seedPlatformSettings(db *gorm.DB) error {
	for _, setting := range DefaultSeedData.PlatformSettings {
		// Check if setting already exists
		var count int64
		db.Table("platform_settings").Where("key = ?", setting.Key).Count(&count)
		if count > 0 {
			continue
		}

		sql := `
			INSERT INTO platform_settings (id, key, value, description, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, NOW())
			ON CONFLICT (key) DO NOTHING
		`
		if err := db.Exec(sql, setting.Key, setting.Value, setting.Description).Error; err != nil {
			return err
		}
	}
	return nil
}

// seedDefaultAdminUser creates the default admin user
func seedDefaultAdminUser(db *gorm.DB) error {
	// Get admin credentials from environment variables or use defaults
	adminEmail := getEnvOrDefault("ADMIN_EMAIL", "admin@videostreamgo.com")
	adminPassword := getEnvOrDefault("ADMIN_PASSWORD", "ChangeMe123!")
	adminName := getEnvOrDefault("ADMIN_NAME", "Super Admin")

	// Check if admin user already exists
	var count int64
	db.Table("admin_users").Where("email = ?", adminEmail).Count(&count)
	if count > 0 {
		return nil
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	sql := `
		INSERT INTO admin_users (id, email, password_hash, display_name, role, status, permissions, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, 'super_admin', 'active', '[]', NOW(), NOW())
	`
	if err := db.Exec(sql, adminEmail, string(hashedPassword), adminName).Error; err != nil {
		return err
	}

	return nil
}

// SeedInstanceDatabase seeds a new instance database with default data
func SeedInstanceDatabase(db *gorm.DB, instanceID uuid.UUID) error {
	// Seed default categories
	if err := seedDefaultCategories(db, instanceID); err != nil {
		return fmt.Errorf("failed to seed default categories: %w", err)
	}

	// Seed default settings
	if err := seedDefaultSettings(db, instanceID); err != nil {
		return fmt.Errorf("failed to seed default settings: %w", err)
	}

	// Seed default branding config
	if err := seedDefaultBrandingConfig(db, instanceID); err != nil {
		return fmt.Errorf("failed to seed default branding config: %w", err)
	}

	// Seed default pages
	if err := seedDefaultPages(db, instanceID); err != nil {
		return fmt.Errorf("failed to seed default pages: %w", err)
	}

	return nil
}

// seedDefaultCategories creates default categories for a new instance
func seedDefaultCategories(db *gorm.DB, instanceID uuid.UUID) error {
	for _, category := range DefaultSeedData.DefaultCategories {
		sql := `
			INSERT INTO categories (id, instance_id, name, slug, description, color, sort_order, is_active, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, true, NOW(), NOW())
		`
		if err := db.Exec(sql, instanceID, category.Name, category.Slug, category.Description, category.Color, category.SortOrder).Error; err != nil {
			return err
		}
	}
	return nil
}

// seedDefaultSettings creates default settings for a new instance
func seedDefaultSettings(db *gorm.DB, instanceID uuid.UUID) error {
	defaultSettings := []struct {
		Key   string
		Value string
		Desc  string
	}{
		{"registration_enabled", "true", "Allow user registration"},
		{"email_verification_required", "true", "Require email verification for new users"},
		{"allow_anonymous_views", "true", "Allow anonymous users to view videos"},
		{"default_video_visibility", "public", "Default visibility for new videos"},
		{"max_upload_size_mb", "1024", "Maximum video upload size in MB"},
		{"allowed_video_formats", "mp4,webm,mov,avi", "Allowed video formats for upload"},
		{"enable_comments", "true", "Enable video comments"},
		{"enable_ratings", "true", "Enable video ratings"},
		{"enable_favorites", "true", "Enable video favorites"},
		{"enable_playlists", "true", "Enable user playlists"},
		{"videos_per_page", "24", "Number of videos per page in listings"},
		{"comments_per_page", "50", "Number of comments per page"},
		{"enable_watermark", "false", "Enable video watermark"},
		{"enable_cdn", "false", "Enable CDN for video delivery"},
	}

	for _, setting := range defaultSettings {
		sql := `
			INSERT INTO settings (id, instance_id, key, value, description, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW())
			ON CONFLICT (instance_id, key) DO NOTHING
		`
		if err := db.Exec(sql, instanceID, setting.Key, setting.Value, setting.Desc).Error; err != nil {
			return err
		}
	}
	return nil
}

// seedDefaultBrandingConfig creates default branding configuration for a new instance
func seedDefaultBrandingConfig(db *gorm.DB, instanceID uuid.UUID) error {
	sql := `
		INSERT INTO branding_config (id, instance_id, site_name, primary_color, secondary_color, accent_color, background_color, text_color, social_links, footer_links, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, 'VideoTube', '#2563eb', '#64748b', '#f59e0b', '#ffffff', '#1e293b', '{}', '[]', NOW(), NOW())
		ON CONFLICT (instance_id) DO NOTHING
	`
	return db.Exec(sql, instanceID).Error
}

// seedDefaultPages creates default pages for a new instance
func seedDefaultPages(db *gorm.DB, instanceID uuid.UUID) error {
	defaultPages := []struct {
		Title     string
		Slug      string
		Content   string
		PageType  string
		ShowMenu  bool
		SortOrder int
	}{
		{
			Title:     "Home",
			Slug:      "home",
			Content:   "<h1>Welcome to our video platform</h1><p>Discover amazing videos from our community.</p>",
			PageType:  "static",
			ShowMenu:  false,
			SortOrder: 0,
		},
		{
			Title:     "About",
			Slug:      "about",
			Content:   "<h1>About Us</h1><p>Learn more about our video platform and our mission.</p>",
			PageType:  "custom",
			ShowMenu:  true,
			SortOrder: 10,
		},
		{
			Title:     "Contact",
			Slug:      "contact",
			Content:   "<h1>Contact Us</h1><p>Get in touch with our team.</p>",
			PageType:  "custom",
			ShowMenu:  true,
			SortOrder: 20,
		},
		{
			Title:     "Terms of Service",
			Slug:      "terms",
			Content:   "<h1>Terms of Service</h1><p>Please read our terms of service carefully.</p>",
			PageType:  "legal",
			ShowMenu:  false,
			SortOrder: 0,
		},
		{
			Title:     "Privacy Policy",
			Slug:      "privacy",
			Content:   "<h1>Privacy Policy</h1><p>Learn how we protect your privacy.</p>",
			PageType:  "legal",
			ShowMenu:  false,
			SortOrder: 0,
		},
	}

	for _, page := range defaultPages {
		sql := `
			INSERT INTO pages (id, instance_id, title, slug, content, page_type, is_published, show_in_menu, menu_order, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, true, $6, $7, NOW(), NOW())
			ON CONFLICT (slug) DO NOTHING
		`
		if err := db.Exec(sql, instanceID, page.Title, page.Slug, page.Content, page.PageType, page.ShowMenu, page.SortOrder).Error; err != nil {
			return err
		}
	}
	return nil
}

// getEnvOrDefault returns the environment variable value or a default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// RunSeeds runs all seed functions
func RunSeeds(db *gorm.DB) error {
	startTime := time.Now()
	fmt.Println("Running database seeds...")

	if err := SeedMasterDatabase(db); err != nil {
		return err
	}

	fmt.Printf("Seeds completed in %v\n", time.Since(startTime))
	return nil
}
