package semaphore

import (
	"context"
	"time"

	"github.com/Tsukikage7/microservice-kit/cache"
)

// Redis 分布式信号量.
//
// 基于 Redis 实现，适用于分布式并发控制.
type Redis struct {
	cache     cache.Cache
	key       string
	size      int64
	ttl       time.Duration
	retryWait time.Duration
}

// RedisOption Redis 信号量配置选项.
type RedisOption func(*Redis)

// WithTTL 设置许可的过期时间.
//
// 防止因客户端崩溃导致许可无法释放.
// 默认 30 秒.
func WithTTL(ttl time.Duration) RedisOption {
	return func(s *Redis) {
		s.ttl = ttl
	}
}

// WithRetryWait 设置重试等待时间.
//
// 当无法获取许可时，等待多长时间后重试.
// 默认 100ms.
func WithRetryWait(wait time.Duration) RedisOption {
	return func(s *Redis) {
		s.retryWait = wait
	}
}

// NewRedis 创建分布式信号量.
//
// c: Redis 缓存客户端
// key: 信号量唯一标识
// size: 最大并发数
func NewRedis(c cache.Cache, key string, size int64, opts ...RedisOption) *Redis {
	if c == nil {
		panic("semaphore: 缓存实例不能为空")
	}
	if size <= 0 {
		panic("semaphore: 信号量大小必须为正数")
	}

	s := &Redis{
		cache:     c,
		key:       "semaphore:" + key,
		size:      size,
		ttl:       30 * time.Second,
		retryWait: 100 * time.Millisecond,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Acquire 获取一个许可.
func (s *Redis) Acquire(ctx context.Context) error {
	for {
		if s.TryAcquire(ctx) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.retryWait):
			// 重试
		}
	}
}

// TryAcquire 尝试获取一个许可.
func (s *Redis) TryAcquire(ctx context.Context) bool {
	// 使用 INCR 原子增加计数
	count, err := s.cache.Increment(ctx, s.key)
	if err != nil {
		return false
	}

	// 首次创建时设置过期时间
	if count == 1 {
		_ = s.cache.Expire(ctx, s.key, s.ttl)
	}

	// 检查是否超过限制
	if count > s.size {
		// 超过限制，回退
		_, _ = s.cache.Decrement(ctx, s.key)
		return false
	}

	// 刷新过期时间
	_ = s.cache.Expire(ctx, s.key, s.ttl)
	return true
}

// Release 释放一个许可.
func (s *Redis) Release(ctx context.Context) error {
	_, err := s.cache.Decrement(ctx, s.key)
	return err
}

// Available 返回当前可用的许可数量.
func (s *Redis) Available(ctx context.Context) (int64, error) {
	val, err := s.cache.Get(ctx, s.key)
	if err != nil {
		// 键不存在，返回全部可用
		return s.size, nil
	}

	var current int64
	if val != "" {
		// 简单解析（假设值是数字字符串）
		for _, c := range val {
			if c >= '0' && c <= '9' {
				current = current*10 + int64(c-'0')
			}
		}
	}

	available := s.size - current
	if available < 0 {
		available = 0
	}
	return available, nil
}

// Size 返回信号量的总大小.
func (s *Redis) Size() int64 {
	return s.size
}
