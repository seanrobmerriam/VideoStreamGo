package instance

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Comment represents a video comment
type Comment struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID uuid.UUID  `gorm:"type:uuid;not null;index" json:"instance_id"`
	VideoID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"video_id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	ParentID   *uuid.UUID `gorm:"type:uuid;index" json:"parent_id"`
	Content    string     `gorm:"type:text;not null" json:"content"`
	IsEdited   bool       `gorm:"default:false" json:"is_edited"`
	IsDeleted  bool       `gorm:"default:false" json:"is_deleted"`
	LikeCount  int        `gorm:"default:0" json:"like_count"`
	CreatedAt  time.Time  `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// Soft delete
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Video   Video     `gorm:"foreignKey:VideoID" json:"video,omitempty"`
	User    User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Parent  *Comment  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Replies []Comment `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}

// TableName sets the table name for Comment
func (Comment) TableName() string {
	return "comments"
}

// BeforeCreate generates a UUID before creating a new Comment
func (c *Comment) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// Rating represents a video rating (like/dislike)
type Rating struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID uuid.UUID `gorm:"type:uuid;not null;index" json:"instance_id"`
	VideoID    uuid.UUID `gorm:"type:uuid;not null;index" json:"video_id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Rating     int8      `gorm:"not null;check:rating in (-1, 1)" json:"rating"` // -1 or 1
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Video Video `gorm:"foreignKey:VideoID" json:"video,omitempty"`
	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName sets the table name for Rating
func (Rating) TableName() string {
	return "ratings"
}

// BeforeCreate generates a UUID before creating a new Rating
func (r *Rating) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// Favorite represents a user's favorite video
type Favorite struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID uuid.UUID `gorm:"type:uuid;not null;index" json:"instance_id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	VideoID    uuid.UUID `gorm:"type:uuid;not null;index" json:"video_id"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Relationships
	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Video Video `gorm:"foreignKey:VideoID" json:"video,omitempty"`
}

// TableName sets the table name for Favorite
func (Favorite) TableName() string {
	return "favorites"
}

// BeforeCreate generates a UUID before creating a new Favorite
func (f *Favorite) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}
