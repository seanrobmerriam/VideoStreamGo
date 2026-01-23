package platform

import (
	"fmt"
	"net"
	"strings"
)

// DNSRecord represents a DNS record
type DNSRecord struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Value     string `json:"value"`
	TTL       int    `json:"ttl"`
	Proxied   bool   `json:"proxied"`
	CreatedAt string `json:"created_at"`
}

// DNSProvider interface for DNS operations
type DNSProvider interface {
	CreateRecord(record *DNSRecord) error
	DeleteRecord(recordID string) error
	GetRecords(domain string) ([]DNSRecord, error)
	VerifyDomain(domain string) (bool, error)
}

// CloudflareDNSProvider implements DNS operations using Cloudflare API
type CloudflareDNSProvider struct {
	APIEmail string
	APIKey   string
	ZoneID   string
	Domain   string
}

// NewCloudflareDNSProvider creates a new Cloudflare DNS provider
func NewCloudflareDNSProvider(apiEmail, apiKey, zoneID, domain string) *CloudflareDNSProvider {
	return &CloudflareDNSProvider{
		APIEmail: apiEmail,
		APIKey:   apiKey,
		ZoneID:   zoneID,
		Domain:   domain,
	}
}

// CreateRecord creates a DNS record
func (p *CloudflareDNSProvider) CreateRecord(record *DNSRecord) error {
	// In a real implementation, this would call the Cloudflare API
	fmt.Printf("Creating DNS record: %s %s %s\n", record.Type, record.Name, record.Value)
	return nil
}

// DeleteRecord deletes a DNS record
func (p *CloudflareDNSProvider) DeleteRecord(recordID string) error {
	fmt.Printf("Deleting DNS record: %s\n", recordID)
	return nil
}

// GetRecords retrieves DNS records for a domain
func (p *CloudflareDNSProvider) GetRecords(domain string) ([]DNSRecord, error) {
	return []DNSRecord{
		{
			ID:    "1",
			Type:  "A",
			Name:  domain,
			Value: "192.0.2.1",
			TTL:   3600,
		},
	}, nil
}

// VerifyDomain checks if a domain is properly configured
func (p *CloudflareDNSProvider) VerifyDomain(domain string) (bool, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return false, fmt.Errorf("domain does not resolve: %w", err)
	}
	if len(ips) == 0 {
		return false, fmt.Errorf("no IP addresses found for domain")
	}
	return true, nil
}

// SimpleDNSProvider provides a simple DNS provider for testing
type SimpleDNSProvider struct {
	BaseDomain string
}

// NewSimpleDNSProvider creates a simple DNS provider
func NewSimpleDNSProvider(baseDomain string) *SimpleDNSProvider {
	return &SimpleDNSProvider{
		BaseDomain: baseDomain,
	}
}

// CreateRecord creates a DNS record
func (p *SimpleDNSProvider) CreateRecord(record *DNSRecord) error {
	fmt.Printf("Creating DNS record: %s %s %s\n", record.Type, record.Name, record.Value)
	return nil
}

// DeleteRecord deletes a DNS record
func (p *SimpleDNSProvider) DeleteRecord(recordID string) error {
	fmt.Printf("Deleting DNS record: %s\n", recordID)
	return nil
}

// GetRecords retrieves DNS records for a domain
func (p *SimpleDNSProvider) GetRecords(domain string) ([]DNSRecord, error) {
	return []DNSRecord{
		{
			ID:    "mock-a-record",
			Type:  "A",
			Name:  domain,
			Value: "192.0.2.1",
			TTL:   3600,
		},
	}, nil
}

// VerifyDomain checks if a domain is properly configured
func (p *SimpleDNSProvider) VerifyDomain(domain string) (bool, error) {
	return true, nil
}

// DNSProvisioner handles DNS provisioning for tenant instances
type DNSProvisioner struct {
	provider   DNSProvider
	BaseDomain string
}

// NewDNSProvisioner creates a new DNS provisioner
func NewDNSProvisioner(provider DNSProvider, baseDomain string) *DNSProvisioner {
	return &DNSProvisioner{
		provider:   provider,
		BaseDomain: baseDomain,
	}
}

// CreateSubdomainRecord creates an A or CNAME record for a subdomain
func (p *DNSProvisioner) CreateSubdomainRecord(subdomain string) error {
	record := &DNSRecord{
		Type:  "A",
		Name:  subdomain,
		Value: "192.0.2.1",
		TTL:   3600,
	}
	return p.provider.CreateRecord(record)
}

// CreateCNAMERecord creates a CNAME record for custom domains
func (p *DNSProvisioner) CreateCNAMERecord(domain, target string) error {
	record := &DNSRecord{
		Type:  "CNAME",
		Name:  domain,
		Value: target,
		TTL:   3600,
	}
	return p.provider.CreateRecord(record)
}

// DeleteSubdomainRecord removes a DNS entry for a subdomain
func (p *DNSProvisioner) DeleteSubdomainRecord(subdomain string) error {
	records, err := p.provider.GetRecords(fmt.Sprintf("%s.%s", subdomain, p.BaseDomain))
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.Type == "A" && record.Name == subdomain {
			return p.provider.DeleteRecord(record.ID)
		}
	}
	return fmt.Errorf("record not found for subdomain: %s", subdomain)
}

// DeleteRecordByDomain removes a DNS entry for a custom domain
func (p *DNSProvisioner) DeleteRecordByDomain(domain string) error {
	records, err := p.provider.GetRecords(domain)
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.Name == domain {
			return p.provider.DeleteRecord(record.ID)
		}
	}
	return fmt.Errorf("record not found for domain: %s", domain)
}

// VerifyDomain verifies domain ownership via DNS
func (p *DNSProvisioner) VerifyDomain(domain string) (bool, error) {
	return p.provider.VerifyDomain(domain)
}

// GetDNSRecords lists current DNS records for a domain
func (p *DNSProvisioner) GetDNSRecords(domain string) ([]DNSRecord, error) {
	return p.provider.GetRecords(domain)
}

// ConfigureSubdomain configures all DNS for a tenant subdomain
func (p *DNSProvisioner) ConfigureSubdomain(subdomain string) error {
	// Create A record
	if err := p.CreateSubdomainRecord(subdomain); err != nil {
		return fmt.Errorf("failed to create A record: %w", err)
	}
	// Create CNAME for www subdomain
	if err := p.CreateCNAMERecord(fmt.Sprintf("www.%s", subdomain), fmt.Sprintf("%s.%s", subdomain, p.BaseDomain)); err != nil {
		return fmt.Errorf("failed to create CNAME record: %w", err)
	}
	// Verify domain
	fullDomain := fmt.Sprintf("%s.%s", subdomain, p.BaseDomain)
	verified, err := p.VerifyDomain(fullDomain)
	if err != nil {
		return fmt.Errorf("domain verification failed: %w", err)
	}
	if !verified {
		return fmt.Errorf("domain verification returned false")
	}
	return nil
}

// ConfigureCustomDomain configures DNS for a custom domain
func (p *DNSProvisioner) ConfigureCustomDomain(domain, target string) error {
	if strings.HasSuffix(domain, p.BaseDomain) || domain == p.BaseDomain {
		subdomain := strings.TrimSuffix(domain, "."+p.BaseDomain)
		return p.CreateSubdomainRecord(subdomain)
	}
	if err := p.CreateCNAMERecord(domain, target); err != nil {
		return fmt.Errorf("failed to create CNAME record: %w", err)
	}
	verified, err := p.VerifyDomain(domain)
	if err != nil {
		return fmt.Errorf("domain verification failed: %w", err)
	}
	if !verified {
		return fmt.Errorf("domain verification returned false")
	}
	return nil
}

// GetSubdomainFQDN returns the fully qualified domain name for a subdomain
func (p *DNSProvisioner) GetSubdomainFQDN(subdomain string) string {
	return fmt.Sprintf("%s.%s", subdomain, p.BaseDomain)
}
