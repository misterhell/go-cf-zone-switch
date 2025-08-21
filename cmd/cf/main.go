package main

import (
	"context"
	"flag"
	"log"
	"os"

	// "go-cf-zone-switch/pkg/at"
	"go-cf-zone-switch/pkg/at"
	"go-cf-zone-switch/pkg/cf"
	"go-cf-zone-switch/pkg/config"
)

func main() {
	cfgPath := flag.String("config-path", "config.toml", "Set path of toml file with config ")

	flag.Parse()
	cfg, err := config.Load(*cfgPath)

	log.Printf("Config loaded %s ", *cfgPath)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = cfg
	_ = ctx

	atr := at.NewLocalRepository()
	domains, _ := atr.GetAllDomains()

	for _, domain := range domains {
		c := cf.NewClient(domain.CfApiToken)
		ip, err := c.GetDomainIP(domain.Domain)
		if err != nil {
			log.Panicln(err)
		}
		log.Printf("Domain: %s %s\n", domain.Domain, ip)

		err = c.UpdateDomainIP(domain.Domain, "176.9.70.12")
		if err != nil {
			log.Panicln(err)
		}
	}
}
