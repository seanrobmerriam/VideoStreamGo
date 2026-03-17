package config

import (
	"encoding/json"
	"time"
)

// TenantConfig holds configuration for a specific tenant
type TenantConfig struct {
	// Instance identification
	InstanceID   string
	InstanceUUID string

	// Feature flags
	Features FeatureFlagsConfig

	// Resource limits
	Limits ResourceLimitsConfig

	// Branding configuration
	Branding BrandingConfig

	// Email templates
	EmailTemplates EmailTemplatesConfig

	// Rate limiting
	RateLimits RateLimitConfig

	// Storage configuration
	Storage StorageConfig
}

// FeatureFlagsConfig holds feature flags for a tenant
type FeatureFlagsConfig struct {
	VideoUpload      bool
	VideoDownload    bool
	UserRegistration bool
	SocialLogin      bool
	Comments         bool
	Likes            bool
	Dislikes         bool
	Playlists        bool
	ChannelPages     bool
	LiveStreaming    bool
	Analytics        bool
	CustomBranding   bool
	CustomDomain     bool
	APIAccess        bool
	WhiteLabel       bool
}

// ResourceLimitsConfig holds resource limits for a tenant
type ResourceLimitsConfig struct {
	MaxStorageGB      int64
	MaxBandwidthGB    int64
	MaxVideos         int64
	MaxUsers          int64
	MaxVideoSizeMB    int64
	MaxVideoDuration  time.Duration
	MaxCategories     int
	MaxTags           int
	MaxPlaylistVideos int
	MaxCommentLength  int
	MaxBioLength      int
}

// BrandingConfig holds branding configuration for a tenant
type BrandingConfig struct {
	SiteName        string
	SiteDescription string
	LogoURL         string
	FaviconURL      string
	OGImageURL      string
	PrimaryColor    string
	SecondaryColor  string
	AccentColor     string
	BackgroundColor string
	TextColor       string
	HeaderHTML      string
	FooterHTML      string
	CustomCSS       string
	SocialLinks     map[string]string
	FooterLinks     []FooterLink
}

// FooterLink represents a link in the footer
type FooterLink struct {
	Title string
	URL   string
}

// EmailTemplatesConfig holds email template configuration for a tenant
type EmailTemplatesConfig struct {
	FromName                   string
	FromEmail                  string
	ReplyToEmail               string
	WelcomeSubject             string
	WelcomeBody                string
	PasswordResetSubject       string
	PasswordResetBody          string
	VideoUploadSubject         string
	VideoUploadBody            string
	CommentNotificationSubject string
	CommentNotificationBody    string
}

// RateLimitConfig holds rate limiting configuration for a tenant
type RateLimitConfig struct {
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
	BurstSize         int
	VideoUploadLimit  int // per hour
	APIRequestLimit   int // per minute
}

// StorageConfig holds storage configuration for a tenant
type StorageConfig struct {
	Provider       string // "s3", "gcs", "azure"
	BucketName     string
	Region         string
	Endpoint       string
	UseSSL         bool
	CDNURL         string
	ThumbnailSizes []ThumbnailSize
	VideoQualities []VideoQuality
}

// ThumbnailSize represents a thumbnail size configuration
type ThumbnailSize struct {
	Width   int
	Height  int
	Quality int
}

// VideoQuality represents a video quality configuration
type VideoQuality struct {
	Name      string
	Width     int
	Height    int
	Bitrate   int
	Extension string
}

// DefaultTenantConfig returns default configuration for a tenant
func DefaultTenantConfig() TenantConfig {
	return TenantConfig{
		Features: FeatureFlagsConfig{
			VideoUpload:      true,
			VideoDownload:    true,
			UserRegistration: true,
			SocialLogin:      false,
			Comments:         true,
			Likes:            true,
			Dislikes:         true,
			Playlists:        true,
			ChannelPages:     true,
			LiveStreaming:    false,
			Analytics:        true,
			CustomBranding:   true,
			CustomDomain:     true,
			APIAccess:        true,
			WhiteLabel:       false,
		},
		Limits: ResourceLimitsConfig{
			MaxStorageGB:      100,
			MaxBandwidthGB:    1000,
			MaxVideos:         10000,
			MaxUsers:          10000,
			MaxVideoSizeMB:    2048,
			MaxVideoDuration:  4 * time.Hour,
			MaxCategories:     20,
			MaxTags:           100,
			MaxPlaylistVideos: 100,
			MaxCommentLength:  5000,
			MaxBioLength:      500,
		},
		Branding: BrandingConfig{
			SiteName:        "My Video Site",
			PrimaryColor:    "#2563eb",
			SecondaryColor:  "#64748b",
			AccentColor:     "#f59e0b",
			BackgroundColor: "#ffffff",
			TextColor:       "#1e293b",
			SocialLinks:     make(map[string]string),
			FooterLinks:     []FooterLink{},
		},
		EmailTemplates: EmailTemplatesConfig{
			FromName:     "Video Site",
			FromEmail:    "noreply@videosite.com",
			ReplyToEmail: "support@videosite.com",
		},
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			RequestsPerDay:    10000,
			BurstSize:         10,
			VideoUploadLimit:  10,
			APIRequestLimit:   100,
		},
		Storage: StorageConfig{
			Provider: "s3",
			Region:   "us-east-1",
			UseSSL:   true,
			ThumbnailSizes: []ThumbnailSize{
				{Width: 320, Height: 180, Quality: 70},
				{Width: 640, Height: 360, Quality: 80},
				{Width: 1280, Height: 720, Quality: 85},
			},
			VideoQualities: []VideoQuality{
				{Name: "360p", Width: 640, Height: 360, Bitrate: 800000, Extension: "mp4"},
				{Name: "480p", Width: 854, Height: 480, Bitrate: 1400000, Extension: "mp4"},
				{Name: "720p", Width: 1280, Height: 720, Bitrate: 3000000, Extension: "mp4"},
				{Name: "1080p", Width: 1920, Height: 1080, Bitrate: 6000000, Extension: "mp4"},
			},
		},
	}
}

// LoadTenantConfig loads tenant configuration from the master database or cache
type TenantConfigLoader struct {
	masterDB  MasterDBInterface
	cache     CacheInterface
	configTTL time.Duration
}

// MasterDBInterface defines the interface for master database operations
type MasterDBInterface interface {
	GetConfig(instanceID string, key string) (string, error)
	SetConfig(instanceID string, key string, value string) error
	DeleteConfig(instanceID string, key string) error
}

// CacheInterface defines the interface for cache operations
type CacheInterface interface {
	Get(key string) (string, error)
	Set(key string, value string, ttl time.Duration) error
	Delete(key string) error
}

// NewTenantConfigLoader creates a new tenant config loader
func NewTenantConfigLoader(masterDB MasterDBInterface, cache CacheInterface, ttl time.Duration) *TenantConfigLoader {
	return &TenantConfigLoader{
		masterDB:  masterDB,
		cache:     cache,
		configTTL: ttl,
	}
}

// LoadTenantConfig loads configuration for a specific tenant
func (l *TenantConfigLoader) LoadTenantConfig(instanceID string) (*TenantConfig, error) {
	// Try to load from cache first
	cacheKey := "tenant_config:" + instanceID
	if l.cache != nil {
		if cached, err := l.cache.Get(cacheKey); err == nil {
			if config, err := ParseTenantConfig(cached); err == nil {
				return config, nil
			}
		}
	}

	// Load from database
	config := DefaultTenantConfig()
	config.InstanceID = instanceID

	// Load individual config values
	configMap := map[string]*string{
		"site_name":         &config.Branding.SiteName,
		"logo_url":          &config.Branding.LogoURL,
		"primary_color":     &config.Branding.PrimaryColor,
		"max_storage_gb":    nil,
		"max_videos":        nil,
		"video_upload":      nil,
		"user_registration": nil,
	}

	for key, ptr := range configMap {
		if value, err := l.masterDB.GetConfig(instanceID, key); err == nil && ptr != nil {
			*ptr = value
		}
	}

	// Cache the config
	if l.cache != nil {
		serialized := SerializeTenantConfig(config)
		l.cache.Set(cacheKey, serialized, l.configTTL)
	}

	return &config, nil
}

// SaveTenantConfig saves configuration for a specific tenant
func (l *TenantConfigLoader) SaveTenantConfig(instanceID string, config *TenantConfig) error {
	// Save individual config values
	values := map[string]string{
		"site_name":     config.Branding.SiteName,
		"logo_url":      config.Branding.LogoURL,
		"primary_color": config.Branding.PrimaryColor,
	}

	for key, value := range values {
		if err := l.masterDB.SetConfig(instanceID, key, value); err != nil {
			return err
		}
	}

	// Invalidate cache
	cacheKey := "tenant_config:" + instanceID
	if l.cache != nil {
		l.cache.Delete(cacheKey)
	}

	return nil
}

// SerializeTenantConfig serializes a tenant config to JSON string
func SerializeTenantConfig(config TenantConfig) string {
	// Marshal all fields to JSON
	data, err := json.Marshal(config)
	if err != nil {
		// Fallback to minimal serialization on error
		return "{}"
	}
	return string(data)
}

// ParseTenantConfig parses a serialized tenant config
func ParseTenantConfig(data string) (*TenantConfig, error) {
	if data == "" || data == "{}" {
		// Return default config for empty data
		defaultConfig := DefaultTenantConfig()
		return &defaultConfig, nil
	}

	var config TenantConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		// Return default config on parse error
		defaultConfig := DefaultTenantConfig()
		return &defaultConfig, err
	}
	return &config, nil
}
