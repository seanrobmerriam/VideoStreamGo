package migrations

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InstanceMigrations contains all instance database migrations
var InstanceMigrations = []func(*gorm.DB) error{
	instanceMigrate001_createUsers,
	instanceMigrate002_createVideos,
	instanceMigrate003_createCategories,
	instanceMigrate004_createTags,
	instanceMigrate005_createVideoTags,
	instanceMigrate006_createComments,
	instanceMigrate007_createRatings,
	instanceMigrate008_createFavorites,
	instanceMigrate009_createPlaylists,
	instanceMigrate010_createPlaylistVideos,
	instanceMigrate011_createVideoViews,
	instanceMigrate012_createUserSessions,
	instanceMigrate013_createBrandingConfig,
	instanceMigrate014_createPages,
	instanceMigrate015_createSettings,
}

// instanceMigrate001_createUsers creates the users table
func instanceMigrate001_createUsers(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL,
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
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			deleted_at TIMESTAMP WITH TIME ZONE
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_users_instance ON users(instance_id);`,
		`CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);`,
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);`,
		`CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NULL;`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// instanceMigrate002_createVideos creates the videos table
func instanceMigrate002_createVideos(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS videos (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL,
			title VARCHAR(255) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			description TEXT,
			user_id UUID NOT NULL REFERENCES users(id),
			category_id UUID REFERENCES categories(id),
			status VARCHAR(50) DEFAULT 'processing' CHECK (status IN ('pending', 'processing', 'transcoding', 'ready', 'active', 'hidden', 'failed', 'deleted')),
			video_url VARCHAR(500) NOT NULL,
			thumbnail_url VARCHAR(500),
			hls_path VARCHAR(500),
			dash_path VARCHAR(500),
			duration DOUBLE PRECISION DEFAULT 0,
			file_size BIGINT DEFAULT 0,
			resolution VARCHAR(20),
			resolution_label VARCHAR(20),
			bitrate INTEGER DEFAULT 0,
			codec VARCHAR(50),
			audio_codec VARCHAR(50),
			frame_rate DOUBLE PRECISION DEFAULT 0,
			processing_status VARCHAR(50) DEFAULT 'pending' CHECK (processing_status IN ('pending', 'uploading', 'uploaded', 'extracting_metadata', 'transcoding', 'generating_thumbnails', 'completed', 'failed')),
			processing_progress INTEGER DEFAULT 0,
			processing_error TEXT,
			view_count BIGINT DEFAULT 0,
			like_count INTEGER DEFAULT 0,
			dislike_count INTEGER DEFAULT 0,
			comment_count INTEGER DEFAULT 0,
			is_featured BOOLEAN DEFAULT false,
			is_public BOOLEAN DEFAULT true,
			published_at TIMESTAMP WITH TIME ZONE,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			deleted_at TIMESTAMP WITH TIME ZONE
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes as specified in ARCHITECTURE.md Section 4.3
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_videos_instance ON videos(instance_id);`,
		`CREATE INDEX IF NOT EXISTS idx_videos_user ON videos(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_videos_category ON videos(category_id);`,
		`CREATE INDEX IF NOT EXISTS idx_videos_status ON videos(status);`,
		`CREATE INDEX IF NOT EXISTS idx_videos_created ON videos(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_videos_view_count ON videos(view_count DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_videos_slug ON videos(slug);`,
		`CREATE INDEX IF NOT EXISTS idx_videos_featured ON videos(is_featured) WHERE is_featured = true;`,
		`CREATE INDEX IF NOT EXISTS idx_videos_public_status ON videos(is_public, status) WHERE is_public = true AND status = 'active';`,
		`CREATE INDEX IF NOT EXISTS idx_videos_deleted_at ON videos(deleted_at) WHERE deleted_at IS NULL;`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// instanceMigrate003_createCategories creates the categories table
func instanceMigrate003_createCategories(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS categories (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL,
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
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_categories_instance ON categories(instance_id);`,
		`CREATE INDEX IF NOT EXISTS idx_categories_parent ON categories(parent_id);`,
		`CREATE INDEX IF NOT EXISTS idx_categories_sort_order ON categories(sort_order);`,
		`CREATE INDEX IF NOT EXISTS idx_categories_active ON categories(is_active);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// instanceMigrate004_createTags creates the tags table
func instanceMigrate004_createTags(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS tags (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL,
			name VARCHAR(100) NOT NULL,
			slug VARCHAR(100) UNIQUE NOT NULL,
			usage_count INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_tags_instance ON tags(instance_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tags_slug ON tags(slug);`,
		`CREATE INDEX IF NOT EXISTS idx_tags_usage_count ON tags(usage_count DESC);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// instanceMigrate005_createVideoTags creates the video_tags junction table
func instanceMigrate005_createVideoTags(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS video_tags (
			video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
			tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			PRIMARY KEY (video_id, tag_id)
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_video_tags_tag ON video_tags(tag_id);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// instanceMigrate006_createComments creates the comments table
func instanceMigrate006_createComments(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS comments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL,
			video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id),
			parent_id UUID REFERENCES comments(id),
			content TEXT NOT NULL,
			is_edited BOOLEAN DEFAULT false,
			is_deleted BOOLEAN DEFAULT false,
			like_count INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			deleted_at TIMESTAMP WITH TIME ZONE
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes as specified in ARCHITECTURE.md Section 4.3
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_comments_instance ON comments(instance_id);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_video ON comments(video_id, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_user ON comments(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// instanceMigrate007_createRatings creates the ratings table
func instanceMigrate007_createRatings(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS ratings (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL,
			video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id),
			rating SMALLINT NOT NULL CHECK (rating IN (-1, 1)),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(video_id, user_id)
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes as specified in ARCHITECTURE.md Section 4.3
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_ratings_instance ON ratings(instance_id);`,
		`CREATE INDEX IF NOT EXISTS idx_ratings_video ON ratings(video_id);`,
		`CREATE INDEX IF NOT EXISTS idx_ratings_user ON ratings(user_id);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// instanceMigrate008_createFavorites creates the favorites table
func instanceMigrate008_createFavorites(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS favorites (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(user_id, video_id)
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes as specified in ARCHITECTURE.md Section 4.3
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_favorites_instance ON favorites(instance_id);`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_user ON favorites(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_video ON favorites(video_id);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// instanceMigrate009_createPlaylists creates the playlists table
func instanceMigrate009_createPlaylists(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS playlists (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			is_public BOOLEAN DEFAULT true,
			view_count BIGINT DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_playlists_instance ON playlists(instance_id);`,
		`CREATE INDEX IF NOT EXISTS idx_playlists_user ON playlists(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_playlists_public ON playlists(is_public);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// instanceMigrate010_createPlaylistVideos creates the playlist_videos junction table
func instanceMigrate010_createPlaylistVideos(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS playlist_videos (
			playlist_id UUID NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
			video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
			position INTEGER NOT NULL,
			added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			PRIMARY KEY (playlist_id, video_id)
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes as specified in ARCHITECTURE.md Section 4.3
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_playlist_videos_playlist ON playlist_videos(playlist_id, position);`,
		`CREATE INDEX IF NOT EXISTS idx_playlist_videos_video ON playlist_videos(video_id);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// instanceMigrate011_createVideoViews creates the video_views table
func instanceMigrate011_createVideoViews(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS video_views (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL,
			video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
			user_id UUID REFERENCES users(id),
			ip_address INET,
			user_agent TEXT,
			referrer TEXT,
			country_code CHAR(2),
			watch_duration INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes as specified in ARCHITECTURE.md Section 4.3
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_video_views_instance ON video_views(instance_id);`,
		`CREATE INDEX IF NOT EXISTS idx_video_views_video ON video_views(video_id, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_video_views_created ON video_views(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_video_views_user ON video_views(user_id);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// instanceMigrate012_createUserSessions creates the user_sessions table
func instanceMigrate012_createUserSessions(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS user_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL,
			ip_address INET,
			user_agent TEXT,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes as specified in ARCHITECTURE.md Section 4.3
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_token ON user_sessions(token_hash);`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_expires ON user_sessions(expires_at);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// instanceMigrate013_createBrandingConfig creates the branding_config table
func instanceMigrate013_createBrandingConfig(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS branding_config (
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
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add index
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_branding_config_instance ON branding_config(instance_id);`).Error; err != nil {
		return err
	}

	return nil
}

// instanceMigrate014_createPages creates the pages table
func instanceMigrate014_createPages(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS pages (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL,
			title VARCHAR(255) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			content TEXT NOT NULL,
			excerpt VARCHAR(500),
			page_type VARCHAR(50) DEFAULT 'custom' CHECK (page_type IN ('static', 'custom', 'legal', 'error')),
			template VARCHAR(100),
			meta_title VARCHAR(255),
			meta_description VARCHAR(500),
			featured_image VARCHAR(500),
			is_published BOOLEAN DEFAULT true,
			show_in_menu BOOLEAN DEFAULT false,
			menu_order INTEGER DEFAULT 0,
			created_by UUID,
			updated_by UUID,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_pages_instance ON pages(instance_id);`,
		`CREATE INDEX IF NOT EXISTS idx_pages_published ON pages(is_published);`,
		`CREATE INDEX IF NOT EXISTS idx_pages_type ON pages(page_type);`,
		`CREATE INDEX IF NOT EXISTS idx_pages_menu ON pages(show_in_menu, menu_order);`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// instanceMigrate015_createSettings creates the settings table
func instanceMigrate015_createSettings(db *gorm.DB) error {
	sql := `
		CREATE TABLE IF NOT EXISTS settings (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			instance_id UUID NOT NULL,
			key VARCHAR(100) NOT NULL,
			value TEXT,
			description TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(instance_id, key)
		);
	`
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	// Add index
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_settings_instance ON settings(instance_id);`).Error; err != nil {
		return err
	}

	return nil
}

// MigrationVersion tracks which migrations have been applied
type MigrationVersion struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
	Version   int       `gorm:"not null" json:"version"`
	AppliedAt time.Time `gorm:"autoCreateTime" json:"applied_at"`
}

// TableName sets the table name for MigrationVersion
func (MigrationVersion) TableName() string {
	return "migration_versions"
}
