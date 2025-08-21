package main

import (
	"flag"
	"log"
	"os"

	"go-cf-zone-switch/pkg/at"
	"go-cf-zone-switch/pkg/config"
)

func main() {
	cfgPath := flag.String("config-path", "config.toml", "Set path of toml file with config")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*cfgPath)
	log.Printf("Config loaded %s", *cfgPath)
	if err != nil {
		log.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Create AT repository
	repo := at.NewRemoteRepository(cfg.At)

	// Get all domains
	domains, err := repo.GetAllDomains()
	if err != nil {
		log.Printf("Failed to get domains: %v\n", err)
		os.Exit(1)
	}

	// Print results
	log.Printf("Found %d domains:\n", len(domains))
	for _, domain := range domains {
		log.Printf("Domain: %s, Token: %s, Hosting IP: %s\n", domain.Domain, domain.CfApiToken, domain.HostingIP)
	}
}
