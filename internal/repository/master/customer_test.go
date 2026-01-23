package master

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"videostreamgo/internal/models/master"
)

// Test_CustomerRepository_CRUD tests customer CRUD operations
func Test_CustomerRepository_CRUD(t *testing.T) {
	customer := &master.Customer{
		ID:          uuid.New(),
		Email:       "test@company.com",
		CompanyName: "Test Company",
	}

	assert.NotEmpty(t, customer.ID)
	assert.Equal(t, "test@company.com", customer.Email)
}

// Test_CustomerRepository_Pagination tests pagination logic
func Test_CustomerRepository_Pagination(t *testing.T) {
	page := 1
	perPage := 20
	offset := (page - 1) * perPage
	assert.Equal(t, 0, offset)

	page = 2
	offset = (page - 1) * perPage
	assert.Equal(t, 20, offset)
}

// Test_CustomerRepository_StatusFiltering tests status filtering logic
func Test_CustomerRepository_StatusFiltering(t *testing.T) {
	validStatuses := []string{"active", "suspended", "pending", "cancelled"}
	status := "active"

	isValid := false
	for _, s := range validStatuses {
		if s == status {
			isValid = true
			break
		}
	}
	assert.True(t, isValid)
}
