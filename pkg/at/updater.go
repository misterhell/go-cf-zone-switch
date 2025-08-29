package at

import (
	"context"
	"log"
	"time"

	"go-cf-zone-switch/pkg/db"
)

type DbDomainsUpdater struct {
	Db       db.Storage
	Repo     *RemoteRepository
	Interval time.Duration
	Notifier Notifier
}

func NewDbDomainsSync(db db.Storage, at *RemoteRepository, interval time.Duration, notifier Notifier) *DbDomainsUpdater {
	return &DbDomainsUpdater{
		Db:       db,
		Repo:     at,
		Interval: interval,
		Notifier: notifier,
	}
}

func (d *DbDomainsUpdater) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(d.Interval)
		defer ticker.Stop()
		err := d.Sync()
		if err != nil {
			log.Println("updater: Domains update error", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := d.Sync(); err != nil {
					log.Println("updater: Domains update error", err)
					_ = d.Notifier.Notify("Error updating domains: " + err.Error())
				}
				log.Println("updater: Domains updated")

			case <-ctx.Done():
				log.Println("updater: Data updater stopped")
				return
			}
		}
	}()
}

func (d *DbDomainsUpdater) Sync() error {
	log.Println("updater: loading all domains")
	domains, err := d.Repo.GetAllDomains()
	if err != nil {
		return err
	}

	dbRows := []db.DomainRow{}
	for _, d := range domains {
		if d.Domain == "" {
			continue
		}
		dbRows = append(dbRows, db.DomainRow{
			Domain:     d.Domain,
			HostingIP:  d.HostingIP,
			CfApiToken: d.CfApiToken,
		})
	}
	log.Println("updater: domains loaded, saving")
	err = d.Db.SaveDomains(dbRows)

	if err == nil {
		log.Println("updater: domains saved")
	} else {
		log.Println("updater: domains save error")
	}

	return err
}
