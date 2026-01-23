package migrations

import (
	"gorm.io/gorm"
)

// MasterMigrations contains all master database migrations
var MasterMigrations = []func(*gorm.DB) error{
	migrate001_createCustomers,
	migrate002_createSubscriptionPlans,
	migrate003_createInstances,
	migrate004_createSubscriptions,
	migrate005_createUsageMetrics,
	migrate006_createInstanceConfig,
	migrate007_createBillingRecords,
	migrate008_createLicenses,
	migrate009_createAdminUsers,
	migrate010_createPlatformSettings,
}

// migrate001_createCustomers creates the customers table
func migrate001_createCustomers(db *gorm.DB) error {
	// Create the customers table with all constraints and indexes
	sql := `
		CREATE TABLE IF NOT EXISTS customers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			company_name VARCHAR(255) NOT NULL,
			contact_name VARCHAR(255),
			phone VARCHAR(50),
			status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'cancelled', 'pending')),
			stripe_customer_id VARCHAR(255),
			billing_email VARCHAR(255),
			tax_id VARCHAR(100),
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes for customers table
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_customers_status ON customers(status);`,
		`CREATE INDEX IF NOT EXISTS idx_customers_created_at ON customers(created_at DESC);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// migrate002_createSubscriptionPlans creates the subscription_plans table
func migrate002_createSubscriptionPlans(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS subscription_plans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			description TEXT,
			monthly_price DECIMAL(10, 2) NOT NULL DEFAULT 0,
			yearly_price DECIMAL(10, 2) NOT NULL DEFAULT 0,
			stripe_monthly_price_id VARCHAR(255),
			stripe_yearly_price_id VARCHAR(255),
			max_storage_gb INTEGER NOT NULL DEFAULT 100,
			max_bandwidth_gb INTEGER NOT NULL DEFAULT 1000,
			max_videos INTEGER NOT NULL DEFAULT 10000,
			max_users INTEGER NOT NULL DEFAULT 10000,
			features JSONB DEFAULT '[]',
			is_active BOOLEAN DEFAULT true,
			sort_order INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_subscription_plans_is_active ON subscription_plans(is_active);`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_plans_sort_order ON subscription_plans(sort_order);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// migrate003_createInstances creates the instances table
func migrate003_createInstances(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS instances (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			subdomain VARCHAR(63) UNIQUE NOT NULL,
			custom_domains TEXT[] DEFAULT '{}',
			status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'provisioning', 'active', 'suspended', 'terminated')),
			plan_id UUID REFERENCES subscription_plans(id),
			database_name VARCHAR(63) NOT NULL,
			storage_bucket VARCHAR(63) NOT NULL,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			activated_at TIMESTAMP WITH TIME ZONE
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes for instances table
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_instances_customer ON instances(customer_id);`,
		`CREATE INDEX IF NOT EXISTS idx_instances_subdomain ON instances(subdomain);`,
		`CREATE INDEX IF NOT EXISTS idx_instances_status ON instances(status);`,
		`CREATE INDEX IF NOT EXISTS idx_instances_created_at ON instances(created_at DESC);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// migrate004_createSubscriptions creates the subscriptions table
func migrate004_createSubscriptions(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS subscriptions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			customer_id UUID NOT NULL REFERENCES customers(id),
			plan_id UUID NOT NULL REFERENCES subscription_plans(id),
			status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'cancelled', 'past_due', 'paused', 'trialing')),
			billing_cycle VARCHAR(20) DEFAULT 'monthly' CHECK (billing_cycle IN ('monthly', 'yearly')),
			stripe_subscription_id VARCHAR(255),
			stripe_customer_id VARCHAR(255),
			current_period_start TIMESTAMP WITH TIME ZONE,
			current_period_end TIMESTAMP WITH TIME ZONE,
			cancel_at_period_end BOOLEAN DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_customer ON subscriptions(customer_id);`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_plan ON subscriptions(plan_id);`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe ON subscriptions(stripe_subscription_id);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// migrate005_createUsageMetrics creates the usage_metrics table
func migrate005_createUsageMetrics(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS usage_metrics (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
			metric_type VARCHAR(50) NOT NULL CHECK (metric_type IN ('storage', 'bandwidth', 'videos', 'users', 'views')),
			period_start TIMESTAMP WITH TIME ZONE NOT NULL,
			period_end TIMESTAMP WITH TIME ZONE NOT NULL,
			value BIGINT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(instance_id, metric_type, period_start)
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_usage_metrics_instance ON usage_metrics(instance_id, metric_type);`,
		`CREATE INDEX IF NOT EXISTS idx_usage_metrics_period ON usage_metrics(period_start, period_end);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// migrate006_createInstanceConfig creates the instance_config table
func migrate006_createInstanceConfig(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS instance_config (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
			config_key VARCHAR(100) NOT NULL,
			config_value TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(instance_id, config_key)
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add index
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_instance_config_instance ON instance_config(instance_id);`).Error; err != nil {
		return err
	}

	return nil
}

// migrate007_createBillingRecords creates the billing_records table
func migrate007_createBillingRecords(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS billing_records (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			customer_id UUID NOT NULL REFERENCES customers(id),
			subscription_id UUID REFERENCES subscriptions(id),
			amount DECIMAL(10, 2) NOT NULL,
			currency VARCHAR(3) DEFAULT 'USD',
			status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'failed', 'refunded')),
			type VARCHAR(50) DEFAULT 'subscription',
			invoice_id VARCHAR(255),
			stripe_payment_intent_id VARCHAR(255),
			stripe_charge_id VARCHAR(255),
			period_start TIMESTAMP WITH TIME ZONE,
			period_end TIMESTAMP WITH TIME ZONE,
			description TEXT,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			paid_at TIMESTAMP WITH TIME ZONE
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_billing_records_customer ON billing_records(customer_id);`,
		`CREATE INDEX IF NOT EXISTS idx_billing_records_subscription ON billing_records(subscription_id);`,
		`CREATE INDEX IF NOT EXISTS idx_billing_records_status ON billing_records(status);`,
		`CREATE INDEX IF NOT EXISTS idx_billing_records_created ON billing_records(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_billing_records_invoice ON billing_records(invoice_id);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// migrate008_createLicenses creates the licenses table
func migrate008_createLicenses(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS licenses (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			license_key VARCHAR(255) UNIQUE NOT NULL,
			license_type VARCHAR(50) DEFAULT 'standard' CHECK (license_type IN ('trial', 'standard', 'enterprise')),
			status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'expired', 'revoked', 'suspended')),
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			max_instances INTEGER DEFAULT 1,
			features JSONB DEFAULT '{}',
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_licenses_customer ON licenses(customer_id);`,
		`CREATE INDEX IF NOT EXISTS idx_licenses_status ON licenses(status);`,
		`CREATE INDEX IF NOT EXISTS idx_licenses_expires ON licenses(expires_at);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// migrate009_createAdminUsers creates the admin_users table
func migrate009_createAdminUsers(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS admin_users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			display_name VARCHAR(100),
			role VARCHAR(50) DEFAULT 'admin' CHECK (role IN ('super_admin', 'admin', 'moderator', 'support')),
			status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended')),
			permissions JSONB DEFAULT '[]',
			last_login_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_admin_users_status ON admin_users(status);`,
		`CREATE INDEX IF NOT EXISTS idx_admin_users_role ON admin_users(role);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// migrate010_createPlatformSettings creates the platform_settings table
func migrate010_createPlatformSettings(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS platform_settings (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			key VARCHAR(100) UNIQUE NOT NULL,
			value TEXT NOT NULL,
			description TEXT,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	return nil
}
