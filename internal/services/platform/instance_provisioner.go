package platform

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"videostreamgo/internal/config"
	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
)

// InstanceStatus represents the provisioning status
type InstanceStatus string

const (
	InstanceStatusPending      InstanceStatus = "pending"
	InstanceStatusProvisioning InstanceStatus = "provisioning"
	InstanceStatusActive       InstanceStatus = "active"
	InstanceStatusSuspended    InstanceStatus = "suspended"
	InstanceStatusTerminated   InstanceStatus = "terminated"
)

// ProvisioningStep represents a step in the provisioning process
type ProvisioningStep struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
}

// ProvisioningProgress tracks the progress of instance provisioning
type ProvisioningProgress struct {
	InstanceID  uuid.UUID          `json:"instance_id"`
	Steps       []ProvisioningStep `json:"steps"`
	CurrentStep int                `json:"current_step"`
	TotalSteps  int                `json:"total_steps"`
	StartedAt   time.Time          `json:"started_at"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
	Error       string             `json:"error,omitempty"`
}

// InstanceProvisioner handles full instance provisioning workflow
type InstanceProvisioner struct {
	masterDB        *gorm.DB
	instanceRepo    *masterRepo.InstanceRepository
	dbProvisioner   *DBProvisioner
	storage         *StorageProvisioner
	dnsProvisioner  *DNSProvisioner
	certProvisioner *CertProvisioner
	certStorage     *CertStorage
	config          *config.Config
	baseDomain      string
}

// NewInstanceProvisioner creates a new instance provisioner
func NewInstanceProvisioner(
	masterDB *gorm.DB,
	instanceRepo *masterRepo.InstanceRepository,
	cfg *config.Config,
) (*InstanceProvisioner, error) {
	dbProvisioner, err := NewDBProvisioner(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create database provisioner: %w", err)
	}

	storage, err := NewStorageProvisioner(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage provisioner: %w", err)
	}

	dnsProvider := NewSimpleDNSProvider("videostreamgo.com")
	dnsProvisioner := NewDNSProvisioner(dnsProvider, "videostreamgo.com")

	certProvisioner := NewCertProvisioner("admin@videostreamgo.com", "VideoStreamGo", "self-signed")
	certStorage := NewCertStorage()

	return &InstanceProvisioner{
		masterDB:        masterDB,
		instanceRepo:    instanceRepo,
		dbProvisioner:   dbProvisioner,
		storage:         storage,
		dnsProvisioner:  dnsProvisioner,
		certProvisioner: certProvisioner,
		certStorage:     certStorage,
		config:          cfg,
		baseDomain:      "videostreamgo.com",
	}, nil
}

// ProvisionInstance provisions a complete tenant instance
func (p *InstanceProvisioner) ProvisionInstance(ctx context.Context, instanceID uuid.UUID) (*ProvisioningProgress, error) {
	progress := &ProvisioningProgress{
		InstanceID:  instanceID,
		Steps:       p.getProvisioningSteps(),
		CurrentStep: 0,
		TotalSteps:  len(p.getProvisioningSteps()),
		StartedAt:   time.Now(),
	}

	instance, err := p.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		progress.Steps[0].Status = "failed"
		progress.Steps[0].Message = fmt.Sprintf("Instance not found: %v", err)
		progress.Error = err.Error()
		return progress, fmt.Errorf("instance not found: %w", err)
	}

	instance.Status = master.InstanceStatusProvisioning
	if err := p.instanceRepo.Update(ctx, instance); err != nil {
		return progress, fmt.Errorf("failed to update instance status: %w", err)
	}

	for i, step := range progress.Steps {
		progress.CurrentStep = i
		step.StartedAt = time.Now()
		step.Status = "in_progress"

		err := p.executeStep(ctx, instance, &step)
		step.EndedAt = time.Now()

		if err != nil {
			step.Status = "failed"
			step.Message = err.Error()
			progress.Error = err.Error()

			instance.Status = master.InstanceStatusPending
			p.instanceRepo.Update(ctx, instance)

			return progress, fmt.Errorf("step %s failed: %w", step.Name, err)
		}

		step.Status = "completed"
		step.Message = "Success"
	}

	now := time.Now()
	progress.CompletedAt = &now
	instance.Status = master.InstanceStatusActive
	instance.ActivatedAt = &now
	if err := p.instanceRepo.Update(ctx, instance); err != nil {
		return progress, fmt.Errorf("failed to update instance status: %w", err)
	}

	return progress, nil
}

// executeStep executes a single provisioning step
func (p *InstanceProvisioner) executeStep(ctx context.Context, instance *master.Instance, step *ProvisioningStep) error {
	switch step.Name {
	case "validate_instance":
		return p.validateInstance(ctx, instance)
	case "provision_database":
		return p.provisionDatabase(ctx, instance)
	case "run_migrations":
		return p.runMigrations(ctx, instance)
	case "provision_storage":
		return p.provisionStorage(ctx, instance)
	case "configure_dns":
		return p.configureDNS(ctx, instance)
	case "provision_ssl":
		return p.provisionSSL(ctx, instance)
	case "finalize":
		return p.finalizeProvisioning(ctx, instance)
	default:
		return fmt.Errorf("unknown step: %s", step.Name)
	}
}

// getProvisioningSteps returns the ordered list of provisioning steps
func (p *InstanceProvisioner) getProvisioningSteps() []ProvisioningStep {
	return []ProvisioningStep{
		{Name: "validate_instance", Status: "pending"},
		{Name: "provision_database", Status: "pending"},
		{Name: "run_migrations", Status: "pending"},
		{Name: "provision_storage", Status: "pending"},
		{Name: "configure_dns", Status: "pending"},
		{Name: "provision_ssl", Status: "pending"},
		{Name: "finalize", Status: "pending"},
	}
}

// validateInstance validates instance before provisioning
func (p *InstanceProvisioner) validateInstance(ctx context.Context, instance *master.Instance) error {
	if instance.Status != master.InstanceStatusPending {
		return fmt.Errorf("instance status must be pending, got: %s", instance.Status)
	}
	if instance.DatabaseName == "" {
		return fmt.Errorf("database name is required")
	}
	if instance.StorageBucket == "" {
		return fmt.Errorf("storage bucket is required")
	}
	return nil
}

// provisionDatabase creates the instance database
func (p *InstanceProvisioner) provisionDatabase(ctx context.Context, instance *master.Instance) error {
	if err := p.dbProvisioner.CreateDatabase(instance.DatabaseName); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	return nil
}

// runMigrations runs database migrations
func (p *InstanceProvisioner) runMigrations(ctx context.Context, instance *master.Instance) error {
	if err := p.dbProvisioner.RunMigrations(instance.DatabaseName); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

// provisionStorage creates the S3 bucket
func (p *InstanceProvisioner) provisionStorage(ctx context.Context, instance *master.Instance) error {
	if err := p.storage.CreateBucket(instance.StorageBucket); err != nil {
		return fmt.Errorf("failed to create storage bucket: %w", err)
	}
	return nil
}

// configureDNS sets up DNS records
func (p *InstanceProvisioner) configureDNS(ctx context.Context, instance *master.Instance) error {
	if err := p.dnsProvisioner.ConfigureSubdomain(instance.Subdomain); err != nil {
		return fmt.Errorf("failed to configure DNS: %w", err)
	}
	return nil
}

// provisionSSL provisions SSL certificate
func (p *InstanceProvisioner) provisionSSL(ctx context.Context, instance *master.Instance) error {
	domain := fmt.Sprintf("%s.%s", instance.Subdomain, p.baseDomain)

	cert, err := p.certProvisioner.RequestCertificate(domain)
	if err != nil {
		return fmt.Errorf("failed to request certificate: %w", err)
	}

	p.certStorage.StoreCertificate(domain, cert)

	return nil
}

// finalizeProvisioning completes the provisioning process
func (p *InstanceProvisioner) finalizeProvisioning(ctx context.Context, instance *master.Instance) error {
	log.Printf("Instance %s provisioned successfully", instance.ID)
	return nil
}

// DeprovisionInstance removes all resources for a tenant instance
func (p *InstanceProvisioner) DeprovisionInstance(ctx context.Context, instanceID uuid.UUID) error {
	instance, err := p.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	instance.Status = master.InstanceStatusTerminated
	if err := p.instanceRepo.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	if err := p.dnsProvisioner.DeleteSubdomainRecord(instance.Subdomain); err != nil {
		log.Printf("Warning: failed to delete DNS records: %v", err)
	}

	domain := fmt.Sprintf("%s.%s", instance.Subdomain, p.baseDomain)
	if cert, ok := p.certStorage.GetCertificate(domain); ok {
		p.certProvisioner.RevokeCertificate(domain, cert)
		p.certStorage.DeleteCertificate(domain)
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := p.storage.DeleteBucket(ctx, instance.StorageBucket); err != nil {
			log.Printf("Warning: failed to delete storage bucket: %v", err)
		}
	}()

	log.Printf("Instance %s deprovisioned", instanceID)
	return nil
}

// GetInstanceStatus returns the current provisioning status
func (p *InstanceProvisioner) GetInstanceStatus(ctx context.Context, instanceID uuid.UUID) (*master.Instance, error) {
	return p.instanceRepo.GetByID(ctx, instanceID)
}

// ListInstances returns instances with optional filtering
func (p *InstanceProvisioner) ListInstances(ctx context.Context, status string, limit, offset int) ([]*master.Instance, int64, error) {
	instances, total, err := p.instanceRepo.List(ctx, offset, limit, status)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*master.Instance, len(instances))
	for i := range instances {
		result[i] = &instances[i]
	}
	return result, total, nil
}

// GetProvisioningProgress returns the progress of a provisioning operation
func (p *InstanceProvisioner) GetProvisioningProgress(ctx context.Context, instanceID uuid.UUID) (*ProvisioningProgress, error) {
	instance, err := p.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	progress := &ProvisioningProgress{
		InstanceID: instanceID,
		Steps:      p.getProvisioningSteps(),
		TotalSteps: len(p.getProvisioningSteps()),
	}

	switch instance.Status {
	case master.InstanceStatusPending:
		progress.CurrentStep = 0
	case master.InstanceStatusProvisioning:
		progress.CurrentStep = progress.TotalSteps / 2
		for i := 0; i < progress.CurrentStep; i++ {
			progress.Steps[i].Status = "completed"
		}
		progress.Steps[progress.CurrentStep].Status = "in_progress"
	case master.InstanceStatusActive:
		progress.CurrentStep = progress.TotalSteps - 1
		for i := 0; i < progress.TotalSteps; i++ {
			progress.Steps[i].Status = "completed"
		}
	}

	return progress, nil
}

// AddCustomDomain adds a custom domain to an instance
func (p *InstanceProvisioner) AddCustomDomain(ctx context.Context, instanceID uuid.UUID, domain string) error {
	instance, err := p.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	verified, err := p.dnsProvisioner.VerifyDomain(domain)
	if err != nil {
		return fmt.Errorf("domain verification failed: %w", err)
	}
	if !verified {
		return fmt.Errorf("domain verification returned false")
	}

	target := fmt.Sprintf("%s.%s", instance.Subdomain, p.baseDomain)
	if err := p.dnsProvisioner.ConfigureCustomDomain(domain, target); err != nil {
		return fmt.Errorf("failed to configure DNS: %w", err)
	}

	cert, err := p.certProvisioner.RequestCertificate(domain)
	if err != nil {
		return fmt.Errorf("failed to request certificate: %w", err)
	}
	p.certStorage.StoreCertificate(domain, cert)

	instance.CustomDomains = append(instance.CustomDomains, domain)
	if err := p.instanceRepo.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	return nil
}

// GetInstanceMetrics returns usage metrics for an instance
func (p *InstanceProvisioner) GetInstanceMetrics(ctx context.Context, instanceID uuid.UUID) (map[string]interface{}, error) {
	instance, err := p.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	metrics := make(map[string]interface{})

	dbStats, err := p.dbProvisioner.GetDatabaseStats(instance.DatabaseName)
	if err != nil {
		log.Printf("Warning: failed to get database stats: %v", err)
		dbStats = &DatabaseStats{}
	}
	metrics["database_size_bytes"] = dbStats.SizeBytes
	metrics["database_connections"] = dbStats.ConnectionCount

	storageUsage, err := p.storage.GetBucketUsage(ctx, instance.StorageBucket)
	if err != nil {
		log.Printf("Warning: failed to get storage usage: %v", err)
		storageUsage = &BucketUsage{}
	}
	metrics["storage_bytes"] = storageUsage.SizeBytes
	metrics["storage_objects"] = storageUsage.ObjectCount

	domain := fmt.Sprintf("%s.%s", instance.Subdomain, p.baseDomain)
	if cert, ok := p.certStorage.GetCertificate(domain); ok {
		metrics["ssl_status"] = p.certProvisioner.GetCertificateStatus(cert)
		metrics["ssl_expires_at"] = cert.NotAfter
	}

	return metrics, nil
}

// InstanceProvisionerInterface defines the interface for instance provisioning
type InstanceProvisionerInterface interface {
	ProvisionInstance(ctx context.Context, instanceID uuid.UUID) (*ProvisioningProgress, error)
	DeprovisionInstance(ctx context.Context, instanceID uuid.UUID) error
	GetInstanceStatus(ctx context.Context, instanceID uuid.UUID) (*master.Instance, error)
	ListInstances(ctx context.Context, status string, limit, offset int) ([]*master.Instance, int64, error)
	GetProvisioningProgress(ctx context.Context, instanceID uuid.UUID) (*ProvisioningProgress, error)
	AddCustomDomain(ctx context.Context, instanceID uuid.UUID, domain string) error
	GetInstanceMetrics(ctx context.Context, instanceID uuid.UUID) (map[string]interface{}, error)
}
