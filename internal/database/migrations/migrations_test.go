package migrations

import (
	"testing"
)

// TestMasterMigrations tests that all master migrations are defined correctly
func TestMasterMigrations(t *testing.T) {
	// Test that master migrations array exists and has correct length
	if MasterMigrations == nil {
		t.Fatal("MasterMigrations should not be nil")
	}

	expectedCount := 10
	if len(MasterMigrations) != expectedCount {
		t.Errorf("Expected %d master migrations, got %d", expectedCount, len(MasterMigrations))
	}

	// Verify each migration function is not nil
	for i, migration := range MasterMigrations {
		if migration == nil {
			t.Errorf("Master migration %d is nil", i+1)
		}
	}
}

// TestInstanceMigrations tests that all instance migrations are defined correctly
func TestInstanceMigrations(t *testing.T) {
	// Test that instance migrations array exists and has correct length
	if InstanceMigrations == nil {
		t.Fatal("InstanceMigrations should not be nil")
	}

	expectedCount := 15
	if len(InstanceMigrations) != expectedCount {
		t.Errorf("Expected %d instance migrations, got %d", expectedCount, len(InstanceMigrations))
	}

	// Verify each migration function is not nil
	for i, migration := range InstanceMigrations {
		if migration == nil {
			t.Errorf("Instance migration %d is nil", i+1)
		}
	}
}

// TestSeedData tests that seed data is properly structured
func TestSeedData(t *testing.T) {
	// Test that subscription plans have valid data
	if len(DefaultSeedData.SubscriptionPlans) == 0 {
		t.Error("Expected at least one subscription plan")
	}

	expectedPlans := 4
	if len(DefaultSeedData.SubscriptionPlans) != expectedPlans {
		t.Errorf("Expected %d subscription plans, got %d", expectedPlans, len(DefaultSeedData.SubscriptionPlans))
	}

	for i, plan := range DefaultSeedData.SubscriptionPlans {
		if plan.Name == "" {
			t.Errorf("Subscription plan %d has empty name", i)
		}
		if plan.MonthlyPrice < 0 {
			t.Errorf("Subscription plan %d has negative monthly price", i)
		}
		if plan.MaxStorageGB <= 0 {
			t.Errorf("Subscription plan %d has invalid max storage", i)
		}
		if plan.MaxBandwidthGB <= 0 {
			t.Errorf("Subscription plan %d has invalid max bandwidth", i)
		}
		if plan.MaxVideos <= 0 {
			t.Errorf("Subscription plan %d has invalid max videos", i)
		}
		if plan.MaxUsers <= 0 {
			t.Errorf("Subscription plan %d has invalid max users", i)
		}
	}

	// Test that platform settings have valid data
	if len(DefaultSeedData.PlatformSettings) == 0 {
		t.Error("Expected at least one platform setting")
	}

	for i, setting := range DefaultSeedData.PlatformSettings {
		if setting.Key == "" {
			t.Errorf("Platform setting %d has empty key", i)
		}
	}

	// Test that default categories have valid data
	if len(DefaultSeedData.DefaultCategories) == 0 {
		t.Error("Expected at least one default category")
	}

	expectedCategories := 8
	if len(DefaultSeedData.DefaultCategories) != expectedCategories {
		t.Errorf("Expected %d default categories, got %d", expectedCategories, len(DefaultSeedData.DefaultCategories))
	}

	for i, category := range DefaultSeedData.DefaultCategories {
		if category.Name == "" {
			t.Errorf("Default category %d has empty name", i)
		}
		if category.Slug == "" {
			t.Errorf("Default category %d has empty slug", i)
		}
		if category.Color == "" {
			t.Errorf("Default category %d has empty color", i)
		}
	}
}

// TestSubscriptionPlanPricing tests that subscription plan pricing follows expected tiers
func TestSubscriptionPlanPricing(t *testing.T) {
	expectedPricing := map[string]struct {
		monthly float64
		yearly  float64
	}{
		"Free":         {0, 0},
		"Starter":      {29, 290},
		"Professional": {99, 990},
		"Enterprise":   {299, 2990},
	}

	for _, plan := range DefaultSeedData.SubscriptionPlans {
		expected, ok := expectedPricing[plan.Name]
		if !ok {
			t.Errorf("Unexpected subscription plan: %s", plan.Name)
			continue
		}
		if plan.MonthlyPrice != expected.monthly {
			t.Errorf("Plan %s: expected monthly price %f, got %f", plan.Name, expected.monthly, plan.MonthlyPrice)
		}
		if plan.YearlyPrice != expected.yearly {
			t.Errorf("Plan %s: expected yearly price %f, got %f", plan.Name, expected.yearly, plan.YearlyPrice)
		}
	}
}

// TestDefaultCategoryColors tests that all default categories have valid hex colors
func TestDefaultCategoryColors(t *testing.T) {
	for i, category := range DefaultSeedData.DefaultCategories {
		if len(category.Color) != 7 {
			t.Errorf("Default category %d (%s): expected 7-character hex color, got '%s'", i, category.Name, category.Color)
		}
		if category.Color[0] != '#' {
			t.Errorf("Default category %d (%s): expected color to start with '#', got '%s'", i, category.Name, category.Color)
		}
	}
}

// TestMigrationRunnerCreation tests that migration runner can be created
func TestMigrationRunnerCreation(t *testing.T) {
	// Test that runner creation functions exist
	if MasterMigrationRunner == nil {
		t.Error("MasterMigrationRunner function should not be nil")
	}
	if InstanceMigrationRunner == nil {
		t.Error("InstanceMigrationRunner function should not be nil")
	}
}

// TestSeedDataCompleteness tests that all expected seed data is present
func TestSeedDataCompleteness(t *testing.T) {
	// Check that we have the expected subscription tiers
	tierNames := make(map[string]bool)
	for _, plan := range DefaultSeedData.SubscriptionPlans {
		tierNames[plan.Name] = true
	}

	expectedTiers := []string{"Free", "Starter", "Professional", "Enterprise"}
	for _, tier := range expectedTiers {
		if !tierNames[tier] {
			t.Errorf("Missing expected subscription tier: %s", tier)
		}
	}

	// Check for required platform settings
	settingKeys := make(map[string]bool)
	for _, setting := range DefaultSeedData.PlatformSettings {
		settingKeys[setting.Key] = true
	}

	requiredSettings := []string{"maintenance_mode", "allow_new_signups", "support_email"}
	for _, setting := range requiredSettings {
		if !settingKeys[setting] {
			t.Errorf("Missing required platform setting: %s", setting)
		}
	}

	// Check for required categories
	categoryNames := make(map[string]bool)
	for _, cat := range DefaultSeedData.DefaultCategories {
		categoryNames[cat.Name] = true
	}

	requiredCategories := []string{"Entertainment", "Education", "Gaming", "Music", "Sports"}
	for _, cat := range requiredCategories {
		if !categoryNames[cat] {
			t.Errorf("Missing required category: %s", cat)
		}
	}
}

// TestMasterMigrationFunctions tests migration function signatures
func TestMasterMigrationFunctions(t *testing.T) {
	migrationFuncs := []struct {
		name string
		fn   interface{}
	}{
		{"migrate001_createCustomers", migrate001_createCustomers},
		{"migrate002_createSubscriptionPlans", migrate002_createSubscriptionPlans},
		{"migrate003_createInstances", migrate003_createInstances},
		{"migrate004_createSubscriptions", migrate004_createSubscriptions},
		{"migrate005_createUsageMetrics", migrate005_createUsageMetrics},
		{"migrate006_createInstanceConfig", migrate006_createInstanceConfig},
		{"migrate007_createBillingRecords", migrate007_createBillingRecords},
		{"migrate008_createLicenses", migrate008_createLicenses},
		{"migrate009_createAdminUsers", migrate009_createAdminUsers},
		{"migrate010_createPlatformSettings", migrate010_createPlatformSettings},
	}

	for _, mf := range migrationFuncs {
		if mf.fn == nil {
			t.Errorf("Migration function %s is nil", mf.name)
		}
	}
}

// TestInstanceMigrationFunctions tests migration function signatures
func TestInstanceMigrationFunctions(t *testing.T) {
	migrationFuncs := []struct {
		name string
		fn   interface{}
	}{
		{"instanceMigrate001_createUsers", instanceMigrate001_createUsers},
		{"instanceMigrate002_createVideos", instanceMigrate002_createVideos},
		{"instanceMigrate003_createCategories", instanceMigrate003_createCategories},
		{"instanceMigrate004_createTags", instanceMigrate004_createTags},
		{"instanceMigrate005_createVideoTags", instanceMigrate005_createVideoTags},
		{"instanceMigrate006_createComments", instanceMigrate006_createComments},
		{"instanceMigrate007_createRatings", instanceMigrate007_createRatings},
		{"instanceMigrate008_createFavorites", instanceMigrate008_createFavorites},
		{"instanceMigrate009_createPlaylists", instanceMigrate009_createPlaylists},
		{"instanceMigrate010_createPlaylistVideos", instanceMigrate010_createPlaylistVideos},
		{"instanceMigrate011_createVideoViews", instanceMigrate011_createVideoViews},
		{"instanceMigrate012_createUserSessions", instanceMigrate012_createUserSessions},
		{"instanceMigrate013_createBrandingConfig", instanceMigrate013_createBrandingConfig},
		{"instanceMigrate014_createPages", instanceMigrate014_createPages},
		{"instanceMigrate015_createSettings", instanceMigrate015_createSettings},
	}

	for _, mf := range migrationFuncs {
		if mf.fn == nil {
			t.Errorf("Migration function %s is nil", mf.name)
		}
	}
}
