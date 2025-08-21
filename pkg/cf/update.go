package cf


// UpdateDomainsIP updates A records for multiple domains with a new IP address
// domainTokens is a map where key is the domain and value is the Cloudflare API token
// newIP is the new IP address to set for all domains
// Returns a map of domain to error for any domains that failed to update
func UpdateDomainsIP(domainTokens map[string]string, newIP string) map[string]error {
	results := make(map[string]error)

	for domain, token := range domainTokens {
		client := NewClient(token)
		err := client.UpdateDomainIP(domain, newIP)
		results[domain] = err
	}

	return results
}