package jwt

import (
	"time"

	"github.com/Tsukikage7/microservice-kit/cache"
	"github.com/Tsukikage7/microservice-kit/logger"
)

// Option JWT 配置选项.
type Option func(*options)

// options JWT 内部配置.
type options struct {
	name            string
	secretKey       string
	issuer          string
	accessDuration  time.Duration
	refreshDuration time.Duration
	refreshWindow   time.Duration
	tokenPrefix     string
	cacheKeyPrefix  string
	cache           cache.Cache
	logger          logger.Logger
	whitelist       *Whitelist
}

// defaultOptions 返回默认配置.
func defaultOptions() *options {
	return &options{
		name:            "JWT",
		accessDuration:  2 * time.Hour,
		refreshDuration: 7 * 24 * time.Hour,
		refreshWindow:   1 * time.Hour,
		tokenPrefix:     "Bearer ",
		cacheKeyPrefix:  "jwt:token:",
	}
}

// WithName 设置 JWT 服务名称.
func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// WithSecretKey 设置签名密钥.
func WithSecretKey(key string) Option {
	return func(o *options) {
		o.secretKey = key
	}
}

// WithIssuer 设置签发者.
func WithIssuer(issuer string) Option {
	return func(o *options) {
		o.issuer = issuer
	}
}

// WithAccessDuration 设置访问令牌有效期.
//
// 默认: 2 小时.
func WithAccessDuration(d time.Duration) Option {
	return func(o *options) {
		o.accessDuration = d
	}
}

// WithRefreshDuration 设置刷新令牌有效期.
//
// 默认: 7 天.
func WithRefreshDuration(d time.Duration) Option {
	return func(o *options) {
		o.refreshDuration = d
	}
}

// WithRefreshWindow 设置过期后可刷新窗口.
//
// 默认: 1 小时.
func WithRefreshWindow(d time.Duration) Option {
	return func(o *options) {
		o.refreshWindow = d
	}
}

// WithTokenPrefix 设置令牌前缀.
//
// 默认: "Bearer ".
func WithTokenPrefix(prefix string) Option {
	return func(o *options) {
		o.tokenPrefix = prefix
	}
}

// WithCacheKeyPrefix 设置缓存 key 前缀.
//
// 默认: "jwt:token:".
func WithCacheKeyPrefix(prefix string) Option {
	return func(o *options) {
		o.cacheKeyPrefix = prefix
	}
}

// WithCache 设置缓存（用于令牌存储和撤销）.
func WithCache(c cache.Cache) Option {
	return func(o *options) {
		o.cache = c
	}
}

// WithLogger 设置日志记录器.
func WithLogger(log logger.Logger) Option {
	return func(o *options) {
		o.logger = log
	}
}

// WithWhitelist 设置白名单.
func WithWhitelist(w *Whitelist) Option {
	return func(o *options) {
		o.whitelist = w
	}
}
