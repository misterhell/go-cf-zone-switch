package switcher

import (
	"go-cf-zone-switch/pkg/cf"
	"go-cf-zone-switch/pkg/db"
)

type MockCfClient struct {
	DomainIpUpdatedTo  string
	DomainUpdated      string
	GetDomainIPFunc    func(domain string) (string, error)
	UpdateDomainIPFunc func(domain, newIP string) error

	cf.Client
}



func (m *MockCfClient) GetZoneID(domain string) (string, error) {
	return "mock-zone-id", nil
}

func (m *MockCfClient) GetDNSRecords(zoneID, recordType, name string) ([]cf.DNSRecord, error) {
	// Return an empty slice of DNSRecord in the mock.
	return []cf.DNSRecord{}, nil
}

func (m *MockCfClient) UpdateDNSRecord(zoneID, recordID, newIP string) error {
	// Simulate successful update.
	return nil
}

func (m *MockCfClient) GetDomainIP(domain string) (string, error) {
	if m.GetDomainIPFunc != nil {
		return m.GetDomainIPFunc(domain)
	}
	return "", nil
}

func (m *MockCfClient) UpdateDomainIP(domain, newIP string) error {
	if m.UpdateDomainIPFunc != nil {
		m.DomainIpUpdatedTo = newIP
		m.DomainUpdated = domain
		return m.UpdateDomainIPFunc(domain, newIP)
	}
	return nil
}

type MockNotifier struct {
	Messages []string
}

func (m *MockNotifier) Notify(msg string) error {
	m.Messages = append(m.Messages, msg)
	return nil
}

type MockStorage struct {
	SaveProxyServersCalled      bool
	GetProxyServersCalled       bool
	GetProxyServersFunc         func(onlyHealthy bool) ([]db.ProxyServerRow, error)
	GetDomainWithCfTokensCalled bool
	GetDomainWithCfTokensFunc   func() ([]db.DomainRow, error)
}

func (m *MockStorage) SaveProxyServers(rows []db.ProxyServerRow) error {
	m.SaveProxyServersCalled = true
	return nil
}

func (m *MockStorage) GetProxyServers(onlyHealthy bool) ([]db.ProxyServerRow, error) {
	m.GetProxyServersCalled = true
	return m.GetProxyServersFunc(onlyHealthy)
}

func (m *MockStorage) GetDomainWithCfTokens() ([]db.DomainRow, error) {
	m.GetDomainWithCfTokensCalled = true
	return m.GetDomainWithCfTokensFunc()
}
func (m *MockStorage) Close() {}

func (m *MockStorage) SaveDomains(rows []db.DomainRow) error {
	return nil
}

func (m *MockStorage) GetAllDomains() ([]db.DomainRow, error) {
	return nil, nil
}
