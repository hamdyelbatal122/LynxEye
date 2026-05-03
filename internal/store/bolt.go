package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	bolt "go.etcd.io/bbolt"

	"github.com/hamdyelbatal122/lynxeye/internal/model"
)

var clusterBucket = []byte("clusters")

type Store struct {
	db *bolt.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}

	db, err := bolt.Open(path, 0o600, nil)
	if err != nil {
		return nil, fmt.Errorf("open bbolt database: %w", err)
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(clusterBucket)
		return err
	}); err != nil {
		db.Close()
		return nil, fmt.Errorf("init buckets: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) LoadClusters() ([]*model.Cluster, error) {
	clusters := make([]*model.Cluster, 0)
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clusterBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(_, value []byte) error {
			var cluster model.Cluster
			if err := json.Unmarshal(value, &cluster); err != nil {
				return err
			}
			copied := cluster
			clusters = append(clusters, &copied)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("load clusters: %w", err)
	}

	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].ID < clusters[j].ID
	})

	return clusters, nil
}

func (s *Store) SaveCluster(cluster *model.Cluster) error {
	encoded, err := json.Marshal(cluster)
	if err != nil {
		return fmt.Errorf("marshal cluster: %w", err)
	}

	if err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clusterBucket)
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", clusterBucket)
		}
		return bucket.Put([]byte(cluster.Pattern), encoded)
	}); err != nil {
		return fmt.Errorf("save cluster: %w", err)
	}

	return nil
}
