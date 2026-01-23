package instance

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRole represents the role of a user
type UserRole string

const (
	UserRoleUser      UserRole = "user"
	UserRoleModerator UserRole = "moderator"
	UserRoleAdmin     UserRole = "admin"
)

// UserStatus represents the status of a user account
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusBanned    UserStatus = "banned"
	UserStatusSuspended UserStatus = "suspended"
)

// User represents an end user of a customer site
type User struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"instance_id"`
	Username      string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	Email         string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash  string     `gorm:"type:varchar(255);not null" json:"-"`
	DisplayName   string     `gorm:"type:varchar(100)" json:"display_name"`
	AvatarURL     string     `gorm:"type:varchar(500)" json:"avatar_url"`
	Bio           string     `gorm:"type:text" json:"bio"`
	Role          UserRole   `gorm:"type:varchar(50);default:'user'" json:"role"`
	Status        UserStatus `gorm:"type:varchar(50);default:'active';index" json:"status"`
	EmailVerified bool       `gorm:"default:false" json:"email_verified"`
	LastLoginAt   *time.Time `json:"last_login_at"`
	Metadata      JSONMap    `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// Soft delete
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName sets the table name for User
func (User) TableName() string {
	return "users"
}

// BeforeCreate generates a UUID before creating a new User
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// UserSession represents a user login session
type UserSession struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"-"`
	IPAddress string    `gorm:"type:inet" json:"ip_address"`
	UserAgent string    `gorm:"type:text" json:"user_agent"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName sets the table name for UserSession
func (UserSession) TableName() string {
	return "user_sessions"
}

// BeforeCreate generates a UUID before creating a new UserSession
func (us *UserSession) BeforeCreate(tx *gorm.DB) error {
	if us.ID == uuid.Nil {
		us.ID = uuid.New()
	}
	return nil
}

// IsValid checks if the session is still valid
func (us *UserSession) IsValid() bool {
	return time.Now().Before(us.ExpiresAt)
}
