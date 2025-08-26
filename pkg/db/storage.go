package db

import (
	"encoding/json"

	"github.com/boltdb/bolt"
)

const storage = "changer.boltdb"

var (
	serverBucket  = []byte("servers")
	domainsBucket = []byte("domains")
)

type Storage struct {
	db *bolt.DB
}

func (s *Storage) Close() {
	defer s.db.Close()
}

func NewStorage() (*Storage, error) {
	db, err := bolt.Open(storage, 0o600, nil)
	if err != nil {
		return nil, err
	}

	storage := &Storage{
		db: db,
	}

	if storage.initDbBuckets() != nil {
		return nil, err
	}

	return storage, nil
}

func (s *Storage) initDbBuckets() error {
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

func (s *Storage) SaveDomains(domains []DomainRow) error {
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

func (s *Storage) GetAllDomains() ([]DomainRow, error) {
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
