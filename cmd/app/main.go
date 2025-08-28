package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	// "go-cf-zone-switch/pkg/at"
	"go-cf-zone-switch/pkg/at"
	"go-cf-zone-switch/pkg/config"
	"go-cf-zone-switch/pkg/db"
	"go-cf-zone-switch/pkg/notifications"
	"go-cf-zone-switch/pkg/servers"
)

type Reporter struct {
	storage *db.Storage
}

func NewReporter(storage *db.Storage) *Reporter {
	return &Reporter{
		storage: storage,
	}
}

func (r *Reporter) ReportStatus(statuses []servers.ServerStatus) error {
	serverRows := []db.ProxyServerRow{}
	for _, s := range statuses {
		log.Printf("reporter: Report received %+v\n", s)
		portInt, err := strconv.Atoi(s.Port)
		if err != nil {
			log.Printf("Invalid port %s for host %s, using 0", s.Port, s.Host)
			portInt = 0
		}

		serverRows = append(serverRows, db.ProxyServerRow{
			Host:      s.Host,
			IsUp:      s.IsUp,
			CheckPort: portInt,
			LastCheck: s.LastCheck,
		})
	}

	err := r.storage.SaveProxyServers(serverRows)
	if err != nil {
		log.Println("Error saving servers", err)
	}

	return nil
}

type Notifier interface {
	Notify() error
}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func main() {
	cfgPath := flag.String("config-path", "config.toml", "Set path of toml file with config ")

	// config loading
	flag.Parse()
	cfg, err := config.Load(*cfgPath)
	checkErr(err)

	log.Printf("Config loaded %s ", *cfgPath)

	// create database
	storage, err := db.NewStorage()
	checkErr(err)
	defer storage.Close()

	// Create AT repository
	repo := at.NewRemoteRepository(cfg.At)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// notifier := notifications.TelegramNotifier{}

	// TODO: replace with a real service
	reporter := NewReporter(storage)

	err = startMonitoring(ctx, cfg, reporter)
	checkErr(err)

	startDomainDataSync(ctx, storage, repo, cfg)

	startProxyConfigurator(ctx, storage, cfg)

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigs:
			log.Println(sig)
			cancel() // Cancel context on signal
		case <-ctx.Done():
			log.Println("app: context canceled")
		}
		done <- true
	}()

	log.Println("app: awaiting signal or context cancellation")
	<-done
	time.Sleep(time.Second * 1) // awaiting for routines to finish
	log.Println("app: exiting")
}

func startMonitoring(ctx context.Context, cfg *config.Config, reporter *Reporter) error {
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

func startDomainDataSync(ctx context.Context, storage *db.Storage, repo *at.RemoteRepository, config *config.Config) {
	updateInterval := time.Duration(config.At.DomainsUpdateMin) * time.Minute

	updater := at.NewDbDomainsSync(storage, repo, updateInterval)

	updater.Start(ctx)
}

func startProxyConfigurator(ctx context.Context, storage *db.Storage, config *config.Config) {
	configUpdater := servers.NewProxyConfigUpdater(storage, &config.Servers)

	configUpdater.Start(ctx)
}
