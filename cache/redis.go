package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// redisCache Redis 缓存实现.
type redisCache struct {
	client *redis.Client
	config *Config
	logger logger.Logger
}

// NewRedisCache 创建 Redis 缓存.
func NewRedisCache(config *Config, log logger.Logger) (Cache, error) {
	if config == nil {
		return nil, ErrNilConfig
	}

	if config.Addr == "" {
		return nil, ErrEmptyAddr
	}

	config.ApplyDefaults()

	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		DialTimeout:  config.Timeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		MaxRetries:   config.MaxRetries,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Errorf("[cache] Redis 连接失败: addr=%s, err=%v", config.Addr, err)
		return nil, fmt.Errorf("redis 连接失败: %w", err)
	}

	log.Debugf("[cache] redis connected: addr=%s, db=%d", config.Addr, config.DB)

	return &redisCache{
		client: client,
		config: config,
		logger: log,
	}, nil
}

// Set 设置键值对.
func (r *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := r.serialize(value)
	if err != nil {
		return err
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		r.logger.Errorf("[cache] SET failed: key=%s, err=%v", key, err)
		return err
	}
	return nil
}

// Get 获取值.
func (r *redisCache) Get(ctx context.Context, key string) (string, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrNotFound
		}
		r.logger.Errorf("[cache] GET failed: key=%s, err=%v", key, err)
		return "", err
	}
	return result, nil
}

// Del 删除键.
func (r *redisCache) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		r.logger.Errorf("[cache] DEL failed: keys=%v, err=%v", keys, err)
		return err
	}
	return nil
}

// Exists 检查键是否存在.
func (r *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		r.logger.Errorf("[cache] EXISTS failed: key=%s, err=%v", key, err)
		return false, err
	}
	return result > 0, nil
}

// SetNX 仅当键不存在时设置.
func (r *redisCache) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	data, err := r.serialize(value)
	if err != nil {
		return false, err
	}

	result, err := r.client.SetNX(ctx, key, data, ttl).Result()
	if err != nil {
		r.logger.Errorf("[cache] SETNX failed: key=%s, err=%v", key, err)
		return false, err
	}
	return result, nil
}

// Increment 递增.
func (r *redisCache) Increment(ctx context.Context, key string) (int64, error) {
	result, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		r.logger.Errorf("[cache] INCR failed: key=%s, err=%v", key, err)
		return 0, err
	}
	return result, nil
}

// IncrementBy 增加指定值.
func (r *redisCache) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	result, err := r.client.IncrBy(ctx, key, value).Result()
	if err != nil {
		r.logger.Errorf("[cache] INCRBY failed: key=%s, err=%v", key, err)
		return 0, err
	}
	return result, nil
}

// Decrement 递减.
func (r *redisCache) Decrement(ctx context.Context, key string) (int64, error) {
	result, err := r.client.Decr(ctx, key).Result()
	if err != nil {
		r.logger.Errorf("[cache] DECR failed: key=%s, err=%v", key, err)
		return 0, err
	}
	return result, nil
}

// Expire 设置过期时间.
func (r *redisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
		r.logger.Errorf("[cache] EXPIRE failed: key=%s, err=%v", key, err)
		return err
	}
	return nil
}

// TTL 获取剩余过期时间.
func (r *redisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	result, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		r.logger.Errorf("[cache] TTL failed: key=%s, err=%v", key, err)
		return 0, err
	}
	return result, nil
}

// TryLock 尝试获取分布式锁.
func (r *redisCache) TryLock(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	result, err := r.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		r.logger.Errorf("[cache] TryLock failed: key=%s, err=%v", key, err)
		return false, err
	}
	return result, nil
}

// Unlock 释放分布式锁.
func (r *redisCache) Unlock(ctx context.Context, key string, value string) error {
	// Lua 脚本：只有当锁的值匹配时才删除
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, r.client, []string{key}, value).Result()
	if err != nil {
		r.logger.Errorf("[cache] Unlock failed: key=%s, err=%v", key, err)
		return err
	}

	if result.(int64) == 0 {
		r.logger.Warnf("[cache] Unlock skipped: key=%s, reason=value mismatch or expired", key)
		return ErrLockNotHeld
	}

	return nil
}

// MGet 批量获取.
func (r *redisCache) MGet(ctx context.Context, keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, nil
	}

	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		r.logger.Errorf("[cache] MGET failed: keys=%v, err=%v", keys, err)
		return nil, err
	}

	values := make([]string, len(results))
	for i, v := range results {
		if v != nil {
			values[i] = v.(string)
		}
	}
	return values, nil
}

// MSet 批量设置.
func (r *redisCache) MSet(ctx context.Context, pairs map[string]any, ttl time.Duration) error {
	if len(pairs) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()

	for key, value := range pairs {
		data, err := r.serialize(value)
		if err != nil {
			return err
		}
		pipe.Set(ctx, key, data, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.Errorf("[cache] MSET failed: err=%v", err)
		return err
	}
	return nil
}

// Ping 测试连接.
func (r *redisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close 关闭连接.
func (r *redisCache) Close() error {
	if err := r.client.Close(); err != nil {
		r.logger.Errorf("[cache] redis close failed: err=%v", err)
		return err
	}
	r.logger.Debug("[cache] redis connection closed")
	return nil
}

// Client 返回底层客户端.
func (r *redisCache) Client() any {
	return r.client
}

// serialize 序列化值.
func (r *redisCache) serialize(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("序列化值失败: %w", err)
		}
		return string(data), nil
	}
}
