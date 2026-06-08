package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

type Store struct {
	db     *bbolt.DB
	dbPath string
}

func NewStore(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}

	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open bolt db: %w", err)
	}

	store := &Store{db: db, dbPath: dbPath}

	if err := store.initBuckets(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *Store) initBuckets() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		buckets := []string{"project", "agents", "tasks", "decisions", "sessions", "config"}
		for _, name := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(name))
			if err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", name, err)
			}
		}
		return nil
	})
}

func (s *Store) Put(bucket, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.Put([]byte(key), data)
	})
}

func (s *Store) Get(bucket, key string, dest interface{}) error {
	return s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		data := b.Get([]byte(key))
		if data == nil {
			return fmt.Errorf("key %s not found in bucket %s", key, bucket)
		}
		return json.Unmarshal(data, dest)
	})
}

func (s *Store) Delete(bucket, key string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.Delete([]byte(key))
	})
}

func (s *Store) List(bucket string) ([]string, error) {
	var keys []string
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.ForEach(func(k, v []byte) error {
			keys = append(keys, string(k))
			return nil
		})
	})
	return keys, err
}

func (s *Store) Close() error {
	return s.db.Close()
}

type FileStore struct {
	basePath string
}

func NewFileStore(basePath string) *FileStore {
	return &FileStore{basePath: basePath}
}

func (fs *FileStore) Write(filename string, content []byte) error {
	fullPath := filepath.Join(fs.basePath, filename)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, content, 0644)
}

func (fs *FileStore) Read(filename string) ([]byte, error) {
	return os.ReadFile(filepath.Join(fs.basePath, filename))
}
