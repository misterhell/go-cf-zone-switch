package servers

import (
	"context"
	"log"
	"sync"
	"time"
)


type ServerStatus struct {
	ID string
	Host string
	Port string
	IsUp bool
	LastCheck time.Time
	Error string
}

type StatusReporter interface {
    ReportStatus(statuses []ServerStatus) error
}

type ServerMonitor struct {
	servers []struct{ Host, Port, ID string}
	checkInterval time.Duration
	timeout time.Duration
	reporter StatusReporter
	// notifier Notifier
	stopCh chan struct{}
	wg sync.WaitGroup
}

func NewServerMonitoring(checkInterval, timeout time.Duration, reporter StatusReporter, 
	// notifier Notifier
	) *ServerMonitor {
	return &ServerMonitor{
		servers:  []struct{ Host, Port, ID string}{},
		checkInterval: checkInterval,
		timeout: timeout,
		reporter: reporter,
		// notifier: notifier,
		stopCh: make(chan struct{}),
	}
}

func (m *ServerMonitor) AddServer(host, port, id string)  {
	m.servers = append(m.servers, struct{Host string; Port string; ID string}{host, port, id})
}

func (m *ServerMonitor) Start(ctx context.Context) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(m.checkInterval)
		defer ticker.Stop()

		m.checkServers(ctx)

		for {
			select {
			case <-ticker.C:
				m.checkServers(ctx)
			case <-ctx.Done():
				log.Println("monitor: stopped due to context cancellation")
				return
			case <-m.stopCh:
				log.Println("monitor: stopped")
				return
			}
		}
	}()	
} 


func (m *ServerMonitor) checkServers(ctx context.Context) {
    if len(m.servers) == 0 {
        log.Println("monitor: No servers configured for monitoring")
        return
    }

    statuses := make([]ServerStatus, 0, len(m.servers))
    
    for _, server := range m.servers {
        select {
        case <-ctx.Done():
            return
        default:
            status := ServerStatus{
                Host:      server.Host,
                Port:      server.Port,
                LastCheck: time.Now(),
            }
            
            isUp, err := IsServerReachable(server.Host, server.Port, m.timeout)
            status.IsUp = isUp
            
            if err != nil {
                status.Error = err.Error()
            }
            
            statuses = append(statuses, status)
        }
    }
    
    if err := m.reporter.ReportStatus(statuses); err != nil {
		// m.notifier.Notify(" Failed to report server statuses")
        log.Printf("monitor: Failed to report server statuses: %v", err)
    }
}