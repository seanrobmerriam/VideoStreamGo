package platform

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"videostreamgo/internal/dto/platform"
	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
)

var (
	ErrCustomerNotFound   = errors.New("customer not found")
	ErrCustomerExists     = errors.New("customer already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// CustomerService handles customer business logic
type CustomerService struct {
	customerRepo *masterRepo.CustomerRepository
	instanceRepo *masterRepo.InstanceRepository
}

// NewCustomerService creates a new CustomerService
func NewCustomerService(customerRepo *masterRepo.CustomerRepository, instanceRepo *masterRepo.InstanceRepository) *CustomerService {
	return &CustomerService{
		customerRepo: customerRepo,
		instanceRepo: instanceRepo,
	}
}

// CreateCustomer creates a new customer
func (s *CustomerService) CreateCustomer(ctx context.Context, req *platform.CreateCustomerRequest) (*master.Customer, error) {
	// Check if customer with email already exists
	existing, err := s.customerRepo.GetByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, ErrCustomerExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	customer := &master.Customer{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		CompanyName:  req.CompanyName,
		ContactName:  req.ContactName,
		Phone:        req.Phone,
		Status:       master.CustomerStatusPending,
	}

	if err := s.customerRepo.Create(ctx, customer); err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	return customer, nil
}

// GetCustomer retrieves a customer by ID
func (s *CustomerService) GetCustomer(ctx context.Context, id string) (*master.Customer, error) {
	customerID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, ErrCustomerNotFound
	}

	return customer, nil
}

// ListCustomers retrieves customers with pagination
func (s *CustomerService) ListCustomers(ctx context.Context, page, perPage int, status string) ([]master.Customer, int64, error) {
	offset := (page - 1) * perPage
	return s.customerRepo.List(ctx, offset, perPage, status)
}

// UpdateCustomer updates a customer
func (s *CustomerService) UpdateCustomer(ctx context.Context, id string, req *platform.UpdateCustomerRequest) (*master.Customer, error) {
	customer, err := s.GetCustomer(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Email != nil {
		customer.Email = *req.Email
	}
	if req.CompanyName != nil {
		customer.CompanyName = *req.CompanyName
	}
	if req.ContactName != nil {
		customer.ContactName = *req.ContactName
	}
	if req.Phone != nil {
		customer.Phone = *req.Phone
	}
	if req.Status != nil {
		customer.Status = master.CustomerStatus(*req.Status)
	}

	if err := s.customerRepo.Update(ctx, customer); err != nil {
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	return customer, nil
}

// DeleteCustomer soft deletes a customer
func (s *CustomerService) DeleteCustomer(ctx context.Context, id string) error {
	customerID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	return s.customerRepo.Delete(ctx, customerID)
}

// AuthenticateCustomer authenticates a customer and returns a token
func (s *CustomerService) AuthenticateCustomer(ctx context.Context, req *platform.LoginRequest) (*master.Customer, error) {
	customer, err := s.customerRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(customer.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return customer, nil
}

// GetCustomerWithInstanceCount retrieves a customer with instance count
func (s *CustomerService) GetCustomerWithInstanceCount(ctx context.Context, id string) (*master.Customer, int, error) {
	customer, err := s.GetCustomer(ctx, id)
	if err != nil {
		return nil, 0, err
	}

	instances, err := s.instanceRepo.GetByCustomerID(ctx, customer.ID)
	if err != nil {
		return customer, 0, nil
	}

	return customer, len(instances), nil
}
