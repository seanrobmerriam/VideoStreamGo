-- Migration: 011_add_foreign_keys_master (DOWN)
-- Purpose: Remove foreign key constraints from master database tables
-- Created: 2025-01-26

-- DOWN Migration: Remove foreign key constraints

-- Remove FK constraint for licenses.customer_id
ALTER TABLE licenses 
DROP CONSTRAINT IF EXISTS fk_licenses_customer;

-- Remove FK constraint for billing_records.subscription_id
ALTER TABLE billing_records 
DROP CONSTRAINT IF EXISTS fk_billing_records_subscription;

-- Remove FK constraint for billing_records.customer_id
ALTER TABLE billing_records 
DROP CONSTRAINT IF EXISTS fk_billing_records_customer;

-- Remove FK constraint for instance_config.instance_id
ALTER TABLE instance_config 
DROP CONSTRAINT IF EXISTS fk_instance_config_instance;

-- Remove FK constraint for usage_metrics.instance_id
ALTER TABLE usage_metrics 
DROP CONSTRAINT IF EXISTS fk_usage_metrics_instance;

-- Remove FK constraint for subscriptions.plan_id
ALTER TABLE subscriptions 
DROP CONSTRAINT IF EXISTS fk_subscriptions_plan;

-- Remove FK constraint for subscriptions.customer_id
ALTER TABLE subscriptions 
DROP CONSTRAINT IF EXISTS fk_subscriptions_customer;

-- Remove FK constraint for instances.plan_id
ALTER TABLE instances 
DROP CONSTRAINT IF EXISTS fk_instances_plan;

-- Remove FK constraint for instances.customer_id
ALTER TABLE instances 
DROP CONSTRAINT IF EXISTS fk_instances_customer;

-- Drop indexes on foreign key columns
DROP INDEX IF EXISTS idx_licenses_customer_id;
DROP INDEX IF EXISTS idx_billing_records_subscription_id;
DROP INDEX IF EXISTS idx_billing_records_customer_id;
DROP INDEX IF EXISTS idx_instance_config_instance_id;
DROP INDEX IF EXISTS idx_usage_metrics_instance_id;
DROP INDEX IF EXISTS idx_subscriptions_plan_id;
DROP INDEX IF EXISTS idx_subscriptions_customer_id;
DROP INDEX IF EXISTS idx_instances_plan_id;
DROP INDEX IF EXISTS idx_instances_customer_id;
