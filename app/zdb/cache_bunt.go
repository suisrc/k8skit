package zdb

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/tidwall/buntdb"
)

// NewCacheBunt 使用 buntdb 创建存储实例, 好处是便于测试，可以在 文件 和 内存 中切换
func NewCacheBunt(path string) (*BuntCache, error) {
	if path == "" {
		path = ":memory:"
	}
	if path != ":memory:" {
		os.MkdirAll(filepath.Dir(path), 0777)
	}
	bdb, err := buntdb.Open(path)
	if err != nil {
		return nil, err
	}
	return &BuntCache{
		bdb: bdb,
	}, nil
}

var _ Cache = new(BuntCache)

// BuntCache buntdb存储
type BuntCache struct {
	bdb *buntdb.DB
}

// Set ...
func (a *BuntCache) Set(ctx context.Context, key, value string, expiration time.Duration) error {
	return a.bdb.Update(func(tx *buntdb.Tx) error {
		var opts *buntdb.SetOptions
		if expiration > 0 {
			opts = &buntdb.SetOptions{Expires: true, TTL: expiration}
		}
		_, _, err := tx.Set(key, value, opts)
		return err
	})
}

// Get ...
func (a *BuntCache) Get(ctx context.Context, key string) (string, bool, error) {
	var exists bool
	var value string
	err := a.bdb.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(key)
		if err != nil {
			if err == buntdb.ErrNotFound {
				return nil
			}
			return err
		}
		value = val
		exists = true
		return nil
	})
	return value, exists, err
}

// Set1 ...
func (a *BuntCache) Set1(ctx context.Context, key string, expiration time.Duration) error {
	return a.bdb.Update(func(tx *buntdb.Tx) error {
		var opts *buntdb.SetOptions
		if expiration > 0 {
			opts = &buntdb.SetOptions{Expires: true, TTL: expiration}
		}
		_, _, err := tx.Set(key, "1", opts)
		return err
	})
}

// Keys ...
func (a *BuntCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys := []string{}
	err := a.bdb.View(func(tx *buntdb.Tx) error {
		tx.AscendKeys(pattern, func(key, value string) bool {
			keys = append(keys, key)
			return true
		})
		return nil
	})
	return keys, err
}

// Expire ...
func (a *BuntCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	err := a.bdb.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(key)
		if err != nil && err != buntdb.ErrNotFound {
			return err
		}
		var opts *buntdb.SetOptions
		if expiration > 0 {
			opts = &buntdb.SetOptions{Expires: true, TTL: expiration}
		}
		_, _, err = tx.Set(key, val, opts)
		return err
	})
	return err
}

// Exists ...
func (a *BuntCache) Exists(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := a.bdb.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(key)
		if err != nil && err != buntdb.ErrNotFound {
			return err
		}
		exists = val != ""
		return nil
	})
	return exists, err
}

// Delete ...
func (a *BuntCache) Delete(ctx context.Context, key string) error {
	return a.bdb.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(key)
		if err != nil && err != buntdb.ErrNotFound {
			return err
		}
		return nil
	})
}

// DelAll ...
func (a *BuntCache) DelAll(ctx context.Context, keys ...string) error {
	return a.bdb.Update(func(tx *buntdb.Tx) error {
		for _, key := range keys {
			tx.Delete(key)
		}
		return nil
	})
}
