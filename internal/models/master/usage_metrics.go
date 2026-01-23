package master

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MetricType represents the type of usage metric
type MetricType string

const (
	MetricTypeStorage   MetricType = "storage"
	MetricTypeBandwidth MetricType = "bandwidth"
	MetricTypeVideos    MetricType = "videos"
	MetricTypeUsers     MetricType = "users"
	MetricTypeViews     MetricType = "views"
)

// UsageMetrics tracks usage per customer instance
type UsageMetrics struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID  uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_usage_metrics_unique" json:"instance_id"`
	MetricType  MetricType `gorm:"type:varchar(50);not null;uniqueIndex:idx_usage_metrics_unique" json:"metric_type"`
	PeriodStart time.Time  `gorm:"not null;uniqueIndex:idx_usage_metrics_unique" json:"period_start"`
	PeriodEnd   time.Time  `gorm:"not null" json:"period_end"`
	Value       int64      `gorm:"not null" json:"value"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// Relationships
	Instance Instance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
}

// TableName sets the table name for UsageMetrics
func (UsageMetrics) TableName() string {
	return "usage_metrics"
}

// BeforeCreate generates a UUID before creating a new UsageMetrics
func (um *UsageMetrics) BeforeCreate(tx *gorm.DB) error {
	if um.ID == uuid.Nil {
		um.ID = uuid.New()
	}
	return nil
}

// UsageSnapshot represents a summary of current usage for an instance
type UsageSnapshot struct {
	StorageUsed   int64 `json:"storage_used"`   // in bytes
	BandwidthUsed int64 `json:"bandwidth_used"` // in bytes
	VideosCount   int64 `json:"videos_count"`
	UsersCount    int64 `json:"users_count"`
	ViewsCount    int64 `json:"views_count"`
}
