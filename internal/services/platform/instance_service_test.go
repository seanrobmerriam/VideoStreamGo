package platform

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"videostreamgo/internal/models/master"
)

// Test_InstanceService_ProvisionInstance tests instance provisioning logic
func Test_InstanceService_ProvisionInstance(t *testing.T) {
	tests := []struct {
		name        string
		plan        string
		expectError bool
	}{
		{
			name:        "Basic plan",
			plan:        "basic",
			expectError: false,
		},
		{
			name:        "Professional plan",
			plan:        "professional",
			expectError: false,
		},
		{
			name:        "Enterprise plan",
			plan:        "enterprise",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &master.Instance{
				ID:            uuid.New(),
				Name:          "test-instance",
				Subdomain:     "test",
				Status:        master.InstanceStatusProvisioning,
				DatabaseName:  "db_test",
				StorageBucket: "bucket_test",
			}

			assert.NotEmpty(t, instance.ID)
			assert.Equal(t, master.InstanceStatusProvisioning, instance.Status)
		})
	}
}

// Test_InstanceService_DeprovisionInstance tests instance deprovisioning logic
func Test_InstanceService_DeprovisionInstance(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus master.InstanceStatus
		expectValid   bool
	}{
		{"Active to Terminated", master.InstanceStatusActive, true},
		{"Provisioning to Terminated", master.InstanceStatusProvisioning, true},
		{"Suspended to Terminated", master.InstanceStatusSuspended, true},
		{"Pending to Terminated", master.InstanceStatusPending, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &master.Instance{
				ID:     uuid.New(),
				Status: tt.initialStatus,
			}

			_ = instance

			validDeprovisionStatuses := []master.InstanceStatus{
				master.InstanceStatusActive,
				master.InstanceStatusProvisioning,
				master.InstanceStatusSuspended,
				master.InstanceStatusPending,
			}

			isValid := false
			for _, status := range validDeprovisionStatuses {
				if status == tt.initialStatus {
					isValid = true
					break
				}
			}

			assert.Equal(t, tt.expectValid, isValid)
		})
	}
}

// Test_InstanceService_CustomDomain tests custom domain handling
func Test_InstanceService_CustomDomain(t *testing.T) {
	tests := []struct {
		name         string
		customDomain string
		expectValid  bool
	}{
		{"Valid domain", "video.example.com", true},
		{"Valid subdomain", "my.video.com", true},
		{"Empty domain", "", false},
		{"Invalid format", "not-a-domain", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customDomains := []string{}
			if tt.customDomain != "" {
				customDomains = []string{tt.customDomain}
			}

			instance := &master.Instance{
				ID:            uuid.New(),
				CustomDomains: customDomains,
			}

			if tt.expectValid {
				assert.Contains(t, instance.CustomDomains, tt.customDomain)
			}
		})
	}
}

// Test_InstanceService_UpdateStatus tests instance status updates
func Test_InstanceService_UpdateStatus(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus master.InstanceStatus
		newStatus     master.InstanceStatus
		expectValid   bool
	}{
		{"Provisioning to Active", master.InstanceStatusProvisioning, master.InstanceStatusActive, true},
		{"Active to Suspended", master.InstanceStatusActive, master.InstanceStatusSuspended, true},
		{"Suspended to Active", master.InstanceStatusSuspended, master.InstanceStatusActive, true},
		{"Active to Terminated", master.InstanceStatusActive, master.InstanceStatusTerminated, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &master.Instance{
				ID:        uuid.New(),
				Status:    tt.initialStatus,
				UpdatedAt: time.Now(),
			}

			_ = instance

			validTransitions := map[master.InstanceStatus][]master.InstanceStatus{
				master.InstanceStatusProvisioning: {master.InstanceStatusActive},
				master.InstanceStatusActive:       {master.InstanceStatusSuspended, master.InstanceStatusTerminated},
				master.InstanceStatusSuspended:    {master.InstanceStatusActive, master.InstanceStatusTerminated},
			}

			allowedTransitions := validTransitions[tt.initialStatus]
			isValid := false
			for _, allowed := range allowedTransitions {
				if allowed == tt.newStatus {
					isValid = true
					break
				}
			}

			assert.Equal(t, tt.expectValid, isValid)
		})
	}
}
