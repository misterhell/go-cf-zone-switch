package at

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

const (
	API_V0            = "https://api.airtable.com/v0"
	fieldApiKey       = "API Key CF (from Domain)"
	fieldDomainReqIDs = "Domain"

	fieldsDomainTblDomain = "Domain"
)

type Client struct {
	cfg AtConfig
}

func NewClient(cfg AtConfig) *Client {
	return &Client{
		cfg: cfg,
	}
}

func (c *Client) makeRequest(reqType, tbl, view string) (*http.Request, error) {
	url := fmt.Sprintf("%s/%s/%s", API_V0, c.cfg.GetBase(), tbl)

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

func (c *Client) GetAllRecords() ([]Record, error) {
	records := []Record{}

	req, err := c.makeRequest("GET", c.cfg.GetAccountTable(), c.cfg.GetAccountView())
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Panicln(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("api returned 404 code")
		}

		var errorResp ErrorResponse

		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error response code %d, %+v", resp.StatusCode, errorResp)
	}

	var airtableResp AirtableResponse
	if err := json.NewDecoder(resp.Body).Decode(&airtableResp); err != nil {
		return nil, err
	}
	records = append(records, airtableResp.Records...)

	offset := airtableResp.Offset
	for offset != "" {
		// Create a new request with the offset parameter added in the query string.
		req, err := c.makeRequest("GET", c.cfg.GetAccountTable(), c.cfg.GetAccountView())
		if err != nil {
			return nil, err
		}

		// Append the offset parameter to the URL query.
		query := req.URL.Query()
		query.Set("offset", offset)
		req.URL.RawQuery = query.Encode()

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := resp.Body.Close()
			if err != nil {
				log.Panicln(err)
			}
		}()

		var airtableResp AirtableResponse
		if err := json.NewDecoder(resp.Body).Decode(&airtableResp); err != nil {
			return nil, err
		}
		records = append(records, airtableResp.Records...)
		offset = airtableResp.Offset
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

    req, err := c.makeRequest("GET", c.cfg.GetDomainsTable(), "")
    if err != nil {
        return nil, err
    }

    q := req.URL.Query()

    var parts []string
    for _, id := range reqIDs {
        parts = append(parts, fmt.Sprintf("RECORD_ID()='%s'", id))
    }
    formula := "OR(" + strings.Join(parts, ",") + ")"

    q.Add("filterByFormula", formula)
    q.Add("fields[]", fieldsDomainTblDomain)

    req.URL.RawQuery = q.Encode()

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        if resp.StatusCode == http.StatusNotFound {
            return nil, fmt.Errorf("api returned 404 code requesting domain")
        }

        var errorResp ErrorResponse

        if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
            return nil, err
        }

        return nil, fmt.Errorf("error response code %d, %+v", resp.StatusCode, errorResp)
    }

    var airtableResp AirtableResponse
    if err := json.NewDecoder(resp.Body).Decode(&airtableResp); err != nil {
        return nil, err
    }

    for _, record := range airtableResp.Records {
        if domain, ok := record.Fields[fieldsDomainTblDomain].(string); ok {
            result[record.ID] = domain
        }
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
