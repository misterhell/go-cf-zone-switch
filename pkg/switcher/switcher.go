package switcher

import (
	"fmt"
	"log"
	"sync"

	"go-cf-zone-switch/pkg/cf"
	"go-cf-zone-switch/pkg/config"
	"go-cf-zone-switch/pkg/db"
	"go-cf-zone-switch/pkg/servers"
)

type Switcher struct {
	storage                 *db.Storage
	notifier                Notifier
	switchAfterFailureCount int

	failureCounts map[string]int // key: Host
	mu            sync.Mutex

	servers.StatusReceiver
}

func NewSwitcher(config *config.Config, storage *db.Storage, notifier Notifier) *Switcher {
	return &Switcher{
		storage:       storage,
		notifier:      notifier,
		failureCounts: make(map[string]int),
	}
}

func (r *Switcher) ReceiveStatus(statuses []servers.ServerStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	serverRows := []db.ProxyServerRow{}
	for _, s := range statuses {
		log.Printf("switcher: Report received %+v\n", s)

		// Track consecutive failures
		if !s.IsUp {
			r.failureCounts[s.Host]++
		} else {
			r.failureCounts[s.Host] = 0
		}

		// If failed 3 times, trigger switch
		if r.failureCounts[s.Host] >= r.switchAfterFailureCount {
			log.Printf("switcher: Host %s failed %d times, switching domains...", s.Host, r.switchAfterFailureCount)
			healthy, err := r.selectHealthyServer()
			if err != nil {
				log.Printf("switcher: No healthy server found: %v", err)
			} else {
				r.changeDomainsTo(healthy)
			}
			r.failureCounts[s.Host] = 0 // reset after switch
		}

		serverRows = append(serverRows, db.ProxyServerRow{
			Host:      s.Host,
			IsUp:      s.IsUp,
			CheckPort: s.Port,
			LastCheck: s.LastCheck,
		})
	}

	err := r.storage.SaveProxyServers(serverRows)
	if err != nil {
		log.Println("Error saving servers", err)
	}

	return nil
}

// selectHealthyServer selects a server from storage where IsUp = true
func (r *Switcher) selectHealthyServer() (*db.ProxyServerRow, error) {
	servers, err := r.storage.GetProxyServers(true)
	if err != nil {
		return nil, err
	}
	for _, s := range servers {
		if s.IsUp {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("no healthy server found")
}

// changeDomainsTo is a placeholder for the logic to change domains to the new server
func (r *Switcher) changeDomainsTo(server *db.ProxyServerRow) {
	log.Printf("switcher: Changing domains to new server: %+v", server)
	
	domains, err := r.storage.GetDomainWithCfTokens()
	if err != nil {
		log.Printf("switcher: Failed to get domains with CF tokens: %v", err)
		return
	}

	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	
	for _, domain := range domains {
		wg.Add(1)
		sem <- struct{}{} // acquire

		go func(d db.DomainRow) {
			defer wg.Done()
			defer func() { <-sem }() // release

			err := r.updateDomainToServer(d, server)
			if err != nil {
				log.Printf("switcher: Failed to update domain %s: %v", d.Domain, err)
				r.Notify(fmt.Sprintf("Failed to update domain %s: %v", d.Domain, err))
			} else {
				log.Printf("switcher: Successfully updated domain %s to point to %s", d.Domain, server.Host)
			}
		}(domain)
	}

	wg.Wait()
	log.Println("switcher: All domain updates attempted")
}


func (r *Switcher) updateDomainToServer(domainWithCfToken db.DomainRow, server *db.ProxyServerRow) error {
	client := cf.NewClient(domainWithCfToken.CfApiToken)

	err := client.UpdateDomainIP(domainWithCfToken.Domain, server.Host)
	if err != nil {
		return err
	}

    return nil
}

func (r *Switcher) Notify(message string) {
	err := r.notifier.Notify(message)
	if err != nil {
		log.Printf("switcher: Failed to send notification: %v", err)
	}
}