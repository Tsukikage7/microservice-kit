package apikey

import (
	"context"
	"testing"
	"time"

	"github.com/Tsukikage7/microservice-kit/auth"
)

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 注册
	principal := &auth.Principal{ID: "service-1"}
	store.Register("api-key-123", principal)

	// 获取
	got, err := store.Get(ctx, "api-key-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "service-1" {
		t.Errorf("got ID = %v, want service-1", got.ID)
	}

	// 不存在
	_, err = store.Get(ctx, "invalid")
	if err != auth.ErrCredentialsNotFound {
		t.Errorf("expected ErrCredentialsNotFound, got %v", err)
	}

	// 取消注册
	store.Unregister("api-key-123")
	_, err = store.Get(ctx, "api-key-123")
	if err != auth.ErrCredentialsNotFound {
		t.Errorf("expected ErrCredentialsNotFound after unregister, got %v", err)
	}
}

func TestMemoryStore_WithExpiry(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	expiry := time.Now().Add(time.Hour)
	principal := &auth.Principal{ID: "service-1"}
	store.RegisterWithExpiry("api-key-123", principal, expiry)

	got, err := store.Get(ctx, "api-key-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(expiry) {
		t.Error("expiry not set correctly")
	}
}

func TestAuthenticator(t *testing.T) {
	store := NewMemoryStore()
	store.Register("api-key-123", &auth.Principal{ID: "service-1"})

	authenticator := NewAuthenticator(store)
	ctx := context.Background()

	// 认证成功
	creds := auth.Credentials{
		Type:  auth.CredentialTypeAPIKey,
		Token: "api-key-123",
	}
	got, err := authenticator.Authenticate(ctx, creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "service-1" {
		t.Errorf("got ID = %v, want service-1", got.ID)
	}

	// 认证失败 - 无效 key
	_, err = authenticator.Authenticate(ctx, auth.Credentials{Token: "invalid"})
	if err != auth.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}

	// 认证失败 - 空 token
	_, err = authenticator.Authenticate(ctx, auth.Credentials{Token: ""})
	if err != auth.ErrCredentialsNotFound {
		t.Errorf("expected ErrCredentialsNotFound, got %v", err)
	}
}

func TestAuthenticator_WrongType(t *testing.T) {
	store := NewMemoryStore()
	authenticator := NewAuthenticator(store)
	ctx := context.Background()

	// 错误的凭据类型
	creds := auth.Credentials{
		Type:  auth.CredentialTypeBearer,
		Token: "some-token",
	}
	_, err := authenticator.Authenticate(ctx, creds)
	if err != auth.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for wrong type, got %v", err)
	}
}

func TestAuthenticator_Expired(t *testing.T) {
	store := NewMemoryStore()
	expired := time.Now().Add(-time.Hour)
	store.RegisterWithExpiry("api-key-123", &auth.Principal{ID: "service-1"}, expired)

	authenticator := NewAuthenticator(store)
	ctx := context.Background()

	_, err := authenticator.Authenticate(ctx, auth.Credentials{Token: "api-key-123"})
	if err != auth.ErrCredentialsExpired {
		t.Errorf("expected ErrCredentialsExpired, got %v", err)
	}
}

func TestAuthenticator_WithHashKeys(t *testing.T) {
	store := NewMemoryStore()
	hashedKey := HashKey("api-key-123")
	store.Register(hashedKey, &auth.Principal{ID: "service-1"})

	authenticator := NewAuthenticator(store, WithHashKeys(true))
	ctx := context.Background()

	// 使用原始 key 认证
	creds := auth.Credentials{Token: "api-key-123"}
	got, err := authenticator.Authenticate(ctx, creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "service-1" {
		t.Errorf("got ID = %v, want service-1", got.ID)
	}
}

func TestHashKey(t *testing.T) {
	key := "test-api-key"
	hash1 := HashKey(key)
	hash2 := HashKey(key)

	if hash1 != hash2 {
		t.Error("hash should be deterministic")
	}

	if hash1 == key {
		t.Error("hash should be different from original")
	}

	if len(hash1) != 64 { // SHA256 = 32 bytes = 64 hex chars
		t.Errorf("hash length = %d, want 64", len(hash1))
	}
}

func TestSecureCompare(t *testing.T) {
	if !SecureCompare("abc", "abc") {
		t.Error("equal strings should match")
	}
	if SecureCompare("abc", "def") {
		t.Error("different strings should not match")
	}
	if SecureCompare("abc", "abcd") {
		t.Error("different length strings should not match")
	}
}

func TestNewAuthenticator_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("should panic with nil store")
		}
	}()

	NewAuthenticator(nil)
}
