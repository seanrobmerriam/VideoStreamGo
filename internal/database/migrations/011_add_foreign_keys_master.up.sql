-- Migration: 011_add_foreign_keys_master
-- Purpose: Add foreign key constraints to master database tables for data integrity
-- Created: 2025-01-26

-- UP Migration: Add foreign key constraints

-- Add FK constraint for instances.customer_id -> customers(id)
-- Using ON DELETE RESTRICT to prevent deletion of customers with active instances
ALTER TABLE instances 
DROP CONSTRAINT IF EXISTS fk_instances_customer;

ALTER TABLE instances 
ADD CONSTRAINT fk_instances_customer 
FOREIGN KEY (customer_id) 
REFERENCES customers(id) 
ON DELETE RESTRICT;

-- Add FK constraint for instances.plan_id -> subscription_plans(id)
ALTER TABLE instances 
DROP CONSTRAINT IF EXISTS fk_instances_plan;

ALTER TABLE instances 
ADD CONSTRAINT fk_instances_plan 
FOREIGN KEY (plan_id) 
REFERENCES subscription_plans(id) 
ON DELETE SET NULL;

-- Add FK constraint for subscriptions.customer_id -> customers(id)
ALTER TABLE subscriptions 
DROP CONSTRAINT IF EXISTS fk_subscriptions_customer;

ALTER TABLE subscriptions 
ADD CONSTRAINT fk_subscriptions_customer 
FOREIGN KEY (customer_id) 
REFERENCES customers(id) 
ON DELETE RESTRICT;

-- Add FK constraint for subscriptions.plan_id -> subscription_plans(id)
ALTER TABLE subscriptions 
DROP CONSTRAINT IF EXISTS fk_subscriptions_plan;

ALTER TABLE subscriptions 
ADD CONSTRAINT fk_subscriptions_plan 
FOREIGN KEY (plan_id) 
REFERENCES subscription_plans(id) 
ON DELETE RESTRICT;

-- Add FK constraint for usage_metrics.instance_id -> instances(id)
-- Using ON DELETE CASCADE since metrics are owned by instances
ALTER TABLE usage_metrics 
DROP CONSTRAINT IF EXISTS fk_usage_metrics_instance;

ALTER TABLE usage_metrics 
ADD CONSTRAINT fk_usage_metrics_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Add FK constraint for instance_config.instance_id -> instances(id)
ALTER TABLE instance_config 
DROP CONSTRAINT IF EXISTS fk_instance_config_instance;

ALTER TABLE instance_config 
ADD CONSTRAINT fk_instance_config_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Add FK constraint for billing_records.customer_id -> customers(id)
ALTER TABLE billing_records 
DROP CONSTRAINT IF EXISTS fk_billing_records_customer;

ALTER TABLE billing_records 
ADD CONSTRAINT fk_billing_records_customer 
FOREIGN KEY (customer_id) 
REFERENCES customers(id) 
ON DELETE RESTRICT;

-- Add FK constraint for billing_records.subscription_id -> subscriptions(id)
ALTER TABLE billing_records 
DROP CONSTRAINT IF EXISTS fk_billing_records_subscription;

ALTER TABLE billing_records 
ADD CONSTRAINT fk_billing_records_subscription 
FOREIGN KEY (subscription_id) 
REFERENCES subscriptions(id) 
ON DELETE SET NULL;

-- Add FK constraint for licenses.customer_id -> customers(id)
ALTER TABLE licenses 
DROP CONSTRAINT IF EXISTS fk_licenses_customer;

ALTER TABLE licenses 
ADD CONSTRAINT fk_licenses_customer 
FOREIGN KEY (customer_id) 
REFERENCES customers(id) 
ON DELETE CASCADE;

-- Create index on foreign key columns for better query performance
CREATE INDEX IF NOT EXISTS idx_instances_customer_id ON instances(customer_id);
CREATE INDEX IF NOT EXISTS idx_instances_plan_id ON instances(plan_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_customer_id ON subscriptions(customer_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_plan_id ON subscriptions(plan_id);
CREATE INDEX IF NOT EXISTS idx_usage_metrics_instance_id ON usage_metrics(instance_id);
CREATE INDEX IF NOT EXISTS idx_instance_config_instance_id ON instance_config(instance_id);
CREATE INDEX IF NOT EXISTS idx_billing_records_customer_id ON billing_records(customer_id);
CREATE INDEX IF NOT EXISTS idx_billing_records_subscription_id ON billing_records(subscription_id);
CREATE INDEX IF NOT EXISTS idx_licenses_customer_id ON licenses(customer_id);
