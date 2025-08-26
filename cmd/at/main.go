package main

import (
	"flag"
	"log"
	"os"

	"go-cf-zone-switch/pkg/at"
	"go-cf-zone-switch/pkg/config"
	"go-cf-zone-switch/pkg/db"
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


	storage, err := db.NewStorage()
	checkErr(err)

	defer storage.Close()
	// storage.SaveDomain()

	// Create AT repository
	repo := at.NewRemoteRepository(cfg.At)

	// Get all domains
	domains, err := repo.GetAllDomains()

	dbRows := []db.DomainRow{}
	for _, d := range domains {
		dbRows = append(dbRows, db.DomainRow{
			Domain: d.Domain,
			HostingIP: d.HostingIP,
			CfApiToken: d.CfApiToken,
		})
	}
	
	err = storage.SaveDomains(dbRows)
	checkErr(err)

	domainsRows, err := storage.GetAllDomains()
	checkErr(err)

	// Print results
	log.Printf("Found %d domains:\n", len(domains))
	for i, domain := range domainsRows {
		if i > 100 {
			break
		}
		log.Printf("Domain: %s, Token: %s, Hosting IP: %s\n", domain.Domain, domain.CfApiToken, domain.HostingIP)
	}
}


func checkErr(e error) {
	if e != nil {
		log.Println(e)
		os.Exit(1)
	}
}