package cert

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"golang.org/x/crypto/acme"

	"os"
	"github.com/netbirdio/netbird/management/server/cloudflare"
)

var (
	CloudflareAPIToken = os.Getenv("CF_DNS_API_TOKEN")
	CloudflareZoneID   = os.Getenv("CF_ZONE_ID")
)
const (
	letsEncryptCA      = "https://acme-v02.api.letsencrypt.org/directory"
	dnsPropagationWait = 35 * time.Second
)

// CertificateManager manages SSL certificate generation via Let's Encrypt
type CertificateManager struct {
	cloudflareClient *cloudflare.CloudflareClient
	certDir          string
	acmeEmail        string
}

// CertResponse holds the generated certificate data
type CertResponse struct {
	Certificate        string
	PrivateKey         string
	CloudflareRecordID string
	RecordDeleted      bool
}

// NewCertificateManager creates a new CertificateManager
func NewCertificateManager(cfClient *cloudflare.CloudflareClient, certDir, email string) *CertificateManager {
	return &CertificateManager{
		cloudflareClient: cfClient,
		certDir:          certDir,
		acmeEmail:        email,
	}
}

// GeneratePeerCertificate generates a real Let's Encrypt SSL certificate
func (cm *CertificateManager) GeneratePeerCertificate(ctx context.Context, domain, email string) (*CertResponse, error) {
	if email == "" {
		email = cm.acmeEmail
	}

	// 1. Generate ACME account key
	accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate account key: %w", err)
	}

	// 2. Create ACME v2 client
	client := &acme.Client{
		Key:          accountKey,
		DirectoryURL: letsEncryptCA,
	}

	// 3. Register ACME account
	acc := &acme.Account{Contact: []string{"mailto:" + email}}
	if _, err := client.Register(ctx, acc, acme.AcceptTOS); err != nil {
		return nil, fmt.Errorf("failed to register ACME account: %w", err)
	}

	// 4. Create certificate order
	order, err := client.AuthorizeOrder(ctx, acme.DomainIDs(domain))
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// 5. Process authorizations
	var recordID string
	for _, authzURL := range order.AuthzURLs {
		authz, err := client.GetAuthorization(ctx, authzURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get authorization: %w", err)
		}

		if authz.Status == acme.StatusValid {
			continue
		}

		// Find DNS-01 challenge
		var challenge *acme.Challenge
		for _, c := range authz.Challenges {
			if c.Type == "dns-01" {
				challenge = c
				break
			}
		}
		if challenge == nil {
			return nil, fmt.Errorf("no DNS-01 challenge found for %s", authz.Identifier.Value)
		}

		// Get TXT record value
		txtValue, err := client.DNS01ChallengeRecord(challenge.Token)
		if err != nil {
			return nil, fmt.Errorf("failed to get DNS challenge value: %w", err)
		}

		// Create TXT record on Cloudflare
		recordID, err = cm.cloudflareClient.CreateDNSRecord(ctx, authz.Identifier.Value, "TXT", txtValue)
		if err != nil {
			return nil, fmt.Errorf("failed to create DNS TXT record: %w", err)
		}

		// Wait for DNS propagation
		time.Sleep(dnsPropagationWait)

		// Accept challenge
		if _, err := client.Accept(ctx, challenge); err != nil {
			_ = cm.cloudflareClient.DeleteDNSRecord(ctx, recordID)
			return nil, fmt.Errorf("failed to accept challenge: %w", err)
		}

		// Wait for authorization
		if _, err := client.WaitAuthorization(ctx, authzURL); err != nil {
			_ = cm.cloudflareClient.DeleteDNSRecord(ctx, recordID)
			return nil, fmt.Errorf("authorization failed: %w", err)
		}
	}

	// 6. Generate certificate private key
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		_ = cm.cloudflareClient.DeleteDNSRecord(ctx, recordID)
		return nil, fmt.Errorf("failed to generate certificate key: %w", err)
	}

	// 7. Create CSR
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		DNSNames: []string{domain},
	}, certKey)
	if err != nil {
		_ = cm.cloudflareClient.DeleteDNSRecord(ctx, recordID)
		return nil, fmt.Errorf("failed to create CSR: %w", err)
	}

	// 8. Finalize order and get certificate
	derCerts, _, err := client.CreateOrderCert(ctx, order.FinalizeURL, csr, true)
	if err != nil {
		_ = cm.cloudflareClient.DeleteDNSRecord(ctx, recordID)
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// 9. Encode certificate chain to PEM
	var certPEM string
	for _, der := range derCerts {
		certPEM += string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: der,
		}))
	}

	// 10. Encode private key to PEM
	keyBytes, err := x509.MarshalECPrivateKey(certKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal key: %w", err)
	}
	keyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	}))

	// 11. Clean up DNS record
	_ = cm.cloudflareClient.DeleteDNSRecord(ctx, recordID)

	return &CertResponse{
		Certificate:        certPEM,
		PrivateKey:         keyPEM,
		CloudflareRecordID: recordID,
		RecordDeleted:      true,
	}, nil
}
