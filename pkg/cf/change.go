package cf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	CloudflareAPI = "https://api.cloudflare.com/client/v4"
)

type Client interface {
	GetZoneID(domain string) (string, error)
	GetDNSRecords(zoneID, recordType, name string) ([]DNSRecord, error)
	UpdateDNSRecord(zoneID, recordID, newIP string) error
	GetDomainIP(domain string) (string, error)
	UpdateDomainIP(domain, newIP string) error
}

type ApiClient struct {
	Token string

	Client
}

// DNSRecord represents a Cloudflare DNS record
type DNSRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl,omitempty"`
	Proxied bool   `json:"proxied,omitempty"`
}

// Zone represents a Cloudflare zone
type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CloudflareResponse is the standard response format from Cloudflare API
type CloudflareResponse struct {
	Success  bool              `json:"success"`
	Errors   []CloudflareError `json:"errors"`
	Messages []string          `json:"messages"`
	Result   json.RawMessage   `json:"result"`
}

// CloudflareError represents an error returned by Cloudflare API
type CloudflareError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewApiClient(token string) Client {
	return &ApiClient{
		Token: token,
	}
}

// newRequest creates a new HTTP request with authorization headers
func (c *ApiClient) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	fullUrl := fmt.Sprintf("%s%s", CloudflareAPI, url)

	req, err := http.NewRequest(method, fullUrl, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

// GetZoneID retrieves the zone ID for a domain
func (c *ApiClient) GetZoneID(domain string) (string, error) {
	// Ensure we're searching by the root domain
	rootDomain := domain
	if strings.Count(domain, ".") > 1 {
		parts := strings.Split(domain, ".")
		l := len(parts)
		rootDomain = fmt.Sprintf("%s.%s", parts[l-2], parts[l-1])
	}

	req, err := c.newRequest("GET", fmt.Sprintf("/zones?name=%s", rootDomain), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status code %d: %s", resp.StatusCode, body)
	}

	var cfResp CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if !cfResp.Success {
		return "", fmt.Errorf("API returned error: %+v", cfResp.Errors)
	}

	var zones []Zone
	if err := json.Unmarshal(cfResp.Result, &zones); err != nil {
		return "", fmt.Errorf("failed to unmarshal zones: %w", err)
	}

	if len(zones) == 0 {
		return "", fmt.Errorf("no zone found for domain %s", domain)
	}

	return zones[0].ID, nil
}

// GetDNSRecords retrieves DNS records of a specific type for a zone
func (c *ApiClient) GetDNSRecords(zoneID, recordType, name string) ([]DNSRecord, error) {
	url := fmt.Sprintf("/zones/%s/dns_records", zoneID)

	// Add filters if provided
	var params []string
	if recordType != "" {
		params = append(params, fmt.Sprintf("type=%s", recordType))
	}
	if name != "" {
		params = append(params, fmt.Sprintf("name=%s", name))
	}

	if len(params) > 0 {
		url = url + "?" + strings.Join(params, "&")
	}

	req, err := c.newRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status code %d: %s", resp.StatusCode, body)
	}

	var cfResp CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !cfResp.Success {
		return nil, fmt.Errorf("API returned error: %+v", cfResp.Errors)
	}

	var records []DNSRecord
	if err := json.Unmarshal(cfResp.Result, &records); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DNS records: %w", err)
	}

	return records, nil
}

// UpdateDNSRecord updates a DNS record with a new IP address
func (c *ApiClient) UpdateDNSRecord(zoneID, recordID, newIP string) error {
	url := fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, recordID)

	updateData := map[string]string{
		"content": newIP,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal update data: %w", err)
	}

	req, err := c.newRequest("PATCH", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status code %d: %s", resp.StatusCode, body)
	}

	var cfResp CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !cfResp.Success {
		return fmt.Errorf("API returned error: %+v", cfResp.Errors)
	}

	return nil
}

// GetDomainIP retrieves the current IP address of the A record for a domain
func (c *ApiClient) GetDomainIP(domain string) (string, error) {
	// Step 1: Get the zone ID for the domain
	zoneID, err := c.GetZoneID(domain)
	if err != nil {
		return "", fmt.Errorf("failed to get zone ID: %w", err)
	}

	// Step 2: Get the A record for the domain
	records, err := c.GetDNSRecords(zoneID, "A", domain)
	if err != nil {
		return "", fmt.Errorf("failed to get DNS records: %w", err)
	}

	if len(records) == 0 {
		return "", fmt.Errorf("no A record found for domain %s", domain)
	}

	// Return the IP address from the first matching A record
	for _, record := range records {
		if record.Type == "A" && record.Name == domain {
			return record.Content, nil
		}
	}

	return "", fmt.Errorf("no matching A record found for domain %s", domain)
}

// UpdateDomainIP updates the A record for a domain with a new IP address
func (c *ApiClient) UpdateDomainIP(domain, newIP string) error {
	// Step 1: Get the zone ID for the domain
	zoneID, err := c.GetZoneID(domain)
	if err != nil {
		return fmt.Errorf("failed to get zone ID: %w", err)
	}

	// Step 2: Get the A record for the domain
	records, err := c.GetDNSRecords(zoneID, "A", domain)
	if err != nil {
		return fmt.Errorf("failed to get DNS records: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("no A record found for domain %s", domain)
	}

	// Step 3: Update the A record with the new IP
	for _, record := range records {
		if record.Type == "A" && record.Name == domain {
			err = c.UpdateDNSRecord(zoneID, record.ID, newIP)
			if err != nil {
				return fmt.Errorf("failed to update DNS record: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("no matching A record found for domain %s", domain)
}
