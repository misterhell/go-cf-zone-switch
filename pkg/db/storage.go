package db

import (
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"
)

const storage = "changer.boltdb"

var (
	serverBucket  = []byte("servers")
	domainsBucket = []byte("domains")
)

type Storage interface {
	SaveProxyServers([]ProxyServerRow) error
	GetProxyServers(onlyHealthy bool) ([]ProxyServerRow, error)
	GetDomainWithCfTokens() ([]DomainRow, error)
	SaveDomains([]DomainRow) error
	GetAllDomains() ([]DomainRow, error)
	Close()
}

type DbStorage struct {
	db *bolt.DB

	Storage
}

func (s *DbStorage) Close() {
	defer s.db.Close()
}

func NewStorage() (*DbStorage, error) {
	db, err := bolt.Open(storage, 0o600, nil)
	if err != nil {
		return nil, err
	}

	storage := &DbStorage{
		db: db,
	}

	if storage.initDbBuckets() != nil {
		return nil, err
	}

	return storage, nil
}

func (s *DbStorage) initDbBuckets() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(serverBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(domainsBucket); err != nil {
			return err
		}

		return nil
	})
}

type DomainRow struct {
	Domain     string `json:"domain"`
	HostingIP  string `json:"hosting_ip"`
	CfApiToken string `json:"cf_api_token,omitempty"`
}

func (d DomainRow) Key() []byte {
	return []byte(d.Domain)
}

func (d DomainRow) Value() ([]byte, error) {
	return json.Marshal(d)
}

func (s *DbStorage) SaveDomains(domains []DomainRow) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(domainsBucket)

		for _, d := range domains {
			key := d.Key()
			val, err := d.Value()
			if err != nil {
				return err
			}
			if err := b.Put(key, val); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *DbStorage) GetAllDomains() ([]DomainRow, error) {
	var domains []DomainRow
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(domainsBucket)
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			var d DomainRow
			if err := json.Unmarshal(v, &d); err != nil {
				return err
			}
			domains = append(domains, d)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return domains, nil
}

func (s *DbStorage) GetDomainWithCfTokens() ([]DomainRow, error) {
	var domainsWithTokens []DomainRow
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(domainsBucket)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var d DomainRow
			if err := json.Unmarshal(v, &d); err != nil {
				return err
			}
			if d.CfApiToken != "" {
				domainsWithTokens = append(domainsWithTokens, d)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return domainsWithTokens, nil
}

type ProxyServerRow struct {
	IsUp      bool      `json:"is_up"`
	Host      string    `json:"host"`
	CheckPort string    `json:"check_port"`
	LastCheck time.Time `json:"last_check"`
}

func (d ProxyServerRow) Key() []byte {
	return []byte(d.Host)
}

func (d ProxyServerRow) Value() ([]byte, error) {
	return json.Marshal(d)
}

func (s *DbStorage) SaveProxyServers(servers []ProxyServerRow) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(serverBucket)
		for _, server := range servers {
			key := server.Key()
			val, err := server.Value()
			if err != nil {
				return err
			}
			if err := b.Put(key, val); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *DbStorage) GetProxyServers(isUp bool) ([]ProxyServerRow, error) {
	var servers []ProxyServerRow

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(serverBucket)
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			var s ProxyServerRow

			if err := json.Unmarshal(v, &s); err != nil {
				return err
			}
			servers = append(servers, s)

			return nil
		})
	})

	if isUp {
		filtered := make([]ProxyServerRow, 0, len(servers))
		for _, server := range servers {
			if server.IsUp {
				filtered = append(filtered, server)
			}
		}
		servers = filtered
	}

	return servers, err
}
