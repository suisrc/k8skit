package cache

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

// NewCacheMem 使用 go-cache 创建存储实例, 暂时只支持单实例

func NewCacheMem(defaultExpiration time.Duration, cleanupInterval time.Duration) *MemCacheX {
	return &MemCacheX{
		cac: cache.New(defaultExpiration, cleanupInterval),
	}
}

func NewCacheMemDef() *MemCacheX {
	return NewCacheMem(5*time.Minute, 10*time.Minute)
}

var _ Cache = new(MemCacheX)
var _ CacheX = new(MemCacheX)

// memory 存储
type MemCacheX struct {
	cac *cache.Cache
}

// Get ...
func (s *MemCacheX) Get(ctx context.Context, key string) (string, bool, error) {
	if val, ok := s.cac.Get(key); !ok {
		return "", ok, nil
	} else if vv, kk := val.(string); kk {
		return vv, ok, nil
	} else {
		return "", ok, errors.New("not string")
		// return fmt.Sprintf("%v", val), ok, nil
	}
}

// Set ...
func (s *MemCacheX) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	s.cac.Set(key, value, expiration)
	return nil
}

// GetX ...
func (s *MemCacheX) GetX(ctx context.Context, key string) (any, bool, error) {
	val, ok := s.cac.Get(key)
	return val, ok, nil
}

// SetX ...
func (s *MemCacheX) SetX(ctx context.Context, key string, value any, expiration time.Duration) error {
	s.cac.Set(key, value, expiration)
	return nil
}

// Set1 ...
func (s *MemCacheX) Set1(ctx context.Context, key string, expiration time.Duration) error {
	s.cac.Set(key, true, expiration)
	return nil
}

// Keys ...
func (s *MemCacheX) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys := []string{}
	for key := range s.cac.Items() {
		if pattern == "" || strings.HasPrefix(key, pattern) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// Expire ...
func (s *MemCacheX) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if val, ok := s.cac.Get(key); ok {
		s.cac.Set(key, val, expiration)
		return nil
	}
	return errors.New("not exist")
}

// Exists ...
func (s *MemCacheX) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := s.cac.Get(key)
	return ok, nil
}

// Delete ...
func (s *MemCacheX) Delete(ctx context.Context, key string) error {
	s.cac.Delete(key)
	return nil
}

// DelAll ...
func (s *MemCacheX) DelAll(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		s.cac.Delete(key)
	}
	return nil
}
