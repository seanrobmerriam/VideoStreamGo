-- Migration: 016_add_foreign_keys_instance
-- Purpose: Add foreign key constraints to instance database tables for data integrity
-- Created: 2025-01-26

-- UP Migration: Add foreign key constraints

-- Add FK constraint for users.instance_id -> instances(id)
-- Using ON DELETE CASCADE since users are owned by instances
ALTER TABLE users 
DROP CONSTRAINT IF EXISTS fk_users_instance;

ALTER TABLE users 
ADD CONSTRAINT fk_users_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Add FK constraint for videos.instance_id -> instances(id)
ALTER TABLE videos 
DROP CONSTRAINT IF EXISTS fk_videos_instance;

ALTER TABLE videos 
ADD CONSTRAINT fk_videos_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Add FK constraint for videos.user_id -> users(id)
ALTER TABLE videos 
DROP CONSTRAINT IF EXISTS fk_videos_user;

ALTER TABLE videos 
ADD CONSTRAINT fk_videos_user 
FOREIGN KEY (user_id) 
REFERENCES users(id) 
ON DELETE RESTRICT;

-- Add FK constraint for videos.category_id -> categories(id)
ALTER TABLE videos 
DROP CONSTRAINT IF EXISTS fk_videos_category;

ALTER TABLE videos 
ADD CONSTRAINT fk_videos_category 
FOREIGN KEY (category_id) 
REFERENCES categories(id) 
ON DELETE SET NULL;

-- Add FK constraint for categories.instance_id -> instances(id)
ALTER TABLE categories 
DROP CONSTRAINT IF EXISTS fk_categories_instance;

ALTER TABLE categories 
ADD CONSTRAINT fk_categories_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Add FK constraint for categories.parent_id -> categories(id) (self-referential)
ALTER TABLE categories 
DROP CONSTRAINT IF EXISTS fk_categories_parent;

ALTER TABLE categories 
ADD CONSTRAINT fk_categories_parent 
FOREIGN KEY (parent_id) 
REFERENCES categories(id) 
ON DELETE SET NULL;

-- Add FK constraint for tags.instance_id -> instances(id)
ALTER TABLE tags 
DROP CONSTRAINT IF EXISTS fk_tags_instance;

ALTER TABLE tags 
ADD CONSTRAINT fk_tags_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Add FK constraint for comments.instance_id -> instances(id)
ALTER TABLE comments 
DROP CONSTRAINT IF EXISTS fk_comments_instance;

ALTER TABLE comments 
ADD CONSTRAINT fk_comments_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Add FK constraint for comments.video_id -> videos(id)
ALTER TABLE comments 
DROP CONSTRAINT IF EXISTS fk_comments_video;

ALTER TABLE comments 
ADD CONSTRAINT fk_comments_video 
FOREIGN KEY (video_id) 
REFERENCES videos(id) 
ON DELETE CASCADE;

-- Add FK constraint for comments.user_id -> users(id)
ALTER TABLE comments 
DROP CONSTRAINT IF EXISTS fk_comments_user;

ALTER TABLE comments 
ADD CONSTRAINT fk_comments_user 
FOREIGN KEY (user_id) 
REFERENCES users(id) 
ON DELETE RESTRICT;

-- Add FK constraint for comments.parent_id -> comments(id) (self-referential)
ALTER TABLE comments 
DROP CONSTRAINT IF EXISTS fk_comments_parent;

ALTER TABLE comments 
ADD CONSTRAINT fk_comments_parent 
FOREIGN KEY (parent_id) 
REFERENCES comments(id) 
ON DELETE CASCADE;

-- Add FK constraint for ratings.instance_id -> instances(id)
ALTER TABLE ratings 
DROP CONSTRAINT IF EXISTS fk_ratings_instance;

ALTER TABLE ratings 
ADD CONSTRAINT fk_ratings_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Add FK constraint for ratings.video_id -> videos(id)
ALTER TABLE ratings 
DROP CONSTRAINT IF EXISTS fk_ratings_video;

ALTER TABLE ratings 
ADD CONSTRAINT fk_ratings_video 
FOREIGN KEY (video_id) 
REFERENCES videos(id) 
ON DELETE CASCADE;

-- Add FK constraint for ratings.user_id -> users(id)
ALTER TABLE ratings 
DROP CONSTRAINT IF EXISTS fk_ratings_user;

ALTER TABLE ratings 
ADD CONSTRAINT fk_ratings_user 
FOREIGN KEY (user_id) 
REFERENCES users(id) 
ON DELETE CASCADE;

-- Add FK constraint for favorites.instance_id -> instances(id)
ALTER TABLE favorites 
DROP CONSTRAINT IF EXISTS fk_favorites_instance;

ALTER TABLE favorites 
ADD CONSTRAINT fk_favorites_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Add FK constraint for favorites.user_id -> users(id)
ALTER TABLE favorites 
DROP CONSTRAINT IF EXISTS fk_favorites_user;

ALTER TABLE favorites 
ADD CONSTRAINT fk_favorites_user 
FOREIGN KEY (user_id) 
REFERENCES users(id) 
ON DELETE CASCADE;

-- Add FK constraint for favorites.video_id -> videos(id)
ALTER TABLE favorites 
DROP CONSTRAINT IF EXISTS fk_favorites_video;

ALTER TABLE favorites 
ADD CONSTRAINT fk_favorites_video 
FOREIGN KEY (video_id) 
REFERENCES videos(id) 
ON DELETE CASCADE;

-- Add FK constraint for playlists.instance_id -> instances(id)
ALTER TABLE playlists 
DROP CONSTRAINT IF EXISTS fk_playlists_instance;

ALTER TABLE playlists 
ADD CONSTRAINT fk_playlists_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Add FK constraint for playlists.user_id -> users(id)
ALTER TABLE playlists 
DROP CONSTRAINT IF EXISTS fk_playlists_user;

ALTER TABLE playlists 
ADD CONSTRAINT fk_playlists_user 
FOREIGN KEY (user_id) 
REFERENCES users(id) 
ON DELETE CASCADE;

-- Add FK constraint for video_views.instance_id -> instances(id)
ALTER TABLE video_views 
DROP CONSTRAINT IF EXISTS fk_video_views_instance;

ALTER TABLE video_views 
ADD CONSTRAINT fk_video_views_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Add FK constraint for video_views.video_id -> videos(id)
ALTER TABLE video_views 
DROP CONSTRAINT IF EXISTS fk_video_views_video;

ALTER TABLE video_views 
ADD CONSTRAINT fk_video_views_video 
FOREIGN KEY (video_id) 
REFERENCES videos(id) 
ON DELETE CASCADE;

-- Add FK constraint for video_views.user_id -> users(id)
ALTER TABLE video_views 
DROP CONSTRAINT IF EXISTS fk_video_views_user;

ALTER TABLE video_views 
ADD CONSTRAINT fk_video_views_user 
FOREIGN KEY (user_id) 
REFERENCES users(id) 
ON DELETE SET NULL;

-- Add FK constraint for user_sessions.user_id -> users(id)
ALTER TABLE user_sessions 
DROP CONSTRAINT IF EXISTS fk_user_sessions_user;

ALTER TABLE user_sessions 
ADD CONSTRAINT fk_user_sessions_user 
FOREIGN KEY (user_id) 
REFERENCES users(id) 
ON DELETE CASCADE;

-- Add FK constraint for branding_config.instance_id -> instances(id)
ALTER TABLE branding_config 
DROP CONSTRAINT IF EXISTS fk_branding_config_instance;

ALTER TABLE branding_config 
ADD CONSTRAINT fk_branding_config_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Add FK constraint for pages.instance_id -> instances(id)
ALTER TABLE pages 
DROP CONSTRAINT IF EXISTS fk_pages_instance;

ALTER TABLE pages 
ADD CONSTRAINT fk_pages_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Add FK constraint for settings.instance_id -> instances(id)
ALTER TABLE settings 
DROP CONSTRAINT IF EXISTS fk_settings_instance;

ALTER TABLE settings 
ADD CONSTRAINT fk_settings_instance 
FOREIGN KEY (instance_id) 
REFERENCES instances(id) 
ON DELETE CASCADE;

-- Create indexes on foreign key columns for better query performance
CREATE INDEX IF NOT EXISTS idx_users_instance_id ON users(instance_id);
CREATE INDEX IF NOT EXISTS idx_videos_instance_id ON videos(instance_id);
CREATE INDEX IF NOT EXISTS idx_videos_user_id ON videos(user_id);
CREATE INDEX IF NOT EXISTS idx_videos_category_id ON videos(category_id);
CREATE INDEX IF NOT EXISTS idx_categories_instance_id ON categories(instance_id);
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_tags_instance_id ON tags(instance_id);
CREATE INDEX IF NOT EXISTS idx_comments_instance_id ON comments(instance_id);
CREATE INDEX IF NOT EXISTS idx_comments_video_id ON comments(video_id);
CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments(parent_id);
CREATE INDEX IF NOT EXISTS idx_ratings_instance_id ON ratings(instance_id);
CREATE INDEX IF NOT EXISTS idx_ratings_video_id ON ratings(video_id);
CREATE INDEX IF NOT EXISTS idx_ratings_user_id ON ratings(user_id);
CREATE INDEX IF NOT EXISTS idx_favorites_instance_id ON favorites(instance_id);
CREATE INDEX IF NOT EXISTS idx_favorites_user_id ON favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_favorites_video_id ON favorites(video_id);
CREATE INDEX IF NOT EXISTS idx_playlists_instance_id ON playlists(instance_id);
CREATE INDEX IF NOT EXISTS idx_playlists_user_id ON playlists(user_id);
CREATE INDEX IF NOT EXISTS idx_video_views_instance_id ON video_views(instance_id);
CREATE INDEX IF NOT EXISTS idx_video_views_video_id ON video_views(video_id);
CREATE INDEX IF NOT EXISTS idx_video_views_user_id ON video_views(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_branding_config_instance_id ON branding_config(instance_id);
CREATE INDEX IF NOT EXISTS idx_pages_instance_id ON pages(instance_id);
CREATE INDEX IF NOT EXISTS idx_settings_instance_id ON settings(instance_id);
