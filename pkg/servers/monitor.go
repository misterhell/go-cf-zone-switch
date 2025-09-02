package servers

import (
	"context"
	"log"
	"time"
)

type ServerStatus struct {
	ID        string
	Host      string
	Port      string
	IsUp      bool
	LastCheck time.Time
	Error     string
}

type StatusReceiver interface {
	ReceiveStatus(statuses []ServerStatus) error
}

type ServerMonitor struct {
	servers        []struct{ Host, Port, ID, Schema string }
	checkInterval  time.Duration
	timeout        time.Duration
	statusReceiver StatusReceiver
	notifier       Notifier
}

func NewServerMonitoring(checkInterval, timeout time.Duration, reporter StatusReceiver, notifier Notifier) *ServerMonitor {
	return &ServerMonitor{
		servers:        []struct{ Host, Port, ID, Schema string }{},
		checkInterval:  checkInterval,
		timeout:        timeout,
		statusReceiver: reporter,
		notifier:       notifier,
	}
}

func (m *ServerMonitor) AddServer(host, port, id, schema string) {
	m.servers = append(m.servers, struct {
		Host   string
		Port   string
		ID     string
		Schema string
	}{host, port, id, schema})
}

func (m *ServerMonitor) Start(ctx context.Context) {
	go func() {
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

	if err := m.statusReceiver.ReceiveStatus(statuses); err != nil {
		// m.notifier.Notify(" Failed to report server statuses")
		log.Printf("monitor: Failed to report server statuses: %v", err)
	}
}
