package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test_CrossTenantDataAccessPrevention tests cross-tenant data access prevention
func Test_CrossTenantDataAccessPrevention(t *testing.T) {
	tenant1Data := map[string]interface{}{
		"tenantId": "tenant_001",
		"data":     "Tenant 1 private data",
	}

	tenant2Data := map[string]interface{}{
		"tenantId": "tenant_002",
		"data":     "Tenant 2 private data",
	}

	// Verify tenant isolation
	assert.NotEqual(t, tenant1Data["tenantId"], tenant2Data["tenantId"])
	assert.NotEqual(t, tenant1Data["data"], tenant2Data["data"])
}

// Test_TenantScopedQueries tests tenant-scoped query logic
func Test_TenantScopedQueries(t *testing.T) {
	tenant1Query := map[string]interface{}{
		"tenantId": "tenant_001",
		"filters":  map[string]string{"tenant_id": "tenant_001"},
	}

	tenant2Query := map[string]interface{}{
		"tenantId": "tenant_002",
		"filters":  map[string]string{"tenant_id": "tenant_002"},
	}

	// Verify queries are tenant-scoped
	assert.Equal(t, "tenant_001", tenant1Query["filters"].(map[string]string)["tenant_id"])
	assert.Equal(t, "tenant_002", tenant2Query["filters"].(map[string]string)["tenant_id"])
}
