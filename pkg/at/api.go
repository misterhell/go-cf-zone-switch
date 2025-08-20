package at

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	apiV0             = "https://api.airtable.com/v0"
	fieldApiKey       = "API Key CF (from Domain)"
	fieldDomainReqIDs = "Domain"
	fieldHostingReqIDs = "Hosting"

	fieldsDomainTblDomain = "Domain"
)

type Client struct {
	cfg        AtConfig
	httpClient *http.Client
}

func NewClient(cfg AtConfig) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

func (c *Client) makeRequest(reqType, tbl, view string) (*http.Request, error) {
	url := fmt.Sprintf("%s/%s/%s", apiV0, c.cfg.GetBase(), tbl)

	req, err := http.NewRequest(reqType, url, nil)

	if view != "" {
		q := req.URL.Query()
		q.Add("view", view)
		req.URL.RawQuery = q.Encode()
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.cfg.GetApiToken()))

	log.Printf("sending request: %s?%s \n", url, req.URL.RawQuery)
	return req, err
}

// AirtableResponse represents the standard response from Airtable.
type AirtableResponse struct {
	Records []Record `json:"records"`
	Offset  string   `json:"offset,omitempty"`
}

// Record represents a single Airtable record.
type Record struct {
	ID          string                 `json:"id"`
	Fields      map[string]interface{} `json:"fields"`
	CreatedTime string                 `json:"createdTime"`
}

// It returns the value as a string if present, or an error if missing or not a string.
func (r *Record) GetAPIKeyCF() string {
	value, exists := r.Fields[fieldApiKey]
	if !exists {
		return ""
	}
	switch vv := value.(type) {
	case []interface{}:
		for _, v := range vv {
			if s, ok := v.(string); ok {
				return s
			}
		}
	case []string:
		for _, s := range vv {
			return s
		}
	default:
		log.Printf("unexpected type: %T\n", value)
	}

	return ""
}

func (r *Record) GetDomainsReqIDs() []string {
	val, exists := r.Fields[fieldDomainReqIDs]

	if exists {
		if dSlice, ok := val.([]interface{}); ok {
			domains := make([]string, len(dSlice))
			for i, v := range dSlice {
				if s, ok := v.(string); ok {
					domains[i] = s
				}
			}
			return domains
		}
	}

	return []string{}
}

// ErrorResponse represents the API error response.
type ErrorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// handleResponse is a helper method to process HTTP responses and decode JSON data
func (c *Client) handleResponse(resp *http.Response, result interface{}) error {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("api returned 404 code")
		}

		var errorResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return fmt.Errorf("failed to decode error response: %w", err)
		}
		return fmt.Errorf("api error: %+v", errorResp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// fetchPageOpts contains options for fetching a page of records
type fetchPageOpts struct {
	Table  string
	View   string
	Offset string
	Params map[string][]string // Additional query parameters
}

// fetchPage fetches a single page of records with optional parameters
func (c *Client) fetchPage(opts fetchPageOpts) (*AirtableResponse, error) {
	req, err := c.makeRequest("GET", opts.Table, opts.View)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Get existing query parameters
	query := req.URL.Query()

	// Add offset if provided
	if opts.Offset != "" {
		query.Set("offset", opts.Offset)
	}

	// Add additional parameters
	for key, values := range opts.Params {
		for _, value := range values {
			query.Add(key, value)
		}
	}

	req.URL.RawQuery = query.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	var airtableResp AirtableResponse
	if err := c.handleResponse(resp, &airtableResp); err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}

	return &airtableResp, nil
}

func (c *Client) GetAllRecords() ([]Record, error) {
	var records []Record
	var offset string

	for {
		page, err := c.fetchPage(fetchPageOpts{
			Table:  c.cfg.GetAccountTable(),
			View:   c.cfg.GetAccountView(),
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch records: %w", err)
		}

		records = append(records, page.Records...)

		// If no more pages, break the loop
		if page.Offset == "" {
			break
		}

		offset = page.Offset
	}

	return records, nil
}

func (c *Client) GetDomain(reqID string) (string, error) {
	domains, err := c.multiDomainRequest([]string{reqID})
	if err != nil {
		return "", err
	}

	if len(domains) > 0 {
		return domains[reqID], nil
	}

	return "", fmt.Errorf("no data")
}

func (c *Client) multiDomainRequest(reqIDs []string) (map[string]string, error) {
	result := make(map[string]string)

	// Prepare the filter formula
	var parts []string
	for _, id := range reqIDs {
		parts = append(parts, fmt.Sprintf("RECORD_ID()='%s'", id))
	}
	formula := "OR(" + strings.Join(parts, ",") + ")"

	var offset string

	// Fetch all pages
	params := map[string][]string{
		"filterByFormula": {formula},
		"fields[]":        {fieldsDomainTblDomain},
	}

	for {
		page, err := c.fetchPage(fetchPageOpts{
			Table:  c.cfg.GetDomainsTable(),
			Params: params,
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch domains page: %w", err)
		}

		// Process records from this page
		for _, record := range page.Records {
			if domain, ok := record.Fields[fieldsDomainTblDomain].(string); ok {
				result[record.ID] = domain
			}
		}

		// If no more pages, break the loop
		if page.Offset == "" {
			break
		}

		offset = page.Offset
	}

	return result, nil
}

// return map of reqIDs to domains
func (c *Client) GetDomains(reqIDs []string) (map[string]string, error) {
	domains, err := c.multiDomainRequest(reqIDs)
	if err != nil {
		return nil, err
	}

	return domains, nil
}
