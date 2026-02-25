package zdb

import (
	"context"
	"time"
)

// 当前是本地缓存，可以扩展为 redis 作为全局缓存使用
// 未什么存储的 value 是 string ? 统一存储格式，便于底层处理跟换存储介质

type Cache interface {
	Get(ctx context.Context, key string) (string, bool, error)                  // 获取数据
	Set(ctx context.Context, key, value string, expiration time.Duration) error // 存储数据，并指定到期时间
	Set1(ctx context.Context, key string, expiration time.Duration) error       // 存放一个键值, 只用来确定键值存在
	Keys(ctx context.Context, pattern string) ([]string, error)                 // 获取所有具有相同模式的键值

	Expire(ctx context.Context, key string, expiration time.Duration) error // 延期数据
	Exists(ctx context.Context, key string) (bool, error)                   // 检查数据
	Delete(ctx context.Context, key string) error                           // 删除数据
	DelAll(ctx context.Context, keys ...string) error                       // 删除数据
}

// CacheX 具有类型的缓存， 一般用于本地存储对象

type CacheX interface {
	GetX(ctx context.Context, key string) (any, bool, error)
	SetX(ctx context.Context, key string, value any, exp time.Duration) error
	Set1(ctx context.Context, key string, expiration time.Duration) error
	Keys(ctx context.Context, pattern string) ([]string, error)

	Expire(ctx context.Context, key string, expiration time.Duration) error // 延期数据
	Exists(ctx context.Context, key string) (bool, error)                   // 检查数据
	Delete(ctx context.Context, key string) error                           // 删除数据
	DelAll(ctx context.Context, keys ...string) error                       // 删除数据
}
