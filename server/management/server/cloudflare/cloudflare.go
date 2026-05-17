package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	cloudflareAPIBase = "https://api.cloudflare.com/client/v4"
)

// CloudflareClient handles communication with Cloudflare API
type CloudflareClient struct {
	APIKey    string
	Email     string
	ZoneID    string
	BaseURL   string
	HTTPClient *http.Client
}

// NewCloudflareClient creates a new Cloudflare client
func NewCloudflareClient(apiKey, email, zoneID string) *CloudflareClient {
	return &CloudflareClient{
		APIKey:   apiKey,
		Email:    email,
		ZoneID:   zoneID,
		BaseURL:  cloudflareAPIBase,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DNSRecord represents a DNS record
type DNSRecord struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int64  `json:"ttl"`
	Proxied  bool   `json:"proxied"`
	ZoneID   string `json:"zone_id"`
	ZoneName string `json:"zone_name"`
}


// DNSRecordSingleResponse represents the Cloudflare API response for a single DNS record
type DNSRecordSingleResponse struct {
	Success bool       `json:"success"`
	Errors  []APIError `json:"errors"`
	Result  DNSRecord  `json:"result"`
}

// DNSRecordResponse represents the Cloudflare API response for DNS records
type DNSRecordResponse struct {
	Success bool       `json:"success"`
	Errors  []APIError `json:"errors"`
	Results []DNSRecord `json:"results"`
	Meta    APIResponseMeta `json:"meta"`
}

// APIError represents a Cloudflare API error
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// APIResponseMeta represents Cloudflare API metadata
type APIResponseMeta struct {
	Requested   string `json:"requested"`
	Count       int    `json:"count"`
	Page        int    `json:"page"`
	PerPage     int    `json:"per_page"`
	TotalPages  int    `json:"total_pages"`
	TotalCount  int    `json:"total_count"`
}

// CreateDNSRecord creates a DNS record on Cloudflare
func (c *CloudflareClient) CreateDNSRecord(ctx context.Context, domain, recordType, recordValue string) (string, error) {
	// Generate the challenge subdomain
	challengeDomain := fmt.Sprintf("_acme-challenge.%s", domain)

	url := fmt.Sprintf("%s/zones/%s/dns_records", c.BaseURL, c.ZoneID)

	record := map[string]interface{}{
		"type":    "TXT",
		"name":    challengeDomain,
		"content": recordValue,
		"ttl":     120, // 2 minutes for ACME challenge
		"proxied": false,
	}

	jsonData, err := json.Marshal(record)
	if err != nil {
		return "", fmt.Errorf("failed to marshal record: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cloudflare API error (status %d): %s", resp.StatusCode, string(body))
	}

	var dnsResp DNSRecordSingleResponse
	if err := json.Unmarshal(body, &dnsResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !dnsResp.Success {
		if len(dnsResp.Errors) > 0 {
			return "", fmt.Errorf("cloudflare API error: %s", dnsResp.Errors[0].Message)
		}
		return "", fmt.Errorf("cloudflare API error: unknown error")
	}

	if dnsResp.Result.ID == "" {
		return "", fmt.Errorf("no record created")
	}

	return dnsResp.Result.ID, nil
}

// DeleteDNSRecord deletes a DNS record from Cloudflare
func (c *CloudflareClient) DeleteDNSRecord(ctx context.Context, recordID string) error {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.BaseURL, c.ZoneID, recordID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete record: HTTP %d", resp.StatusCode)
	}

	return nil
}

// VerifyDNSRecord verifies if a DNS record exists and has the correct value
func (c *CloudflareClient) VerifyDNSRecord(ctx context.Context, recordID, expectedValue string) (bool, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.BaseURL, c.ZoneID, recordID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to get record: HTTP %d", resp.StatusCode)
	}

	var dnsResp DNSRecordResponse
	if err := json.Unmarshal(body, &dnsResp); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	if !dnsResp.Success || len(dnsResp.Results) == 0 {
		return false, fmt.Errorf("record not found")
	}

	record := dnsResp.Results[0]
	if record.Content == expectedValue {
		return true, nil
	}

	return false, fmt.Errorf("record content mismatch: got %s, expected %s", record.Content, expectedValue)
}

// GetDNSRecordsByDomain gets all DNS records for a domain
func (c *CloudflareClient) GetDNSRecordsByDomain(ctx context.Context, domain string) ([]DNSRecord, error) {
	challengeDomain := fmt.Sprintf("_acme-challenge.%s", domain)
	url := fmt.Sprintf("%s/zones/%s/dns_records?name=%s", c.BaseURL, c.ZoneID, challengeDomain)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get records: HTTP %d", resp.StatusCode)
	}

	var dnsResp DNSRecordResponse
	if err := json.Unmarshal(body, &dnsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !dnsResp.Success {
		if len(dnsResp.Errors) > 0 {
			return nil, fmt.Errorf("cloudflare API error: %s", dnsResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("cloudflare API error: unknown error")
	}

	return dnsResp.Results, nil
}