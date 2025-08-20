package at

type Repository interface {
	GetAllDomains() ([]AtDomain, error)
}

type AtDomain struct {
	Domain string
	Token  string
	// HostingIP string
}

type LocalRepository struct {
	Repository
}

func (l *LocalRepository) GetAllDomains() ([]AtDomain, error) {
	return []AtDomain{
		{
			"bzzzzzz.tech",
			"kUSn8Q-SFT4-ISrWuZr16kNf5WHeSD7dZBs0alsy",
			// "176.9.70.14",
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
	records, err := r.client.GetAllRecords()
	if err != nil {
		return nil, err
	}

	// Collect all domain request IDs and map record ID to API key
	var allDomainReqIDs []string
	recordIDToAPIKey := make(map[string]string)
	for _, rec := range records {
		ids := rec.GetDomainsReqIDs()
		allDomainReqIDs = append(allDomainReqIDs, ids...)
		recordIDToAPIKey[rec.ID] = rec.GetAPIKeyCF()
	}

	// Request all domains in one batch
	domainIDToDomain, err := r.client.GetDomains(allDomainReqIDs)
	if err != nil {
		return nil, err
	}

	atDomains := []AtDomain{}
	for recordID, domain := range domainIDToDomain {
		apiKey := ""
		// Find which record this domain belongs to
		for _, rec := range records {
			ids := rec.GetDomainsReqIDs()
			for _, id := range ids {
				if id == recordID {
					apiKey = rec.GetAPIKeyCF()
					break
				}
			}
			if apiKey != "" {
				break
			}
		}
		atDomains = append(atDomains, AtDomain{
			Domain: domain,
			Token:  apiKey,
		})
	}

	return atDomains, nil
}
