package analytics

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
)

var buckets = []string{
	"events",
	"daily",
	"referrers",
	"countries",
	"types",
}

type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Referrer  string    `json:"referrer"`
	Country   string    `json:"country"`
	URL       string    `json:"url"`
	Timestamp time.Time `json:"timestamp"`
}

type Stats struct {
	EventCounts  map[string]int64 `json:"event_counts"`
	TopReferrers map[string]int64 `json:"top_referrers"`
	TopCountries map[string]int64 `json:"top_countries"`
}

type Store struct {
	db *bolt.DB
}

func NewStore(db *bolt.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Init() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		for _, b := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(b)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) Record(event Event) error {
	event.ID = uuid.New().String()
	event.Timestamp = time.Now().UTC()

	return s.db.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}

		key := fmt.Sprintf("%d-%s", event.Timestamp.UnixNano(), event.ID)
		tx.Bucket([]byte("events")).Put([]byte(key), data)

		dateKey := event.Timestamp.Format("2006-01-02")
		s.incrementCounter(tx, "daily", dateKey)

		if event.Referrer != "" {
			s.incrementCounter(tx, "referrers", event.Referrer)
		}

		if event.Country != "" {
			s.incrementCounter(tx, "countries", event.Country)
		}

		s.incrementCounter(tx, "types", event.Type)

		return nil
	})
}

func (s *Store) incrementCounter(tx *bolt.Tx, bucket, key string) {
	b := tx.Bucket([]byte(bucket))
	val := b.Get([]byte(key))
	var count int64
	if val != nil {
		json.Unmarshal(val, &count)
	}
	count++
	data, _ := json.Marshal(count)
	b.Put([]byte(key), data)
}

func (s *Store) GetStats() (*Stats, error) {
	stats := &Stats{
		TopReferrers: make(map[string]int64),
		TopCountries: make(map[string]int64),
		EventCounts:  make(map[string]int64),
	}

	return stats, s.db.View(func(tx *bolt.Tx) error {
		tx.Bucket([]byte("types")).ForEach(func(k, v []byte) error {
			var count int64
			json.Unmarshal(v, &count)
			stats.EventCounts[string(k)] = count
			return nil
		})

		tx.Bucket([]byte("referrers")).ForEach(func(k, v []byte) error {
			var count int64
			json.Unmarshal(v, &count)
			stats.TopReferrers[string(k)] = count
			return nil
		})

		tx.Bucket([]byte("countries")).ForEach(func(k, v []byte) error {
			var count int64
			json.Unmarshal(v, &count)
			stats.TopCountries[string(k)] = count
			return nil
		})

		return nil
	})
}
