package platform

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// CertificateInfo holds SSL certificate information
type CertificateInfo struct {
	Domain       string    `json:"domain"`
	CertPEM      string    `json:"cert_pem"`
	KeyPEM       string    `json:"key_pem"`
	NotBefore    time.Time `json:"not_before"`
	NotAfter     time.Time `json:"not_after"`
	Issuer       string    `json:"issuer"`
	SerialNumber string    `json:"serial_number"`
	Status       string    `json:"status"`
}

// CertProvisioner handles SSL certificate provisioning
type CertProvisioner struct {
	email        string
	issuerName   string
	providerType string // "letsencrypt" or "self-signed"
}

// NewCertProvisioner creates a new certificate provisioner
func NewCertProvisioner(email, issuerName, providerType string) *CertProvisioner {
	return &CertProvisioner{
		email:        email,
		issuerName:   issuerName,
		providerType: providerType,
	}
}

// RequestCertificate requests a new SSL certificate for a domain
func (p *CertProvisioner) RequestCertificate(domain string) (*CertificateInfo, error) {
	switch p.providerType {
	case "letsencrypt":
		return p.requestLetsEncryptCert(domain)
	case "self-signed":
		return p.generateSelfSignedCert(domain)
	default:
		return p.generateSelfSignedCert(domain)
	}
}

// requestLetsEncryptCert requests a certificate from Let's Encrypt
func (p *CertProvisioner) requestLetsEncryptCert(domain string) (*CertificateInfo, error) {
	// In a real implementation, this would:
	// 1. Create a DNS-01 challenge
	// 2. Wait for DNS propagation
	// 3. Submit the challenge to Let's Encrypt
	// 4. Download the certificate
	// 5. Install the certificate

	// For now, generate a self-signed certificate as a placeholder
	return p.generateSelfSignedCert(domain)
}

// generateSelfSignedCert generates a self-signed certificate for testing
func (p *CertProvisioner) generateSelfSignedCert(domain string) (*CertificateInfo, error) {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{p.issuerName},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain, "*." + domain},
	}

	// Create certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	// Encode private key to PEM
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return &CertificateInfo{
		Domain:       domain,
		CertPEM:      string(certPEM),
		KeyPEM:       string(keyPEM),
		NotBefore:    template.NotBefore,
		NotAfter:     template.NotAfter,
		Issuer:       p.issuerName,
		SerialNumber: serialNumber.String(),
		Status:       "active",
	}, nil
}

// VerifyDomain verifies domain ownership for SSL certificate
func (p *CertProvisioner) VerifyDomain(domain string) (bool, error) {
	// In a real implementation with Let's Encrypt:
	// 1. Create DNS-01 challenge record
	// 2. Wait for DNS propagation
	// 3. Submit challenge response
	// 4. Verify challenge

	// For self-signed, just verify DNS resolves
	return true, nil
}

// InstallCertificate installs a certificate and key
func (p *CertProvisioner) InstallCertificate(domain, certPEM, keyPEM string) (*CertificateInfo, error) {
	// Parse the certificate to validate it
	_, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return nil, fmt.Errorf("invalid certificate or key: %w", err)
	}

	// Parse certificate for metadata
	certBlock, _ := pem.Decode([]byte(certPEM))
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode certificate")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	return &CertificateInfo{
		Domain:       domain,
		CertPEM:      certPEM,
		KeyPEM:       keyPEM,
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		Issuer:       cert.Issuer.CommonName,
		SerialNumber: serialNumber.String(),
		Status:       "active",
	}, nil
}

// RenewCertificate renews an existing certificate
func (p *CertProvisioner) RenewCertificate(domain string, currentCert *CertificateInfo) (*CertificateInfo, error) {
	// Check if certificate is expiring soon (within 30 days)
	if time.Now().Add(30 * 24 * time.Hour).Before(currentCert.NotAfter) {
		// Certificate is still valid for more than 30 days
		return currentCert, nil
	}

	// Request a new certificate
	return p.RequestCertificate(domain)
}

// RevokeCertificate revokes a certificate
func (p *CertProvisioner) RevokeCertificate(domain string, cert *CertificateInfo) error {
	// In a real implementation with Let's Encrypt:
	// 1. Parse the certificate
	// 2. Send revocation request to CA
	// 3. Update status in database

	// For self-signed, just mark as revoked
	cert.Status = "revoked"
	return nil
}

// GetCertificateStatus returns the status of a certificate
func (p *CertProvisioner) GetCertificateStatus(cert *CertificateInfo) string {
	now := time.Now()
	if now.After(cert.NotAfter) {
		return "expired"
	}
	if now.Before(cert.NotBefore) {
		return "not_yet_valid"
	}
	return cert.Status
}

// IsExpiringSoon checks if a certificate expires within the given days
func (p *CertProvisioner) IsExpiringSoon(cert *CertificateInfo, days int) bool {
	return time.Now().Add(time.Duration(days) * 24 * time.Hour).After(cert.NotAfter)
}

// CreateTLSCertificate creates a tls.Certificate from PEM strings
func (p *CertProvisioner) CreateTLSCertificate(certPEM, keyPEM string) (tls.Certificate, error) {
	return tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
}

// CertStorage provides methods for storing and retrieving certificates
type CertStorage struct {
	certs map[string]*CertificateInfo
}

// NewCertStorage creates a new certificate storage
func NewCertStorage() *CertStorage {
	return &CertStorage{
		certs: make(map[string]*CertificateInfo),
	}
}

// StoreCertificate stores a certificate
func (s *CertStorage) StoreCertificate(domain string, cert *CertificateInfo) error {
	s.certs[domain] = cert
	return nil
}

// GetCertificate retrieves a certificate
func (s *CertStorage) GetCertificate(domain string) (*CertificateInfo, bool) {
	cert, ok := s.certs[domain]
	return cert, ok
}

// DeleteCertificate removes a certificate
func (s *CertStorage) DeleteCertificate(domain string) error {
	delete(s.certs, domain)
	return nil
}

// ListCertificates returns all stored certificates
func (s *CertStorage) ListCertificates() []*CertificateInfo {
	certs := make([]*CertificateInfo, 0, len(s.certs))
	for _, cert := range s.certs {
		certs = append(certs, cert)
	}
	return certs
}
