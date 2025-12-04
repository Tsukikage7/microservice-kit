// Package cache 提供统一的缓存接口和实现.
package cache

import (
	"context"
	"errors"
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

// 常见错误.
var (
	ErrNotFound    = errors.New("缓存键不存在")
	ErrLockNotHeld = errors.New("锁未持有或已过期")
	ErrNilConfig   = errors.New("缓存配置为空")
	ErrEmptyAddr   = errors.New("缓存地址为空")
	ErrUnsupported = errors.New("不支持的缓存类型")
	ErrNilLogger   = errors.New("日志记录器为空")
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

