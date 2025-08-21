package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	// "go-cf-zone-switch/pkg/at"
	"go-cf-zone-switch/pkg/at"
	"go-cf-zone-switch/pkg/config"
	"go-cf-zone-switch/pkg/servers"
)

type Reporter struct {
	domains map[string]string
}

func NewReporter() *Reporter {
	return &Reporter{
		domains: map[string]string{},
	}
}

func (r *Reporter) AddDomain(domain, cfToken string) {
	r.domains[domain] = cfToken
}

func (r *Reporter) ReportStatus(statuses []servers.ServerStatus) error {
	for _, s := range statuses {
		log.Printf("%+v\n", s)
	}
	return nil
}

type Notifier interface {
	Notify() error
}

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

	reporter, err := initReporter(ctx, cfg)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	err = initMonitoring(ctx, cfg, reporter)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigs:
			log.Println(sig)
			cancel() // Cancel context on signal
		case <-ctx.Done():
			log.Println("context canceled")
		}
		done <- true
	}()

	log.Println("awaiting signal or context cancellation")
	<-done
	time.Sleep(time.Second * 1) // awaiting for routines to finish
	log.Println("exiting")
}

func initReporter(ctx context.Context, cfg *config.Config) (*Reporter, error) {
	reporter := NewReporter()

	rep := at.NewLocalRepository()

	domains, err := rep.GetAllDomains()
	if err != nil {
		return nil, err
	}

	for _, d := range domains {
		reporter.AddDomain(d.Domain, d.CfApiToken)
	}

	return reporter, nil
}

func initMonitoring(ctx context.Context, cfg *config.Config, reporter *Reporter) error {
	checkInterval := time.Second * time.Duration(cfg.Servers.CheckIntervalSec)
	timeout := time.Second * time.Duration(cfg.Servers.TimeoutSec)

	monitoring := servers.NewServerMonitoring(checkInterval, timeout, reporter)

	for _, proxy := range cfg.Servers.Proxy {
		h, p, err := net.SplitHostPort(proxy)
		if err != nil {
			return err
		}
		monitoring.AddServer(h, p, proxy)
	}

	monitoring.Start(ctx)

	return nil
}
