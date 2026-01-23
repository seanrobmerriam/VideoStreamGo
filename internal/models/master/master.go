// Package master contains all database models for the master database.
// The master database stores platform-level data including customers,
// subscriptions, billing records, and instance configurations.
package master

import (
	"github.com/google/uuid"
)

// UUID is an alias for google/uuid.UUID for convenience
type UUID = uuid.UUID

// BeforeCreate is a hook that can be used in models
func BeforeCreate(db *uuid.UUID) error {
	if *db == uuid.Nil {
		*db = uuid.New()
	}
	return nil
}
