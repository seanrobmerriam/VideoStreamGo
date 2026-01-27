-- Migration: 016_add_foreign_keys_instance (DOWN)
-- Purpose: Remove foreign key constraints from instance database tables
-- Created: 2025-01-26

-- DOWN Migration: Remove foreign key constraints

-- Remove FK constraint for settings.instance_id
ALTER TABLE settings 
DROP CONSTRAINT IF EXISTS fk_settings_instance;

-- Remove FK constraint for pages.instance_id
ALTER TABLE pages 
DROP CONSTRAINT IF EXISTS fk_pages_instance;

-- Remove FK constraint for branding_config.instance_id
ALTER TABLE branding_config 
DROP CONSTRAINT IF EXISTS fk_branding_config_instance;

-- Remove FK constraint for user_sessions.user_id
ALTER TABLE user_sessions 
DROP CONSTRAINT IF EXISTS fk_user_sessions_user;

-- Remove FK constraint for video_views.user_id
ALTER TABLE video_views 
DROP CONSTRAINT IF EXISTS fk_video_views_user;

-- Remove FK constraint for video_views.video_id
ALTER TABLE video_views 
DROP CONSTRAINT IF EXISTS fk_video_views_video;

-- Remove FK constraint for video_views.instance_id
ALTER TABLE video_views 
DROP CONSTRAINT IF EXISTS fk_video_views_instance;

-- Remove FK constraint for playlists.user_id
ALTER TABLE playlists 
DROP CONSTRAINT IF EXISTS fk_playlists_user;

-- Remove FK constraint for playlists.instance_id
ALTER TABLE playlists 
DROP CONSTRAINT IF EXISTS fk_playlists_instance;

-- Remove FK constraint for favorites.video_id
ALTER TABLE favorites 
DROP CONSTRAINT IF EXISTS fk_favorites_video;

-- Remove FK constraint for favorites.user_id
ALTER TABLE favorites 
DROP CONSTRAINT IF EXISTS fk_favorites_user;

-- Remove FK constraint for favorites.instance_id
ALTER TABLE favorites 
DROP CONSTRAINT IF EXISTS fk_favorites_instance;

-- Remove FK constraint for ratings.user_id
ALTER TABLE ratings 
DROP CONSTRAINT IF EXISTS fk_ratings_user;

-- Remove FK constraint for ratings.video_id
ALTER TABLE ratings 
DROP CONSTRAINT IF EXISTS fk_ratings_video;

-- Remove FK constraint for ratings.instance_id
ALTER TABLE ratings 
DROP CONSTRAINT IF EXISTS fk_ratings_instance;

-- Remove FK constraint for comments.parent_id
ALTER TABLE comments 
DROP CONSTRAINT IF EXISTS fk_comments_parent;

-- Remove FK constraint for comments.user_id
ALTER TABLE comments 
DROP CONSTRAINT IF EXISTS fk_comments_user;

-- Remove FK constraint for comments.video_id
ALTER TABLE comments 
DROP CONSTRAINT IF EXISTS fk_comments_video;

-- Remove FK constraint for comments.instance_id
ALTER TABLE comments 
DROP CONSTRAINT IF EXISTS fk_comments_instance;

-- Remove FK constraint for tags.instance_id
ALTER TABLE tags 
DROP CONSTRAINT IF EXISTS fk_tags_instance;

-- Remove FK constraint for categories.parent_id
ALTER TABLE categories 
DROP CONSTRAINT IF EXISTS fk_categories_parent;

-- Remove FK constraint for categories.instance_id
ALTER TABLE categories 
DROP CONSTRAINT IF EXISTS fk_categories_instance;

-- Remove FK constraint for videos.category_id
ALTER TABLE videos 
DROP CONSTRAINT IF EXISTS fk_videos_category;

-- Remove FK constraint for videos.user_id
ALTER TABLE videos 
DROP CONSTRAINT IF EXISTS fk_videos_user;

-- Remove FK constraint for videos.instance_id
ALTER TABLE videos 
DROP CONSTRAINT IF EXISTS fk_videos_instance;

-- Remove FK constraint for users.instance_id
ALTER TABLE users 
DROP CONSTRAINT IF EXISTS fk_users_instance;

-- Drop indexes on foreign key columns
DROP INDEX IF EXISTS idx_settings_instance_id;
DROP INDEX IF EXISTS idx_pages_instance_id;
DROP INDEX IF EXISTS idx_branding_config_instance_id;
DROP INDEX IF EXISTS idx_user_sessions_user_id;
DROP INDEX IF EXISTS idx_video_views_user_id;
DROP INDEX IF EXISTS idx_video_views_video_id;
DROP INDEX IF EXISTS idx_video_views_instance_id;
DROP INDEX IF EXISTS idx_playlists_user_id;
DROP INDEX IF EXISTS idx_playlists_instance_id;
DROP INDEX IF EXISTS idx_favorites_video_id;
DROP INDEX IF EXISTS idx_favorites_user_id;
DROP INDEX IF EXISTS idx_favorites_instance_id;
DROP INDEX IF EXISTS idx_ratings_user_id;
DROP INDEX IF EXISTS idx_ratings_video_id;
DROP INDEX IF EXISTS idx_ratings_instance_id;
DROP INDEX IF EXISTS idx_comments_parent_id;
DROP INDEX IF EXISTS idx_comments_user_id;
DROP INDEX IF EXISTS idx_comments_video_id;
DROP INDEX IF EXISTS idx_comments_instance_id;
DROP INDEX IF EXISTS idx_tags_instance_id;
DROP INDEX IF EXISTS idx_categories_parent_id;
DROP INDEX IF EXISTS idx_categories_instance_id;
DROP INDEX IF EXISTS idx_videos_category_id;
DROP INDEX IF EXISTS idx_videos_user_id;
DROP INDEX IF EXISTS idx_videos_instance_id;
DROP INDEX IF EXISTS idx_users_instance_id;
