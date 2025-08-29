package servers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"go-cf-zone-switch/pkg/config"
	"go-cf-zone-switch/pkg/db"
)

type ProxyConfigUpdater struct {
	Storage        *db.Storage
	UpdateInterval time.Duration
	Endpoint       string
	Notifier       Notifier
}

type Domain struct {
	Domain string `json:"domain"`
	IP     string `json:"ip"`
}

type Server struct {
	Address string
}

func NewProxyConfigUpdater(storage *db.Storage, config *config.Servers, notifier Notifier) *ProxyConfigUpdater {
	interval := time.Minute * time.Duration(config.DomainUpdateIntervalMin)

	return &ProxyConfigUpdater{
		Storage:        storage,
		UpdateInterval: interval,
		Endpoint:       config.DomainUpdateEndpoint,
		Notifier:       notifier,
	}
}

func (p *ProxyConfigUpdater) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(p.UpdateInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := p.updateDomains()
				if err != nil {
					log.Println("configurator: error updating domains", err)
				}

			case <-ctx.Done():
				log.Println("configurator: context canceled, stopping ")
				return
			}
		}
	}()
}

func (p *ProxyConfigUpdater) getAllDomains() ([]Domain, error) {
	log.Println("configurator: loading domains")
	domainsRows, err := p.Storage.GetAllDomains()
	if err != nil {
		return nil, err
	}
	domains := []Domain{}

	for _, dr := range domainsRows {
		if dr.HostingIP == "" {
			continue
		}
		domains = append(domains, Domain{
			Domain: dr.Domain,
			IP:     dr.HostingIP,
		})
	}

	return domains, nil
}

func (p *ProxyConfigUpdater) getActiveProxyServers() ([]Server, error) {
	log.Println("configurator: loading servers")

	serversRows, err := p.Storage.GetProxyServers(true)
	if err != nil {
		return nil, err
	}

	s := []Server{}
	for _, sr := range serversRows {
		s = append(s, Server{
			Address: net.JoinHostPort(sr.Host, sr.CheckPort),
		})
	}

	return s, nil
}

func (p *ProxyConfigUpdater) updateDomains() error {
	domains, err := p.getAllDomains()
	if err != nil {
		return err
	}

	servers, err := p.getActiveProxyServers()
	if err != nil {
		return err
	}
	fmt.Println(servers)
	for _, server := range servers {
		if err := p.sendUpdateToServer(server, domains); err != nil {
			// Log error and continue updating other servers
			log.Printf("Failed to update server %s: %v\n", server.Address, err)
		}
	}
	return nil
}

func newHttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // ⚠️ ОПАСНО: только для отладки! //nolint:gosec
		},
	}

	// Execute the POST request.
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 20,
	}

	return client
}

func (p *ProxyConfigUpdater) sendUpdateToServer(server Server, domains []Domain) error {
	log.Printf("configurator: sending domain updates to %s \n", server.Address)

	client := newHttpClient()

	batchSize := 100
	total := len(domains)
	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}
		batch := domains[i:end]

		// TODO: could be http or https?
		url := "http://" + server.Address + p.Endpoint

		// Marshal the batch of domains into JSON.
		payload, err := json.Marshal(batch)
		if err != nil {
			return fmt.Errorf("failed to marshal batch: %w", err)
		}

		// Create a POST request with appropriate headers.
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			_ = p.Notifier.Notify("Error sending update for domains: response code >= 300")
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		// Check if the server returned an error status.
		if resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			err := p.Notifier.Notify("Error sending update for domains: response code >= 300")
			if err != nil {
				return err
			}
			return fmt.Errorf("server returned status %s: %s", resp.Status, body)
		}
		break // TODO: remove
	}
	log.Printf("configurator: domain updates sent to server %s \n", server.Address)
	return nil
}
