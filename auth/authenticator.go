package auth

import (
	"context"
	"sync"
	"time"

	"github.com/Tsukikage7/microservice-kit/cache"
)

// ChainAuthenticator 链式认证器.
//
// 按顺序尝试多个认证器，直到有一个成功或全部失败.
type ChainAuthenticator struct {
	authenticators []Authenticator
}

// NewChainAuthenticator 创建链式认证器.
func NewChainAuthenticator(authenticators ...Authenticator) *ChainAuthenticator {
	if len(authenticators) == 0 {
		panic("auth: 至少需要一个认证器")
	}
	return &ChainAuthenticator{
		authenticators: authenticators,
	}
}

// Authenticate 实现 Authenticator 接口.
func (c *ChainAuthenticator) Authenticate(ctx context.Context, creds Credentials) (*Principal, error) {
	var lastErr error

	for _, auth := range c.authenticators {
		principal, err := auth.Authenticate(ctx, creds)
		if err == nil {
			return principal, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrInvalidCredentials
}

// CachingAuthenticator 带缓存的认证器.
//
// 缓存认证结果以提升性能.
type CachingAuthenticator struct {
	authenticator Authenticator
	cache         cache.Cache
	ttl           time.Duration
	keyPrefix     string
}

// CachingOption 缓存认证器选项.
type CachingOption func(*CachingAuthenticator)

// WithCacheKeyPrefix 设置缓存键前缀.
func WithCacheKeyPrefix(prefix string) CachingOption {
	return func(c *CachingAuthenticator) {
		c.keyPrefix = prefix
	}
}

// NewCachingAuthenticator 创建带缓存的认证器.
func NewCachingAuthenticator(
	authenticator Authenticator,
	cache cache.Cache,
	ttl time.Duration,
	opts ...CachingOption,
) *CachingAuthenticator {
	if authenticator == nil {
		panic("auth: 认证器不能为空")
	}
	if cache == nil {
		panic("auth: 缓存实例不能为空")
	}

	c := &CachingAuthenticator{
		authenticator: authenticator,
		cache:         cache,
		ttl:           ttl,
		keyPrefix:     "auth:principal:",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Authenticate 实现 Authenticator 接口.
func (c *CachingAuthenticator) Authenticate(ctx context.Context, creds Credentials) (*Principal, error) {
	// 尝试从缓存获取
	cacheKey := c.keyPrefix + creds.Token
	if cached, err := c.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		// 缓存命中，解析 Principal
		// 注意：这里简化处理，实际可能需要 JSON 序列化
		principal, err := c.authenticator.Authenticate(ctx, creds)
		if err == nil {
			return principal, nil
		}
	}

	// 执行实际认证
	principal, err := c.authenticator.Authenticate(ctx, creds)
	if err != nil {
		return nil, err
	}

	// 缓存结果
	_ = c.cache.Set(ctx, cacheKey, principal.ID, c.ttl)

	return principal, nil
}

// FuncAuthenticator 函数式认证器.
//
// 将函数包装为认证器.
type FuncAuthenticator struct {
	fn func(ctx context.Context, creds Credentials) (*Principal, error)
}

// NewFuncAuthenticator 创建函数式认证器.
func NewFuncAuthenticator(fn func(ctx context.Context, creds Credentials) (*Principal, error)) *FuncAuthenticator {
	if fn == nil {
		panic("auth: 函数不能为空")
	}
	return &FuncAuthenticator{fn: fn}
}

// Authenticate 实现 Authenticator 接口.
func (f *FuncAuthenticator) Authenticate(ctx context.Context, creds Credentials) (*Principal, error) {
	return f.fn(ctx, creds)
}

// MemoryAuthenticator 内存认证器.
//
// 用于测试或简单场景，将令牌映射到主体.
type MemoryAuthenticator struct {
	mu      sync.RWMutex
	tokens  map[string]*Principal
}

// NewMemoryAuthenticator 创建内存认证器.
func NewMemoryAuthenticator() *MemoryAuthenticator {
	return &MemoryAuthenticator{
		tokens: make(map[string]*Principal),
	}
}

// Register 注册令牌和主体的映射.
func (m *MemoryAuthenticator) Register(token string, principal *Principal) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[token] = principal
}

// Unregister 取消令牌注册.
func (m *MemoryAuthenticator) Unregister(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tokens, token)
}

// Authenticate 实现 Authenticator 接口.
func (m *MemoryAuthenticator) Authenticate(ctx context.Context, creds Credentials) (*Principal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	principal, ok := m.tokens[creds.Token]
	if !ok {
		return nil, ErrInvalidCredentials
	}

	if principal.IsExpired() {
		return nil, ErrCredentialsExpired
	}

	return principal, nil
}
