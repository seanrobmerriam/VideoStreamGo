package platform

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"videostreamgo/internal/config"
)

// DatabaseStats holds database statistics
type DatabaseStats struct {
	SizeBytes       int64      `json:"size_bytes"`
	ConnectionCount int        `json:"connection_count"`
	TableCount      int        `json:"table_count"`
	IndexCount      int        `json:"index_count"`
	LastVacuum      *time.Time `json:"last_vacuum,omitempty"`
	LastAnalyze     *time.Time `json:"last_analyze,omitempty"`
}

// DBProvisioner handles database provisioning for tenant instances
type DBProvisioner struct {
	masterDB *gorm.DB
	config   *config.Config
	dsn      string
}

// NewDBProvisioner creates a new database provisioner
func NewDBProvisioner(cfg *config.Config) (*DBProvisioner, error) {
	// Connect to master database to create new databases
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.MasterDB.Host,
		cfg.MasterDB.Port,
		cfg.MasterDB.Username,
		cfg.MasterDB.Password,
		"postgres", // Connect to default postgres database to create new databases
		cfg.MasterDB.SSLMode,
	)

	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: false,
		},
		Logger: logger.Default.LogMode(logger.Warn),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to master database: %w", err)
	}

	return &DBProvisioner{
		masterDB: db,
		config:   cfg,
		dsn:      dsn,
	}, nil
}

// CreateDatabase creates a new PostgreSQL database for a tenant
func (p *DBProvisioner) CreateDatabase(dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if database already exists
	var count int64
	err := p.masterDB.WithContext(ctx).
		Table("pg_database").
		Where("datname = ?", dbName).
		Count(&count).Error

	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("database %s already exists", dbName)
	}

	// Create the database
	createSQL := fmt.Sprintf("CREATE DATABASE %s", p.quoteIdentifier(dbName))
	if err := p.masterDB.WithContext(ctx).Exec(createSQL).Error; err != nil {
		// Check if it's a duplicate database error
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("database %s already exists", dbName)
		}
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Grant all privileges to the application user
	grantSQL := fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s",
		p.quoteIdentifier(dbName),
		p.quoteIdentifier(p.config.MasterDB.Username),
	)
	if err := p.masterDB.WithContext(ctx).Exec(grantSQL).Error; err != nil {
		// Log but don't fail - the user might already have privileges
		fmt.Printf("Warning: failed to grant privileges: %v\n", err)
	}

	return nil
}

// CreateUser creates a new database user with access to a specific database
func (p *DBProvisioner) CreateUser(username, password, dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if user already exists
	var userExists int64
	err := p.masterDB.WithContext(ctx).
		Table("pg_roles").
		Where("rolname = ?", username).
		Count(&userExists).Error

	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if userExists == 0 {
		// Create the user
		createUserSQL := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'",
			p.quoteIdentifier(username),
			p.escapeString(password),
		)
		if err := p.masterDB.WithContext(ctx).Exec(createUserSQL).Error; err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
	}

	// Grant CONNECT privilege on the database
	grantSQL := fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s",
		p.quoteIdentifier(dbName),
		p.quoteIdentifier(username),
	)
	if err := p.masterDB.WithContext(ctx).Exec(grantSQL).Error; err != nil {
		return fmt.Errorf("failed to grant CONNECT privilege: %w", err)
	}

	return nil
}

// GrantPermissions sets up access permissions for a user on a database
func (p *DBProvisioner) GrantPermissions(username, dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to the specific database to grant schema permissions
	db, err := p.connectToDatabase(dbName)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Grant USAGE on public schema
	grantUsageSQL := fmt.Sprintf("GRANT USAGE ON SCHEMA public TO %s",
		p.quoteIdentifier(username),
	)
	if err := db.WithContext(ctx).Exec(grantUsageSQL).Error; err != nil {
		return fmt.Errorf("failed to grant USAGE: %w", err)
	}

	// Grant all privileges on all tables in public schema
	grantTablesSQL := fmt.Sprintf("GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO %s",
		p.quoteIdentifier(username),
	)
	if err := db.WithContext(ctx).Exec(grantTablesSQL).Error; err != nil {
		return fmt.Errorf("failed to grant table privileges: %w", err)
	}

	// Grant all privileges on all sequences in public schema
	grantSeqSQL := fmt.Sprintf("GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO %s",
		p.quoteIdentifier(username),
	)
	if err := db.WithContext(ctx).Exec(grantSeqSQL).Error; err != nil {
		return fmt.Errorf("failed to grant sequence privileges: %w", err)
	}

	// Set default privileges for future tables
	alterDefaultSQL := fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO %s",
		p.quoteIdentifier(username),
	)
	if err := db.WithContext(ctx).Exec(alterDefaultSQL).Error; err != nil {
		return fmt.Errorf("failed to alter default privileges: %w", err)
	}

	alterDefaultSeqSQL := fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO %s",
		p.quoteIdentifier(username),
	)
	if err := db.WithContext(ctx).Exec(alterDefaultSeqSQL).Error; err != nil {
		return fmt.Errorf("failed to alter default sequence privileges: %w", err)
	}

	return nil
}

// RunMigrations runs the database migrations for an instance
func (p *DBProvisioner) RunMigrations(dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db, err := p.connectToDatabase(dbName)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run instance database migrations
	migrations := p.getInstanceMigrations()

	for _, migration := range migrations {
		if err := db.WithContext(ctx).Exec(migration).Error; err != nil {
			return fmt.Errorf("failed to run migration: %w", err)
		}
	}

	return nil
}

// getInstanceMigrations returns SQL migrations for the instance database
func (p *DBProvisioner) getInstanceMigrations() []string {
	return []string{
		// Create extensions
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,

		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			display_name VARCHAR(100),
			avatar_url VARCHAR(500),
			bio TEXT,
			role VARCHAR(50) DEFAULT 'user' CHECK (role IN ('user', 'moderator', 'admin')),
			status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'banned', 'suspended')),
			email_verified BOOLEAN DEFAULT false,
			last_login_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			metadata JSONB DEFAULT '{}'
		)`,

		// Categories table
		`CREATE TABLE IF NOT EXISTS categories (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			slug VARCHAR(100) UNIQUE NOT NULL,
			description TEXT,
			parent_id UUID REFERENCES categories(id),
			icon_url VARCHAR(500),
			color VARCHAR(7),
			sort_order INTEGER DEFAULT 0,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Videos table
		`CREATE TABLE IF NOT EXISTS videos (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title VARCHAR(255) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			description TEXT,
			user_id UUID NOT NULL REFERENCES users(id),
			category_id UUID REFERENCES categories(id),
			status VARCHAR(50) DEFAULT 'processing' CHECK (status IN ('processing', 'active', 'hidden', 'deleted')),
			video_url VARCHAR(500) NOT NULL,
			thumbnail_url VARCHAR(500),
			duration INTEGER,
			file_size BIGINT,
			resolution VARCHAR(20),
			view_count BIGINT DEFAULT 0,
			like_count INTEGER DEFAULT 0,
			dislike_count INTEGER DEFAULT 0,
			comment_count INTEGER DEFAULT 0,
			is_featured BOOLEAN DEFAULT false,
			is_public BOOLEAN DEFAULT true,
			published_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			metadata JSONB DEFAULT '{}'
		)`,

		// Tags table
		`CREATE TABLE IF NOT EXISTS tags (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			slug VARCHAR(100) UNIQUE NOT NULL,
			usage_count INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Video tags junction table
		`CREATE TABLE IF NOT EXISTS video_tags (
			video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
			tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			PRIMARY KEY (video_id, tag_id)
		)`,

		// Comments table
		`CREATE TABLE IF NOT EXISTS comments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id),
			parent_id UUID REFERENCES comments(id),
			content TEXT NOT NULL,
			is_edited BOOLEAN DEFAULT false,
			is_deleted BOOLEAN DEFAULT false,
			like_count INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Ratings table
		`CREATE TABLE IF NOT EXISTS ratings (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id),
			rating SMALLINT NOT NULL CHECK (rating IN (-1, 1)),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(video_id, user_id)
		)`,

		// Favorites table
		`CREATE TABLE IF NOT EXISTS favorites (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(user_id, video_id)
		)`,

		// Playlists table
		`CREATE TABLE IF NOT EXISTS playlists (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			is_public BOOLEAN DEFAULT true,
			view_count BIGINT DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Playlist videos junction table
		`CREATE TABLE IF NOT EXISTS playlist_videos (
			playlist_id UUID NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
			video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
			position INTEGER NOT NULL,
			added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			PRIMARY KEY (playlist_id, video_id)
		)`,

		// Video views table
		`CREATE TABLE IF NOT EXISTS video_views (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
			user_id UUID REFERENCES users(id),
			ip_address INET,
			user_agent TEXT,
			referrer TEXT,
			country_code CHAR(2),
			watch_duration INTEGER,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// User sessions table
		`CREATE TABLE IF NOT EXISTS user_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL,
			ip_address INET,
			user_agent TEXT,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Branding configuration table
		`CREATE TABLE IF NOT EXISTS branding_config (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL,
			site_name VARCHAR(255) DEFAULT 'VideoTube',
			logo_url VARCHAR(500),
			favicon_url VARCHAR(500),
			primary_color VARCHAR(7) DEFAULT '#2563eb',
			secondary_color VARCHAR(7) DEFAULT '#64748b',
			accent_color VARCHAR(7) DEFAULT '#f59e0b',
			background_color VARCHAR(7) DEFAULT '#ffffff',
			text_color VARCHAR(7) DEFAULT '#1e293b',
			header_html TEXT,
			footer_html TEXT,
			custom_css TEXT,
			social_links JSONB DEFAULT '{}',
			footer_links JSONB DEFAULT '[]',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Create indexes
		`CREATE INDEX IF NOT EXISTS idx_videos_user ON videos(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_videos_category ON videos(category_id)`,
		`CREATE INDEX IF NOT EXISTS idx_videos_status ON videos(status)`,
		`CREATE INDEX IF NOT EXISTS idx_videos_created ON videos(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_videos_view_count ON videos(view_count DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_videos_slug ON videos(slug)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_video ON comments(video_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_user ON comments(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ratings_video ON ratings(video_id)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_user ON favorites(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_video_views_video ON video_views(video_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_video_views_created ON video_views(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_token ON user_sessions(token_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_playlist_videos_playlist ON playlist_videos(playlist_id, position)`,
		`CREATE INDEX IF NOT EXISTS idx_tags_slug ON tags(slug)`,
		`CREATE INDEX IF NOT EXISTS idx_video_tags_tag ON video_tags(tag_id)`,
		`CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug)`,
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
	}
}

// DeleteDatabase removes a tenant database
func (p *DBProvisioner) DeleteDatabase(dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Terminate all connections to the database first
	terminateSQL := fmt.Sprintf(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = %s AND pid <> pg_backend_pid()",
		p.quoteLiteral(dbName),
	)
	if err := p.masterDB.WithContext(ctx).Exec(terminateSQL).Error; err != nil {
		// Log but continue - database might not have active connections
		fmt.Printf("Warning: failed to terminate connections: %v\n", err)
	}

	// Drop the database
	dropSQL := fmt.Sprintf("DROP DATABASE IF EXISTS %s", p.quoteIdentifier(dbName))
	if err := p.masterDB.WithContext(ctx).Exec(dropSQL).Error; err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}

	return nil
}

// GetDatabaseStats returns statistics about a database
func (p *DBProvisioner) GetDatabaseStats(dbName string) (*DatabaseStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := p.connectToDatabase(dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	stats := &DatabaseStats{}

	// Get database size
	var sizeBytes int64
	err = db.WithContext(ctx).Raw("SELECT pg_database_size(?)", dbName).Scan(&sizeBytes).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get database size: %w", err)
	}
	stats.SizeBytes = sizeBytes

	// Get connection count
	var connectionCount int
	err = db.WithContext(ctx).Raw("SELECT count(*) FROM pg_stat_activity WHERE datname = ?", dbName).Scan(&connectionCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get connection count: %w", err)
	}
	stats.ConnectionCount = connectionCount

	// Get table count
	var tableCount int
	err = db.WithContext(ctx).Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tableCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get table count: %w", err)
	}
	stats.TableCount = tableCount

	// Get index count
	var indexCount int
	err = db.WithContext(ctx).Raw("SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public'").Scan(&indexCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get index count: %w", err)
	}
	stats.IndexCount = indexCount

	return stats, nil
}

// GenerateSecurePassword generates a cryptographically secure password
func GenerateSecurePassword(length int) (string, error) {
	if length < 16 {
		length = 32
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

// GenerateInstanceCredentials generates database credentials for a new instance
func GenerateInstanceCredentials(instanceID uuid.UUID) (username, password, dbName string) {
	shortID := instanceID.String()[:8]
	dbName = fmt.Sprintf("instance_%s", shortID)
	username = fmt.Sprintf("vsgo_%s", shortID)

	password, _ = GenerateSecurePassword(32)

	return
}

// connectToDatabase establishes a connection to a specific database
func (p *DBProvisioner) connectToDatabase(dbName string) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.config.MasterDB.Host,
		p.config.MasterDB.Port,
		p.config.MasterDB.Username,
		p.config.MasterDB.Password,
		dbName,
		p.config.MasterDB.SSLMode,
	)

	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: false,
		},
		Logger: logger.Default.LogMode(logger.Warn),
	}

	return gorm.Open(postgres.Open(dsn), gormConfig)
}

// quoteIdentifier quotes a PostgreSQL identifier
func (p *DBProvisioner) quoteIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(identifier, `"`, `""`))
}

// quoteLiteral quotes a PostgreSQL string literal
func (p *DBProvisioner) quoteLiteral(literal string) string {
	return fmt.Sprintf(`'%s'`, strings.ReplaceAll(literal, `'`, `''`))
}

// escapeString escapes a string for use in SQL
func (p *DBProvisioner) escapeString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
