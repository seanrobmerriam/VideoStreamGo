package instance

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Playlist represents a user-created video playlist
type Playlist struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID  uuid.UUID `gorm:"type:uuid;not null;index" json:"instance_id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	IsPublic    bool      `gorm:"default:true" json:"is_public"`
	ViewCount   int64     `gorm:"default:0" json:"view_count"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	User   User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Videos []PlaylistVideo `gorm:"foreignKey:PlaylistID" json:"videos,omitempty"`
}

// TableName sets the table name for Playlist
func (Playlist) TableName() string {
	return "playlists"
}

// BeforeCreate generates a UUID before creating a new Playlist
func (p *Playlist) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// PlaylistVideo represents a video in a playlist
type PlaylistVideo struct {
	PlaylistID uuid.UUID `gorm:"type:uuid;primaryKey" json:"playlist_id"`
	VideoID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"video_id"`
	Position   int       `gorm:"not null" json:"position"`
	AddedAt    time.Time `gorm:"autoCreateTime" json:"added_at"`

	// Relationships
	Playlist Playlist `gorm:"foreignKey:PlaylistID" json:"playlist,omitempty"`
	Video    Video    `gorm:"foreignKey:VideoID" json:"video,omitempty"`
}

// TableName sets the table name for PlaylistVideo
func (PlaylistVideo) TableName() string {
	return "playlist_videos"
}

// BrandingConfig represents the branding configuration for an instance
type BrandingConfig struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID      uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"instance_id"`
	SiteName        string    `gorm:"type:varchar(255);default:'VideoTube'" json:"site_name"`
	LogoURL         string    `gorm:"type:varchar(500)" json:"logo_url"`
	FaviconURL      string    `gorm:"type:varchar(500)" json:"favicon_url"`
	PrimaryColor    string    `gorm:"type:varchar(7);default:'#2563eb'" json:"primary_color"`
	SecondaryColor  string    `gorm:"type:varchar(7);default:'#64748b'" json:"secondary_color"`
	AccentColor     string    `gorm:"type:varchar(7);default:'#f59e0b'" json:"accent_color"`
	BackgroundColor string    `gorm:"type:varchar(7);default:'#ffffff'" json:"background_color"`
	TextColor       string    `gorm:"type:varchar(7);default:'#1e293b'" json:"text_color"`
	HeaderHTML      string    `gorm:"type:text" json:"header_html"`
	FooterHTML      string    `gorm:"type:text" json:"footer_html"`
	CustomCSS       string    `gorm:"type:text" json:"custom_css"`
	SocialLinks     JSONMap   `gorm:"type:jsonb;default:'{}'" json:"social_links"`
	FooterLinks     JSONMap   `gorm:"type:jsonb;default:'[]'" json:"footer_links"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName sets the table name for BrandingConfig
func (BrandingConfig) TableName() string {
	return "branding_config"
}

// BeforeCreate generates a UUID before creating a new BrandingConfig
func (bc *BrandingConfig) BeforeCreate(tx *gorm.DB) error {
	if bc.ID == uuid.Nil {
		bc.ID = uuid.New()
	}
	return nil
}
