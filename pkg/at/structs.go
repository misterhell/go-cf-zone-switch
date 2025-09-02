package at

import (
	"os"
	"regexp"
)

type Repository interface {
	GetAllDomains() ([]AtDomain, error)
}

type AtDomain struct {
	Domain     string
	CfApiToken string
	HostingIP  string
}

type LocalRepository struct {
	Repository
}

func (l *LocalRepository) GetAllDomains() ([]AtDomain, error) {
	return []AtDomain{
		{
			Domain:     os.Getenv("LOCAL_DOMAIN"),
			CfApiToken: os.Getenv("LOCAL_CF_API_TOKEN"),
			HostingIP:  os.Getenv("LOCAL_HOSTING"),
		},
	}, nil
}

func NewLocalRepository() *LocalRepository {
	return &LocalRepository{}
}

type RemoteRepository struct {
	client *Client
	Repository
}

func NewRemoteRepository(cfg AtConfig) *RemoteRepository {
	client := NewClient(cfg)
	return &RemoteRepository{
		client: client,
	}
}

func (r *RemoteRepository) GetAllDomainsForIpChange() ([]AtDomain, error) {
	accountsRecords, err := r.client.FetchAllAccountRecords()
	if err != nil {
		return nil, err
	}

	// Collect all domain request IDs and map record ID to API key
	var allDomainReqIDs []string
	recordIDToAPIToken := make(map[string]string)
	for _, rec := range accountsRecords {
		ids := rec.getDomainsReqIDs()
		allDomainReqIDs = append(allDomainReqIDs, ids...)
		recordIDToAPIToken[rec.ID] = rec.CfApiToken
	}

	// Request all domains with their hosting information
	domainsData, err := r.client.GetDomains(allDomainReqIDs)
	if err != nil {
		return nil, err
	}

	hostingIDs := []string{}

	for _, domain := range domainsData {
		hostingIDs = append(hostingIDs, domain.HostingID)
	}

	hostingRecIdsToIPs, err := r.client.GetHostingByIds(hostingIDs)
	if err != nil {
		return nil, err
	}

	atDomains := []AtDomain{}

	for _, accountRecord := range accountsRecords {
		for _, domainID := range accountRecord.DomainsRecordsIDs {
			if domainData, ok := domainsData[domainID]; ok {
				atDomain := AtDomain{
					CfApiToken: accountRecord.CfApiToken,
					Domain:     domainData.Domain,
				}

				if hostingIP, ok := hostingRecIdsToIPs[domainData.HostingID]; ok {
					atDomain.HostingIP = hostingIP
				}

				atDomains = append(atDomains, atDomain)
			}
		}
	}

	return atDomains, nil
}

func (r *RemoteRepository) GetAllDomains() ([]AtDomain, error) {
	domainsData, err := r.client.FetchAllDomains()
	if err != nil {
		return nil, err
	}

	hostingIDsMap := map[string]bool{}

	for _, domain := range domainsData {
		hostingIDsMap[domain.HostingID] = true
	}

	hostingIDs := []string{}
	for ID := range hostingIDsMap {
		hostingIDs = append(hostingIDs, ID)
	}

	hostingRecIdsToIPs, err := r.client.GetHostingByIds(hostingIDs)
	if err != nil {
		return nil, err
	}

	atDomains := []AtDomain{}
	for _, domain := range domainsData {
		hostingIP := ""

		if ip, ok := hostingRecIdsToIPs[domain.HostingID]; ok {
			hostingIP = ip
		}

		re := regexp.MustCompile(`[^a-zA-Z0-9.-]`)
		cleanDomain := re.ReplaceAllString(domain.Domain, "")
		atDomains = append(atDomains, AtDomain{
			Domain:     cleanDomain,
			HostingIP:  hostingIP,
			CfApiToken: domain.CfApiToken,
		})
	}

	return atDomains, nil
}
