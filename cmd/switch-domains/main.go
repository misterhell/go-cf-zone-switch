package main

import (
	"log"
	"os"

	"go-cf-zone-switch/pkg/at"
	"go-cf-zone-switch/pkg/config"
	"go-cf-zone-switch/pkg/db"
	"go-cf-zone-switch/pkg/notifications"
	"go-cf-zone-switch/pkg/switcher"
)

func main() {
	// Load config
	cfg, err := config.Load("config.toml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Open storage (DB)
	storage, err := db.NewStorage()

	updater := at.NewDbDomainsSync(storage, at.NewRemoteRepository(cfg.At), 0, nil)
	err = updater.Sync()
	if err != nil {
		log.Fatalf("failed to open storage: %v", err)
	}
	defer storage.Close()

	// Find a healthy proxy server
	servers, err := storage.GetProxyServers(true)
	if err != nil {
		log.Fatalf("failed to get proxy servers: %v", err)
	}
	if len(servers) == 0 {
		log.Println("no healthy proxy servers found")
		os.Exit(1)
	}
	healthy := &servers[0]

	domains, err := storage.GetDomainWithCfTokens()
	if err != nil {
		log.Fatalf("failed to get domains: %v", err)
	}
	if len(domains) == 0 {
		log.Println("no domains found")
		os.Exit(0)
	}

	// Switch all domains to the healthy proxy server
	sw := switcher.NewSwitcher(cfg, storage, notifications.NewStackNotifier())
	sw.ChangeAllDomainsToServer(domains, healthy)
}
