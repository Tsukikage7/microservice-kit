package idempotency

import (
	"context"
	"time"

	"github.com/Tsukikage7/microservice-kit/cache"
)

// RedisStore 基于 Redis 的幂等性存储.
//
// 适用于分布式部署场景.
type RedisStore struct {
	cache     cache.Cache
	keyPrefix string
}

// RedisStoreOption Redis 存储配置选项.
type RedisStoreOption func(*RedisStore)

// WithKeyPrefix 设置键前缀.
func WithKeyPrefix(prefix string) RedisStoreOption {
	return func(s *RedisStore) {
		s.keyPrefix = prefix
	}
}

// NewRedisStore 创建 Redis 存储.
func NewRedisStore(c cache.Cache, opts ...RedisStoreOption) *RedisStore {
	s := &RedisStore{
		cache:     c,
		keyPrefix: "idempotency:",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Get 获取幂等键对应的结果.
func (s *RedisStore) Get(ctx context.Context, key string) (*Result, error) {
	fullKey := s.keyPrefix + key

	data, err := s.cache.Get(ctx, fullKey)
	if err != nil {
		// 键不存在不是错误
		return nil, nil
	}
	if data == "" {
		return nil, nil
	}

	return DecodeResult([]byte(data))
}

// Set 设置幂等键和结果.
func (s *RedisStore) Set(ctx context.Context, key string, result *Result, ttl time.Duration) error {
	fullKey := s.keyPrefix + key
	lockKey := s.keyPrefix + "lock:" + key

	data, err := result.Encode()
	if err != nil {
		return err
	}

	// 设置结果
	if err := s.cache.Set(ctx, fullKey, string(data), ttl); err != nil {
		return err
	}

	// 删除锁
	_ = s.cache.Del(ctx, lockKey)

	return nil
}

// SetNX 仅在键不存在时设置（用于获取处理锁）.
func (s *RedisStore) SetNX(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	fullKey := s.keyPrefix + key
	lockKey := s.keyPrefix + "lock:" + key

	// 先检查是否已有结果
	exists, err := s.cache.Exists(ctx, fullKey)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	// 尝试获取锁
	return s.cache.SetNX(ctx, lockKey, "1", ttl)
}

// Delete 删除幂等键.
func (s *RedisStore) Delete(ctx context.Context, key string) error {
	fullKey := s.keyPrefix + key
	lockKey := s.keyPrefix + "lock:" + key
	return s.cache.Del(ctx, fullKey, lockKey)
}
