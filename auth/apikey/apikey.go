// Package apikey 提供基于 API Key 的认证器实现.
//
// API Key 认证适用于服务间调用或第三方集成场景.
//
// 示例:
//
//	// 使用内存存储
//	store := apikey.NewMemoryStore()
//	store.Register("sk_live_xxx", &auth.Principal{
//	    ID:   "service-1",
//	    Type: auth.PrincipalTypeService,
//	})
//
//	authenticator := apikey.NewAuthenticator(store)
package apikey

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"sync"
	"time"

	"github.com/Tsukikage7/microservice-kit/auth"
)

// Store API Key 存储接口.
type Store interface {
	// Get 根据 API Key 获取关联的主体.
	Get(ctx context.Context, key string) (*auth.Principal, error)
}

// Authenticator API Key 认证器.
type Authenticator struct {
	store    Store
	hashKeys bool // 是否对 key 进行哈希处理
}

// Option 认证器选项.
type Option func(*Authenticator)

// WithHashKeys 设置是否对 key 进行哈希处理.
//
// 启用后，存储中保存的是 key 的 SHA256 哈希值，
// 提供额外的安全性。
func WithHashKeys(hash bool) Option {
	return func(a *Authenticator) {
		a.hashKeys = hash
	}
}

// NewAuthenticator 创建 API Key 认证器.
func NewAuthenticator(store Store, opts ...Option) *Authenticator {
	if store == nil {
		panic("apikey: 存储实例不能为空")
	}

	a := &Authenticator{
		store:    store,
		hashKeys: false,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Authenticate 实现 auth.Authenticator 接口.
func (a *Authenticator) Authenticate(ctx context.Context, creds auth.Credentials) (*auth.Principal, error) {
	if creds.Type != "" && creds.Type != auth.CredentialTypeAPIKey {
		return nil, auth.ErrInvalidCredentials
	}

	if creds.Token == "" {
		return nil, auth.ErrCredentialsNotFound
	}

	key := creds.Token
	if a.hashKeys {
		key = hashKey(key)
	}

	principal, err := a.store.Get(ctx, key)
	if err != nil {
		return nil, auth.ErrInvalidCredentials
	}

	if principal == nil {
		return nil, auth.ErrInvalidCredentials
	}

	if principal.IsExpired() {
		return nil, auth.ErrCredentialsExpired
	}

	return principal, nil
}

// hashKey 对 key 进行 SHA256 哈希.
func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// HashKey 对 key 进行 SHA256 哈希（导出函数）.
//
// 用于在注册 API Key 时生成哈希值.
func HashKey(key string) string {
	return hashKey(key)
}

// MemoryStore 内存 API Key 存储.
type MemoryStore struct {
	mu   sync.RWMutex
	keys map[string]*auth.Principal
}

// NewMemoryStore 创建内存存储.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		keys: make(map[string]*auth.Principal),
	}
}

// Register 注册 API Key.
func (m *MemoryStore) Register(key string, principal *auth.Principal) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys[key] = principal
}

// RegisterWithExpiry 注册带过期时间的 API Key.
func (m *MemoryStore) RegisterWithExpiry(key string, principal *auth.Principal, expiry time.Time) {
	p := *principal
	p.ExpiresAt = &expiry
	m.Register(key, &p)
}

// Unregister 取消注册 API Key.
func (m *MemoryStore) Unregister(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.keys, key)
}

// Get 实现 Store 接口.
func (m *MemoryStore) Get(_ context.Context, key string) (*auth.Principal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	principal, ok := m.keys[key]
	if !ok {
		return nil, auth.ErrCredentialsNotFound
	}

	return principal, nil
}

// SecureCompare 安全比较两个字符串.
//
// 使用恒定时间比较，防止时序攻击.
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
