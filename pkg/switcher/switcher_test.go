package switcher

import (
	"fmt"
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
		SaveProxyServersFunc: func(rows []db.ProxyServerRow) error {
			return nil
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

func TestSwitcher_ReceiveStatus_UpdateDomainError(t *testing.T) {
	failedIP := "failhost"
	newHostIP := "100.0.0.2"
	domain := "error.com"
	updateErr := fmt.Errorf("update error")

	mockStorage := &MockStorage{
		GetProxyServersFunc: func(onlyHealthy bool) ([]db.ProxyServerRow, error) {
			return []db.ProxyServerRow{{Host: newHostIP, IsUp: true}}, nil
		},
		GetDomainWithCfTokensFunc: func() ([]db.DomainRow, error) {
			return []db.DomainRow{{Domain: domain, CfApiToken: "token"}}, nil
		},
		SaveProxyServersFunc: func(rows []db.ProxyServerRow) error {
			return nil
		},
	}
	mockNotifier := &MockNotifier{}
	sw := NewSwitcher(&config.Config{}, mockStorage, mockNotifier)

	var mockClient *MockCfClient
	sw.cfClientFactory = func(token string) cf.Client {
		mockClient = &MockCfClient{
			GetDomainIPFunc: func(dom string) (string, error) {
				return failedIP, nil
			},
			UpdateDomainIPFunc: func(dom, newIP string) error {
				return updateErr
			},
		}
		return mockClient
	}

	// Set a lower failure count threshold for testing.
	sw.switchAfterFailureCount = 3

	// Trigger failures for the same host to invoke domain update.
	status := servers.ServerStatus{Host: failedIP, IsUp: false}
	for i := 0; i < sw.switchAfterFailureCount; i++ {
		_ = sw.ReceiveStatus([]servers.ServerStatus{status})
	}

	// Assert that SaveProxyServers was called.
	if !mockStorage.SaveProxyServersCalled {
		t.Error("SaveProxyServers was not called")
	}

	// Assert that a notification about the update error was sent.
	expectedMsg := fmt.Sprintf("Failed to update domain %s: %v", domain, updateErr)
	found := false
	for _, msg := range mockNotifier.Messages {
		if msg == expectedMsg {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected notification '%s' to be sent", expectedMsg)
	}
}

func TestSwitcher_ReceiveStatus_EmptyHealthyServers(t *testing.T) {
	failedIP := "failhost"
	domain := "test.com"

	mockStorage := &MockStorage{
		GetProxyServersFunc: func(onlyHealthy bool) ([]db.ProxyServerRow, error) {
			// Return an empty list to simulate no healthy servers.
			return []db.ProxyServerRow{}, nil
		},
		GetDomainWithCfTokensFunc: func() ([]db.DomainRow, error) {
			// Return a non-empty domain list.
			return []db.DomainRow{{Domain: domain, CfApiToken: "token"}}, nil
		},
		SaveProxyServersFunc: func(rows []db.ProxyServerRow) error {
			return nil
		},
	}
	mockNotifier := &MockNotifier{}
	sw := NewSwitcher(&config.Config{}, mockStorage, mockNotifier)
	// Lower threshold for testing.
	sw.switchAfterFailureCount = 2

	// Simulate failures to trigger switch.
	status := servers.ServerStatus{Host: failedIP, IsUp: false}
	for i := 0; i < sw.switchAfterFailureCount; i++ {
		_ = sw.ReceiveStatus([]servers.ServerStatus{status})
	}

	// Verify that a "No healthy server found" notification was sent.
	found := false
	for _, msg := range mockNotifier.Messages {
		if msg == "No healthy server found" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'No healthy server found' notification to be sent")
	}

	// TODO: add check for message sent to notifier

	if !mockStorage.SaveProxyServersCalled {
		t.Error("SaveProxyServers was not called")
	}
}

func TestSwitcher_ReceiveStatus_EmptyDomains(t *testing.T) {
	failedIP := "failhost"
	newHostIP := "100.0.0.3"

	mockStorage := &MockStorage{
		GetProxyServersFunc: func(onlyHealthy bool) ([]db.ProxyServerRow, error) {
			// Return a healthy server.
			return []db.ProxyServerRow{{Host: newHostIP, IsUp: true}}, nil
		},
		GetDomainWithCfTokensFunc: func() ([]db.DomainRow, error) {
			// Return an empty list to simulate no domains.
			return []db.DomainRow{}, nil
		},
		SaveProxyServersFunc: func(rows []db.ProxyServerRow) error {
			return nil
		},
	}
	mockNotifier := &MockNotifier{}
	sw := NewSwitcher(&config.Config{}, mockStorage, mockNotifier)
	// Lower threshold for testing.
	sw.switchAfterFailureCount = 2

	// Flag to track if CF client factory gets invoked.
	mockClientCalled := false
	sw.cfClientFactory = func(token string) cf.Client {
		mockClientCalled = true
		return &MockCfClient{
			GetDomainIPFunc: func(dom string) (string, error) {
				return failedIP, nil
			},
			UpdateDomainIPFunc: func(dom, newIP string) error {
				return nil
			},
		}
	}

	// Simulate failures to trigger domain update logic.
	status := servers.ServerStatus{Host: failedIP, IsUp: false}
	for i := 0; i < sw.switchAfterFailureCount; i++ {
		_ = sw.ReceiveStatus([]servers.ServerStatus{status})
	}

	// Since there are no domains, CF client should not be invoked.
	if mockClientCalled {
		t.Error("CF client factory should not be called when there are no domains")
	}

	if !mockStorage.SaveProxyServersCalled {
		t.Error("SaveProxyServers was not called")
	}

	// No notifications should be sent because there are no domain updates attempted.
	if len(mockNotifier.Messages) > 0 {
		t.Errorf("Expected no notifications to be sent, got %d", len(mockNotifier.Messages))
	}
}
