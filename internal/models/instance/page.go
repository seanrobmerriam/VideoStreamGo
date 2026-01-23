package instance

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PageType represents the type of page
type PageType string

const (
	PageTypeStatic PageType = "static"
	PageTypeCustom PageType = "custom"
	PageTypeLegal  PageType = "legal" // terms, privacy policy, etc.
	PageTypeError  PageType = "error" // 404, 500, etc.
)

// Page represents a custom page in an instance
type Page struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID      uuid.UUID `gorm:"type:uuid;not null;index" json:"instance_id"`
	Title           string    `gorm:"type:varchar(255);not null" json:"title"`
	Slug            string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Content         string    `gorm:"type:text;not null" json:"content"`
	Excerpt         string    `gorm:"type:varchar(500)" json:"excerpt"`
	PageType        PageType  `gorm:"type:varchar(50);default:'custom'" json:"page_type"`
	Template        string    `gorm:"type:varchar(100)" json:"template"`
	MetaTitle       string    `gorm:"type:varchar(255)" json:"meta_title"`
	MetaDescription string    `gorm:"type:varchar(500)" json:"meta_description"`
	FeaturedImage   string    `gorm:"type:varchar(500)" json:"featured_image"`
	IsPublished     bool      `gorm:"default:true" json:"is_published"`
	ShowInMenu      bool      `gorm:"default:false" json:"show_in_menu"`
	MenuOrder       int       `gorm:"default:0" json:"menu_order"`
	CreatedBy       uuid.UUID `gorm:"type:uuid" json:"created_by"`
	UpdatedBy       uuid.UUID `gorm:"type:uuid" json:"updated_by"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Creator User `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

// TableName sets the table name for Page
func (Page) TableName() string {
	return "pages"
}

// BeforeCreate generates a UUID before creating a new Page
func (p *Page) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// Setting represents instance-wide settings
type Setting struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_instance_setting_key" json:"instance_id"`
	Key         string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_instance_setting_key" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName sets the table name for Setting
func (Setting) TableName() string {
	return "settings"
}

// BeforeCreate generates a UUID before creating a new Setting
func (s *Setting) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// Common setting keys
const (
	SettingRegistrationEnabled       = "registration_enabled"
	SettingEmailVerificationRequired = "email_verification_required"
	SettingAllowAnonymousViews       = "allow_anonymous_views"
	SettingDefaultVideoVisibility    = "default_video_visibility"
	SettingMaxUploadSizeMB           = "max_upload_size_mb"
	SettingAllowedVideoFormats       = "allowed_video_formats"
	SettingEnableComments            = "enable_comments"
	SettingEnableRatings             = "enable_ratings"
	SettingEnableFavorites           = "enable_favorites"
	SettingEnablePlaylists           = "enable_playlists"
	SettingVideosPerPage             = "videos_per_page"
	SettingCommentsPerPage           = "comments_per_page"
	SettingEnableWatermark           = "enable_watermark"
	SettingWatermarkPosition         = "watermark_position"
	SettingEnableCDN                 = "enable_cdn"
	SettingCDNURL                    = "cdn_url"
)
