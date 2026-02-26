package cache

import (
	"context"
	"time"

	"github.com/go-redis/redis"
)

// NewRedisCache 使用redis客户端创建存储实例
func NewCacheRedis(cli *redis.Client, keyPrefix string) *RedisCache {
	return &RedisCache{
		cli:    cli,
		prefix: keyPrefix,
	}
}

// NewRedisCacheX 使用redis集群客户端创建存储实例
func NewCacheRedix(cli *redis.ClusterClient, keyPrefix string) *RedisCache {
	return &RedisCache{
		cli:    cli,
		prefix: keyPrefix,
	}
}

// redis.Cmdable
// type redisClienter interface {
// 	Get(key string) *redis.StringCmd
// 	Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd
// 	Keys(pattern string) *redis.StringSliceCmd
// 	Expire(key string, expiration time.Duration) *redis.BoolCmd
// 	Exists(keys ...string) *redis.IntCmd
// 	TxPipeline() redis.Pipeliner
// 	Del(keys ...string) *redis.IntCmd
// 	Close() error
// }

var _ Cache = new(RedisCache)

// redis存储
type RedisCache struct {
	cli    redis.Cmdable
	prefix string
}

func (s *RedisCache) wrapperKey(key string) string {
	// return fmt.Sprintf("%s%s", s.prefix, key)
	return s.prefix + key
}

// Get ...
func (s *RedisCache) Get(ctx context.Context, key string) (string, bool, error) {
	cmd := s.cli.Get(s.wrapperKey(key))
	if err := cmd.Err(); err == nil {
		return cmd.Val(), true, nil
	} else if err.Error() == "redis: nil" {
		return "", false, nil
	} else {
		return "", false, err
	}
}

// Set ...
func (s *RedisCache) Set(ctx context.Context, key, value string, expiration time.Duration) error {
	cmd := s.cli.Set(s.wrapperKey(key), value, expiration)
	return cmd.Err()
}

// Set1 ...
func (s *RedisCache) Set1(ctx context.Context, key string, expiration time.Duration) error {
	cmd := s.cli.Set(s.wrapperKey(key), "1", expiration)
	return cmd.Err()
}

// Keys ...
func (s *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	cmd := s.cli.Keys(s.wrapperKey(pattern))
	return cmd.Val(), cmd.Err()
}

// Expire ...
func (s *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	cmd := s.cli.Expire(s.wrapperKey(key), expiration)
	return cmd.Err()
}

// Exists ...
func (s *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	cmd := s.cli.Exists(s.wrapperKey(key))
	if err := cmd.Err(); err == nil {
		return cmd.Val() > 0, nil
	} else if err.Error() == "redis: nil" {
		return false, nil
	} else {
		return false, err
	}
}

// Delete ...
func (s *RedisCache) Delete(ctx context.Context, key string) error {
	cmd := s.cli.Del(s.wrapperKey(key))
	return cmd.Err()
}

// DelAll ...
func (s *RedisCache) DelAll(ctx context.Context, keys ...string) error {
	wkeys := []string{}
	for _, key := range keys {
		wkeys = append(wkeys, s.wrapperKey(key))
	}
	cmd := s.cli.Del(wkeys...)
	return cmd.Err()
}
