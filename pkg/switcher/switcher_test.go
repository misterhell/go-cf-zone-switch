package switcher

import (
	"testing"

	"go-cf-zone-switch/pkg/cf"
	"go-cf-zone-switch/pkg/config"
	"go-cf-zone-switch/pkg/db"
	"go-cf-zone-switch/pkg/servers"
)

func TestSwitcher_ReceiveStatus_TriggersSwitch(t *testing.T) {
	failedIP := "failhost"
	newHostIP := "100.0.0.1"
	domain := "hello.com"

	mockStorage := &MockStorage{
		GetProxyServersFunc: func(onlyHealthy bool) ([]db.ProxyServerRow, error) {
			return []db.ProxyServerRow{{Host: newHostIP, IsUp: true}}, nil
		},
		GetDomainWithCfTokensFunc: func() ([]db.DomainRow, error) {
			return []db.DomainRow{{Domain: domain, CfApiToken: "token"}}, nil
		},
	}
	mockNotifier := &MockNotifier{}
	sw := NewSwitcher(&config.Config{}, mockStorage, mockNotifier)

	var mockClient *MockCfClient
	CFClientFactory := func(token string) cf.Client {
		mockClient = &MockCfClient{
			GetDomainIPFunc:    func(domain string) (string, error) { return failedIP, nil },
			UpdateDomainIPFunc: func(domain, newIP string) error { return nil },
		}
		return mockClient
	}

	sw.cfClientFactory = CFClientFactory
	sw.switchAfterFailureCount = 3

	// Simulate 3 failures for the same host
	status := servers.ServerStatus{Host: "failhost", IsUp: false}
	for i := 0; i < defaultSwitchAfterFailureCount; i++ {
		_ = sw.ReceiveStatus([]servers.ServerStatus{status})
	}

	// Assert that SaveProxyServers was called
	if !mockStorage.SaveProxyServersCalled {
		t.Error("SaveProxyServers was not called")
	}
	// Assert that notification was sent (if update fails)
	if len(mockNotifier.Messages) == 0 {
		t.Error("Expected notification to be sent")
	}

	if mockClient.DomainIpUpdatedTo != newHostIP {
		t.Errorf("Expected domain IP to be updated to %s, got %s", failedIP, mockClient.DomainIpUpdatedTo)
	}

	if mockClient.DomainUpdated != domain {
		t.Errorf("Expected domain to be updated to %s, got %s", domain, mockClient.DomainUpdated)
	}
}
