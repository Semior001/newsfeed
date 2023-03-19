package store

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	bolt "go.etcd.io/bbolt"
)

const usersBktName = "users"

// Bolt is a storage that uses BoltDB as a backend.
type Bolt struct {
	db *bolt.DB
}

// NewBolt creates new Bolt storage.
func NewBolt(dir string) (*Bolt, error) {
	db, err := bolt.Open(path.Join(dir, "users.db"), 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make boltdb for %s: %w", dir, err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, name := range []string{usersBktName} {
			if _, err := tx.CreateBucketIfNotExists([]byte(name)); err != nil {
				return fmt.Errorf("create top-level bucket %s: %w", name, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("make buckets: %w", err)
	}

	return &Bolt{db: db}, nil
}

// Put puts user to storage.
func (b *Bolt) Put(_ context.Context, u User) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(usersBktName))

		bts, err := json.Marshal(u)
		if err != nil {
			return fmt.Errorf("marshal user: %w", err)
		}

		if err := bkt.Put([]byte(u.ChatID), bts); err != nil {
			return fmt.Errorf("put sub to storage: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("update storage: %w", err)
	}

	return nil
}

// List returns all users from storage.
func (b *Bolt) List(context.Context, ListRequest) ([]User, error) {
	var result []User
	err := b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(usersBktName))
		err := bkt.ForEach(func(k, v []byte) error {
			var u User
			if err := json.Unmarshal(v, &u); err != nil {
				return fmt.Errorf("unmarshal user %s: %w", k, err)
			}
			result = append(result, u)
			return nil
		})
		if err != nil {
			return fmt.Errorf("foreach: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("view storage: %w", err)
	}
	return result, nil
}

// Get returns user from storage.
func (b *Bolt) Get(_ context.Context, id string) (u User, err error) {
	err = b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(usersBktName))

		bts := bkt.Get([]byte(id))
		if bts == nil {
			return ErrNotFound
		}

		if err := json.Unmarshal(bts, &u); err != nil {
			return fmt.Errorf("unmarshal user: %w", err)
		}

		return nil
	})
	if err != nil {
		return User{}, fmt.Errorf("view storage: %w", err)
	}

	return u, nil
}

// Delete removes user from storage.
func (b *Bolt) Delete(_ context.Context, id string) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(usersBktName))

		if err := bkt.Delete([]byte(id)); err != nil {
			return fmt.Errorf("remove: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("update storage: %w", err)
	}

	return nil
}

// Close closes the storage.
func (b *Bolt) Close() error { return b.db.Close() }
