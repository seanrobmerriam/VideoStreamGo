package instance

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category represents a video category
type Category struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"instance_id"`
	Name        string     `gorm:"type:varchar(100);not null" json:"name"`
	Slug        string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"slug"`
	Description string     `gorm:"type:text" json:"description"`
	ParentID    *uuid.UUID `gorm:"type:uuid;index" json:"parent_id"`
	IconURL     string     `gorm:"type:varchar(500)" json:"icon_url"`
	Color       string     `gorm:"type:varchar(7)" json:"color"` // Hex color code
	SortOrder   int        `gorm:"default:0" json:"sort_order"`
	IsActive    bool       `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Parent   *Category  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Videos   []Video    `gorm:"foreignKey:CategoryID" json:"videos,omitempty"`
}

// TableName sets the table name for Category
func (Category) TableName() string {
	return "categories"
}

// BeforeCreate generates a UUID before creating a new Category
func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// Tag represents a video tag
type Tag struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID uuid.UUID `gorm:"type:uuid;not null;index" json:"instance_id"`
	Name       string    `gorm:"type:varchar(100);not null" json:"name"`
	Slug       string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"slug"`
	UsageCount int       `gorm:"default:0;index" json:"usage_count"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Relationships
	Videos []Video `gorm:"many2many:video_tags;" json:"videos,omitempty"`
}

// TableName sets the table name for Tag
func (Tag) TableName() string {
	return "tags"
}

// BeforeCreate generates a UUID before creating a new Tag
func (t *Tag) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// VideoTag represents the junction table between videos and tags
type VideoTag struct {
	VideoID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"video_id"`
	TagID     uuid.UUID `gorm:"type:uuid;primaryKey" json:"tag_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName sets the table name for VideoTag
func (VideoTag) TableName() string {
	return "video_tags"
}
