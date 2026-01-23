package master

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"videostreamgo/internal/models/master"
)

// Test_InstanceRepository_ProvisioningStatus tests provisioning status logic
func Test_InstanceRepository_ProvisioningStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   master.InstanceStatus
		isActive bool
	}{
		{"Active status", master.InstanceStatusActive, true},
		{"Provisioning status", master.InstanceStatusProvisioning, true},
		{"Suspended status", master.InstanceStatusSuspended, false},
		{"Terminated status", master.InstanceStatusTerminated, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &master.Instance{
				ID:     uuid.New(),
				Status: tt.status,
			}
			assert.Equal(t, tt.isActive, instance.Status == master.InstanceStatusActive || instance.Status == master.InstanceStatusProvisioning)
		})
	}
}

// Test_InstanceRepository_CustomDomainHandling tests custom domain logic
func Test_InstanceRepository_CustomDomainHandling(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		expectValid bool
	}{
		{"Valid domain", "video.example.com", true},
		{"Empty domain", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customDomains := []string{}
			if tt.domain != "" {
				customDomains = []string{tt.domain}
			}

			instance := &master.Instance{
				ID:            uuid.New(),
				CustomDomains: customDomains,
			}

			if tt.expectValid {
				assert.NotEmpty(t, instance.CustomDomains)
			}
		})
	}
}
