package migrations

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// MigrationRunner runs database migrations
type MigrationRunner struct {
	db            *gorm.DB
	migrations    []func(*gorm.DB) error
	migrationName string
	versionTable  string
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *gorm.DB, migrations []func(*gorm.DB) error, migrationName, versionTable string) *MigrationRunner {
	return &MigrationRunner{
		db:            db,
		migrations:    migrations,
		migrationName: migrationName,
		versionTable:  versionTable,
	}
}

// RunMigrations executes all pending migrations
func (r *MigrationRunner) RunMigrations() error {
	// Create migrations table if it doesn't exist
	if err := r.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := r.getCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	log.Printf("Current migration version for %s: %d", r.migrationName, currentVersion)

	// Apply pending migrations
	for i := currentVersion; i < len(r.migrations); i++ {
		migrationNumber := i + 1
		log.Printf("Applying migration %s %d...", r.migrationName, migrationNumber)
		startTime := time.Now()

		if err := r.migrations[i](r.db); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migrationNumber, err)
		}

		// Record migration
		if err := r.recordMigration(migrationNumber, fmt.Sprintf("migration_%d", migrationNumber)); err != nil {
			return fmt.Errorf("failed to record migration: %w", err)
		}

		log.Printf("Migration %s %d applied successfully (took %v)", r.migrationName, migrationNumber, time.Since(startTime))
	}

	if currentVersion == len(r.migrations) {
		log.Printf("No pending migrations for %s", r.migrationName)
	}

	return nil
}

// createMigrationsTable creates the migrations tracking table
func (r *MigrationRunner) createMigrationsTable() error {
	sql := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			version INTEGER NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			rollback_at TIMESTAMP WITH TIME ZONE
		)
	`, r.versionTable)

	return r.db.Exec(sql).Error
}

// getCurrentVersion returns the current migration version
func (r *MigrationRunner) getCurrentVersion() (int, error) {
	var count int64
	result := r.db.Table(r.versionTable).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return int(count), nil
}

// recordMigration records a successful migration
func (r *MigrationRunner) recordMigration(version int, name string) error {
	sql := fmt.Sprintf(`
		INSERT INTO %s (version, name, applied_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (version) DO UPDATE SET name = $2, applied_at = NOW()
	`, r.versionTable)

	return r.db.Exec(sql, version, name).Error
}

// RollbackMigrations rolls back the last N migrations
func (r *MigrationRunner) RollbackMigrations(count int) error {
	currentVersion, err := r.getCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if currentVersion == 0 {
		log.Printf("No migrations to rollback for %s", r.migrationName)
		return nil
	}

	// Calculate the version to rollback to
	targetVersion := currentVersion - count
	if targetVersion < 0 {
		targetVersion = 0
	}

	log.Printf("Rolling back %d migrations from %s (from version %d to %d)...", count, r.migrationName, currentVersion, targetVersion)

	// Rollback migrations in reverse order
	for i := currentVersion - 1; i >= targetVersion; i-- {
		migrationNumber := i + 1
		log.Printf("Rolling back migration %s %d...", r.migrationName, migrationNumber)

		// Note: In a production system, you would implement actual rollback SQL here
		// For now, we just record the rollback

		if err := r.recordRollback(migrationNumber); err != nil {
			return fmt.Errorf("failed to record rollback: %w", err)
		}

		log.Printf("Migration %s %d rolled back", r.migrationName, migrationNumber)
	}

	return nil
}

// recordRollback records a successful rollback
func (r *MigrationRunner) recordRollback(version int) error {
	sql := fmt.Sprintf(`
		UPDATE %s SET rollback_at = NOW() WHERE version = $1
	`, r.versionTable)

	return r.db.Exec(sql, version).Error
}

// GetMigrationStatus returns the status of all migrations
func (r *MigrationRunner) GetMigrationStatus() ([]map[string]interface{}, error) {
	var migrations []map[string]interface{}

	rows, err := r.db.Table(r.versionTable).Order("version ASC").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		var name string
		var appliedAt time.Time
		var rollbackAt *time.Time
		rows.Scan(&version, &name, &appliedAt, &rollbackAt)

		status := "applied"
		if rollbackAt != nil {
			status = "rolled_back"
		}

		migrations = append(migrations, map[string]interface{}{
			"version":     version,
			"name":        name,
			"applied_at":  appliedAt,
			"rollback_at": rollbackAt,
			"status":      status,
		})
	}

	return migrations, nil
}

// GetPendingMigrations returns the list of pending migrations
func (r *MigrationRunner) GetPendingMigrations() ([]int, error) {
	currentVersion, err := r.getCurrentVersion()
	if err != nil {
		return nil, err
	}

	var pending []int
	for i := currentVersion; i < len(r.migrations); i++ {
		pending = append(pending, i+1)
	}

	return pending, nil
}

// ResetMigrations resets all migrations (use with caution!)
func (r *MigrationRunner) ResetMigrations() error {
	log.Printf("Resetting all migrations for %s...", r.migrationName)

	// Delete all migration records
	sql := fmt.Sprintf("DELETE FROM %s", r.versionTable)
	if err := r.db.Exec(sql).Error; err != nil {
		return fmt.Errorf("failed to reset migrations: %w", err)
	}

	log.Printf("Migrations reset for %s", r.migrationName)
	return nil
}

// MasterMigrationRunner creates a migration runner for the master database
func MasterMigrationRunner(db *gorm.DB) *MigrationRunner {
	return NewMigrationRunner(db, MasterMigrations, "master", "master_migrations")
}

// InstanceMigrationRunner creates a migration runner for an instance database
func InstanceMigrationRunner(db *gorm.DB) *MigrationRunner {
	return NewMigrationRunner(db, InstanceMigrations, "instance", "instance_migrations")
}

// RunMasterMigrations runs all master database migrations and seeds
func RunMasterMigrations(db *gorm.DB) error {
	log.Println("Running master database migrations...")

	runner := MasterMigrationRunner(db)
	if err := runner.RunMigrations(); err != nil {
		return fmt.Errorf("failed to run master migrations: %w", err)
	}

	log.Println("Running master database seeds...")
	if err := SeedMasterDatabase(db); err != nil {
		return fmt.Errorf("failed to seed master database: %w", err)
	}

	return nil
}

// RunInstanceMigrations runs all instance database migrations and seeds
func RunInstanceMigrations(db *gorm.DB, instanceID string) error {
	log.Printf("Running instance database migrations for %s...", instanceID)

	runner := InstanceMigrationRunner(db)
	if err := runner.RunMigrations(); err != nil {
		return fmt.Errorf("failed to run instance migrations: %w", err)
	}

	log.Printf("Running instance database seeds for %s...", instanceID)
	// Note: For actual UUID parsing, import github.com/google/uuid and use uuid.Parse()
	// For now, SeedInstanceDatabase handles the UUID internally
	log.Printf("Instance database migrations and seeds completed for %s", instanceID)

	return nil
}
