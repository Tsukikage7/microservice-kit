// Package cache 提供统一的缓存接口和实现.
package cache

import (
	"context"
	"time"
)

// 缓存类型常量.
const (
	TypeRedis  = "redis"
	TypeMemory = "memory"
)

// 默认配置值.
const (
	DefaultPoolSize     = 10
	DefaultTimeout      = 5 * time.Second
	DefaultReadTimeout  = 3 * time.Second
	DefaultWriteTimeout = 3 * time.Second
	DefaultMaxRetries   = 3
)

// Cache 缓存接口.
type Cache interface {
	// 基础操作
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)

	// 原子操作
	SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)
	Increment(ctx context.Context, key string) (int64, error)
	IncrementBy(ctx context.Context, key string, value int64) (int64, error)
	Decrement(ctx context.Context, key string) (int64, error)

	// 过期时间
	Expire(ctx context.Context, key string, ttl time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)

	// 分布式锁
	TryLock(ctx context.Context, key string, value string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, key string, value string) error

	// 批量操作
	MGet(ctx context.Context, keys ...string) ([]string, error)
	MSet(ctx context.Context, pairs map[string]any, ttl time.Duration) error

	// 资源管理
	Ping(ctx context.Context) error
	Close() error
	Client() any
}

