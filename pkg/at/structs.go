package at

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
			Domain:     "bzzzzzz.tech",
			CfApiToken: "kUSn8Q-SFT4-ISrWuZr16kNf5WHeSD7dZBs0alsy",
			HostingIP:  "176.9.70.14",
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

func (r *RemoteRepository) GetAllDomains() ([]AtDomain, error) {
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
