package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/Tsukikage7/microservice-kit/cache"
)

// DistributedLimiter 分布式限流器.
//
// 使用 Redis 等缓存实现分布式限流.
type DistributedLimiter struct {
	cache  cache.Cache
	prefix string
	limit  int
	window time.Duration
}

// DistributedConfig 分布式限流配置.
type DistributedConfig struct {
	// Cache 缓存实例
	Cache cache.Cache

	// Prefix 缓存键前缀
	Prefix string

	// Limit 窗口内允许的最大请求数
	Limit int

	// Window 窗口大小
	Window time.Duration
}

// NewDistributedLimiter 创建分布式限流器.
func NewDistributedLimiter(cfg *DistributedConfig) (*DistributedLimiter, error) {
	if cfg == nil {
		return nil, ErrInvalidConfig
	}
	if cfg.Cache == nil {
		return nil, ErrNilCache
	}
	if cfg.Limit <= 0 {
		return nil, fmt.Errorf("%w: limit 必须大于 0", ErrInvalidConfig)
	}
	if cfg.Window <= 0 {
		return nil, fmt.Errorf("%w: window 必须大于 0", ErrInvalidConfig)
	}

	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "ratelimit"
	}

	return &DistributedLimiter{
		cache:  cfg.Cache,
		prefix: prefix,
		limit:  cfg.Limit,
		window: cfg.Window,
	}, nil
}

// Allow 检查是否允许请求通过.
func (dl *DistributedLimiter) Allow(ctx context.Context) bool {
	return dl.AllowWithKey(ctx, "default")
}

// AllowN 检查是否允许 n 个请求通过.
func (dl *DistributedLimiter) AllowN(ctx context.Context, n int) bool {
	return dl.AllowNWithKey(ctx, "default", n)
}

// AllowWithKey 检查指定键是否允许请求通过.
func (dl *DistributedLimiter) AllowWithKey(ctx context.Context, key string) bool {
	return dl.AllowNWithKey(ctx, key, 1)
}

// AllowNWithKey 检查指定键是否允许 n 个请求通过.
func (dl *DistributedLimiter) AllowNWithKey(ctx context.Context, key string, n int) bool {
	cacheKey := fmt.Sprintf("%s:%s", dl.prefix, key)

	// 使用原子递增操作
	count, err := dl.cache.IncrementBy(ctx, cacheKey, int64(n))
	if err != nil {
		// 发生错误时默认放行，避免影响正常业务
		return true
	}

	// 首次设置过期时间
	if count == int64(n) {
		_ = dl.cache.Expire(ctx, cacheKey, dl.window)
	}

	return count <= int64(dl.limit)
}

// Wait 阻塞等待直到允许请求通过.
func (dl *DistributedLimiter) Wait(ctx context.Context) error {
	return dl.WaitWithKey(ctx, "default")
}

// WaitN 阻塞等待直到允许 n 个请求通过.
func (dl *DistributedLimiter) WaitN(ctx context.Context, n int) error {
	return dl.WaitNWithKey(ctx, "default", n)
}

// WaitWithKey 阻塞等待指定键直到允许请求通过.
func (dl *DistributedLimiter) WaitWithKey(ctx context.Context, key string) error {
	return dl.WaitNWithKey(ctx, key, 1)
}

// WaitNWithKey 阻塞等待指定键直到允许 n 个请求通过.
func (dl *DistributedLimiter) WaitNWithKey(ctx context.Context, key string, n int) error {
	for {
		if dl.AllowNWithKey(ctx, key, n) {
			return nil
		}

		// 获取剩余等待时间
		cacheKey := fmt.Sprintf("%s:%s", dl.prefix, key)
		ttl, err := dl.cache.TTL(ctx, cacheKey)
		if err != nil || ttl <= 0 {
			ttl = time.Millisecond * 100
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(ttl):
			// 继续尝试
		}
	}
}

// KeyedDistributedLimiter 基于键的分布式限流器工厂.
type KeyedDistributedLimiter struct {
	cache  cache.Cache
	prefix string
	limit  int
	window time.Duration
}

// NewKeyedDistributedLimiter 创建基于键的分布式限流器工厂.
func NewKeyedDistributedLimiter(cfg *DistributedConfig) (*KeyedDistributedLimiter, error) {
	if cfg == nil {
		return nil, ErrInvalidConfig
	}
	if cfg.Cache == nil {
		return nil, ErrNilCache
	}
	if cfg.Limit <= 0 {
		return nil, fmt.Errorf("%w: limit 必须大于 0", ErrInvalidConfig)
	}
	if cfg.Window <= 0 {
		return nil, fmt.Errorf("%w: window 必须大于 0", ErrInvalidConfig)
	}

	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "ratelimit"
	}

	return &KeyedDistributedLimiter{
		cache:  cfg.Cache,
		prefix: prefix,
		limit:  cfg.Limit,
		window: cfg.Window,
	}, nil
}

// GetLimiter 获取指定键的限流器.
//
// 返回 KeyedLimiterFunc 以便与 KeyedEndpointMiddleware 等配合使用.
func (kdl *KeyedDistributedLimiter) GetLimiter(key string) Limiter {
	return &keyedDistributedLimiterInstance{
		cache:  kdl.cache,
		key:    fmt.Sprintf("%s:%s", kdl.prefix, key),
		limit:  kdl.limit,
		window: kdl.window,
	}
}

type keyedDistributedLimiterInstance struct {
	cache  cache.Cache
	key    string
	limit  int
	window time.Duration
}

func (i *keyedDistributedLimiterInstance) Allow(ctx context.Context) bool {
	return i.AllowN(ctx, 1)
}

func (i *keyedDistributedLimiterInstance) AllowN(ctx context.Context, n int) bool {
	count, err := i.cache.IncrementBy(ctx, i.key, int64(n))
	if err != nil {
		return true
	}

	if count == int64(n) {
		_ = i.cache.Expire(ctx, i.key, i.window)
	}

	return count <= int64(i.limit)
}

func (i *keyedDistributedLimiterInstance) Wait(ctx context.Context) error {
	return i.WaitN(ctx, 1)
}

func (i *keyedDistributedLimiterInstance) WaitN(ctx context.Context, n int) error {
	for {
		if i.AllowN(ctx, n) {
			return nil
		}

		ttl, err := i.cache.TTL(ctx, i.key)
		if err != nil || ttl <= 0 {
			ttl = time.Millisecond * 100
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(ttl):
		}
	}
}
