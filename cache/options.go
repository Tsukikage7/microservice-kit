package cache

import "github.com/Tsukikage7/microservice-kit/logger"

// New 创建缓存实例.
// logger 是必需参数，不能为 nil.
func New(config *Config, log logger.Logger) (Cache, error) {
	if log == nil {
		return nil, ErrNilLogger
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	config.ApplyDefaults()

	switch config.Type {
	case TypeRedis:
		return NewRedisCache(config, log)
	case TypeMemory:
		return NewMemoryCache(config, log)
	default:
		return nil, ErrUnsupported
	}
}

// MustNew 创建缓存实例，失败时 panic.
func MustNew(config *Config, log logger.Logger) Cache {
	cache, err := New(config, log)
	if err != nil {
		panic(err)
	}
	return cache
}
