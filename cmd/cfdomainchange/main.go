package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go-cf-zone-switch/pkg/at"
	"go-cf-zone-switch/pkg/cf"
	"go-cf-zone-switch/pkg/config"
	"go-cf-zone-switch/pkg/servers"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config-path", "", "Path to the config file")
	newIP := flag.String("new-ip", "", "New IP address to set for all domains")
	checkReachability := flag.Bool("check-reachability", true, "Check if the new IP is reachable before updating")
	useLocal := flag.Bool("use-local", false, "Use local repository instead of Airtable")
	flag.Parse()

	// Validate required flags
	if *newIP == "" {
		log.Fatal("New IP address is required. Use --new-ip flag.")
	}

	var repository at.Repository
	if *useLocal {
		repository = at.NewLocalRepository()
	} else {
		if *configPath == "" {
			log.Fatal("Config path is required when using Airtable. Use --config-path flag.")
		}

		// Load config
		cfg, err := config.Load(*configPath)
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		repository = at.NewRemoteRepository(cfg.At)
	}

	// Check if the new IP is reachable
	if *checkReachability {
		fmt.Println("Checking if the new IP is reachable...")
		reachable, err := servers.IsServerReachable(*newIP, "80", 2*time.Second)
		if !reachable {
			fmt.Printf("⚠️  Warning: New IP is not reachable on port 80. Error: %v\n", err)
			fmt.Print("Do you want to continue anyway? (y/n): ")
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Aborted.")
				os.Exit(0)
			}
		} else {
			fmt.Println("✅ New IP is reachable.")
		}
	}

	// Get all domains
	fmt.Println("Fetching domains...")
	domains, err := repository.GetAllDomains()
	if err != nil {
		log.Fatalf("Error fetching domains: %v", err)
	}

	fmt.Printf("Found %d domains.\n", len(domains))

	// Prepare domain to token map
	domainTokens := make(map[string]string)
	for _, domain := range domains {
		domainTokens[domain.Domain] = domain.Token
	}

	// Update domain IPs
	fmt.Println("Updating domain A records...")
	results := cf.UpdateDomainsIP(domainTokens, *newIP)

	// Print results
	fmt.Println("\nUpdate Results:")
	fmt.Println(cf.PrintUpdateResults(results))

	// Count successes and failures
	successes := 0
	failures := 0
	for _, err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}

	fmt.Printf("\nSummary: %d successful, %d failed\n", successes, failures)
}
