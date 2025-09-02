package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go-cf-zone-switch/pkg/at"
	"go-cf-zone-switch/pkg/config"
	"go-cf-zone-switch/pkg/db"
	"go-cf-zone-switch/pkg/notifications"
	"go-cf-zone-switch/pkg/servers"
	"go-cf-zone-switch/pkg/switcher"
)

type Notifier interface {
	Notify(message string) error
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

	notifier := getNotifier(cfg)

	switcher := switcher.NewSwitcher(cfg, storage, notifier)

	startMonitoring(ctx, cfg, switcher, notifier)

	startDomainDataSync(ctx, storage, repo, cfg, notifier)

	startProxyConfigurator(ctx, storage, cfg, notifier)

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

func startMonitoring(ctx context.Context, cfg *config.Config, reporter servers.StatusReceiver, notifier Notifier) {
	checkInterval := time.Second * time.Duration(cfg.Servers.CheckIntervalSec)
	timeout := time.Second * time.Duration(cfg.Servers.TimeoutSec)

	monitoring := servers.NewServerMonitoring(checkInterval, timeout, reporter, notifier)

	for _, proxy := range cfg.Servers.Proxy {
		schema := "http"
		if strings.HasPrefix(proxy, "https://") {
			schema = "https"
		}

		proxy = strings.TrimPrefix(strings.TrimPrefix(proxy, "http://"), "https://")

		h, p, err := net.SplitHostPort(proxy)
		if err != nil {
			panic(err)
		}
		monitoring.AddServer(h, p, proxy, schema)
	}

	monitoring.Start(ctx)
}

func getNotifier(cfg *config.Config) notifications.Notifier {
	notifier := notifications.NewStackNotifier()
	notifier.AddNotifier(notifications.NewTelegramNotifier(cfg))

	return notifier
}

func startDomainDataSync(ctx context.Context, storage *db.DbStorage, repo *at.RemoteRepository, config *config.Config, notifier Notifier) {
	updateInterval := time.Duration(config.At.DomainsUpdateMin) * time.Minute

	updater := at.NewDbDomainsSync(storage, repo, updateInterval, notifier)

	updater.Start(ctx)
}

func startProxyConfigurator(ctx context.Context, storage *db.DbStorage, config *config.Config, notifier Notifier) {
	configUpdater := servers.NewProxyConfigUpdater(storage, &config.Servers, notifier)

	configUpdater.Start(ctx)
}
