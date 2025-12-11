package auth

import (
	"context"
	"testing"
	"time"
)

func TestPrincipal_HasRole(t *testing.T) {
	tests := []struct {
		name      string
		principal *Principal
		role      string
		want      bool
	}{
		{
			name: "has role",
			principal: &Principal{
				Roles: []string{"admin", "user"},
			},
			role: "admin",
			want: true,
		},
		{
			name: "does not have role",
			principal: &Principal{
				Roles: []string{"user"},
			},
			role: "admin",
			want: false,
		},
		{
			name: "empty roles",
			principal: &Principal{
				Roles: []string{},
			},
			role: "admin",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.principal.HasRole(tt.role); got != tt.want {
				t.Errorf("Principal.HasRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrincipal_HasPermission(t *testing.T) {
	tests := []struct {
		name       string
		principal  *Principal
		permission string
		want       bool
	}{
		{
			name: "has permission",
			principal: &Principal{
				Permissions: []string{"read:orders", "write:orders"},
			},
			permission: "read:orders",
			want:       true,
		},
		{
			name: "does not have permission",
			principal: &Principal{
				Permissions: []string{"read:orders"},
			},
			permission: "write:orders",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.principal.HasPermission(tt.permission); got != tt.want {
				t.Errorf("Principal.HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrincipal_HasAnyRole(t *testing.T) {
	principal := &Principal{
		Roles: []string{"user", "editor"},
	}

	if !principal.HasAnyRole("admin", "user") {
		t.Error("should have any role")
	}

	if principal.HasAnyRole("admin", "superuser") {
		t.Error("should not have any role")
	}
}

func TestPrincipal_HasAllRoles(t *testing.T) {
	principal := &Principal{
		Roles: []string{"user", "editor", "admin"},
	}

	if !principal.HasAllRoles("user", "editor") {
		t.Error("should have all roles")
	}

	if principal.HasAllRoles("user", "superuser") {
		t.Error("should not have all roles")
	}
}

func TestPrincipal_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		principal *Principal
		want      bool
	}{
		{
			name:      "no expiry",
			principal: &Principal{},
			want:      false,
		},
		{
			name: "not expired",
			principal: &Principal{
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(time.Hour)
					return &t
				}(),
			},
			want: false,
		},
		{
			name: "expired",
			principal: &Principal{
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(-time.Hour)
					return &t
				}(),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.principal.IsExpired(); got != tt.want {
				t.Errorf("Principal.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContext(t *testing.T) {
	ctx := context.Background()

	// 测试无 principal
	if _, ok := FromContext(ctx); ok {
		t.Error("should not have principal")
	}

	// 测试有 principal
	principal := &Principal{
		ID:    "user-123",
		Type:  PrincipalTypeUser,
		Roles: []string{"admin"},
	}
	ctx = WithPrincipal(ctx, principal)

	got, ok := FromContext(ctx)
	if !ok {
		t.Error("should have principal")
	}
	if got.ID != principal.ID {
		t.Errorf("got ID = %v, want %v", got.ID, principal.ID)
	}

	// 测试便捷函数
	if !HasRole(ctx, "admin") {
		t.Error("should have admin role")
	}
	if HasRole(ctx, "user") {
		t.Error("should not have user role")
	}

	id, ok := GetPrincipalID(ctx)
	if !ok || id != "user-123" {
		t.Errorf("GetPrincipalID() = %v, %v", id, ok)
	}
}

func TestMustFromContext_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("should panic")
		}
	}()

	ctx := context.Background()
	MustFromContext(ctx)
}

func TestRoleAuthorizer(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		roles     []string
		principal *Principal
		wantErr   bool
	}{
		{
			name:  "has required role",
			roles: []string{"admin"},
			principal: &Principal{
				Roles: []string{"admin", "user"},
			},
			wantErr: false,
		},
		{
			name:  "does not have required role",
			roles: []string{"superuser"},
			principal: &Principal{
				Roles: []string{"admin", "user"},
			},
			wantErr: true,
		},
		{
			name:      "nil principal",
			roles:     []string{"admin"},
			principal: nil,
			wantErr:   true,
		},
		{
			name:  "empty required roles",
			roles: []string{},
			principal: &Principal{
				Roles: []string{"user"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewRoleAuthorizer(tt.roles)
			err := auth.Authorize(ctx, tt.principal, "", "")
			if (err != nil) != tt.wantErr {
				t.Errorf("RoleAuthorizer.Authorize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoleAuthorizer_RequireAll(t *testing.T) {
	ctx := context.Background()
	auth := NewRoleAuthorizer([]string{"admin", "editor"}, true)

	// 有所有角色
	principal := &Principal{Roles: []string{"admin", "editor", "user"}}
	if err := auth.Authorize(ctx, principal, "", ""); err != nil {
		t.Errorf("should authorize: %v", err)
	}

	// 缺少角色
	principal = &Principal{Roles: []string{"admin"}}
	if err := auth.Authorize(ctx, principal, "", ""); err == nil {
		t.Error("should not authorize")
	}
}

func TestPermissionAuthorizer(t *testing.T) {
	ctx := context.Background()
	auth := NewPermissionAuthorizer([]string{"read:orders", "write:orders"})

	// 有权限
	principal := &Principal{Permissions: []string{"read:orders"}}
	if err := auth.Authorize(ctx, principal, "", ""); err != nil {
		t.Errorf("should authorize: %v", err)
	}

	// 无权限
	principal = &Principal{Permissions: []string{"delete:orders"}}
	if err := auth.Authorize(ctx, principal, "", ""); err == nil {
		t.Error("should not authorize")
	}
}

func TestMiddleware_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("should panic with nil authenticator")
		}
	}()

	Middleware(nil)
}

func TestIsUnauthenticated(t *testing.T) {
	if !IsUnauthenticated(ErrUnauthenticated) {
		t.Error("should be unauthenticated")
	}
	if IsUnauthenticated(ErrForbidden) {
		t.Error("should not be unauthenticated")
	}
}

func TestIsForbidden(t *testing.T) {
	if !IsForbidden(ErrForbidden) {
		t.Error("should be forbidden")
	}
	if IsForbidden(ErrUnauthenticated) {
		t.Error("should not be forbidden")
	}
}

func TestCredentialsFromContext(t *testing.T) {
	ctx := context.Background()

	// 无凭据
	if _, ok := CredentialsFromContext(ctx); ok {
		t.Error("should not have credentials")
	}

	// 有凭据
	creds := &Credentials{
		Type:  CredentialTypeBearer,
		Token: "test-token",
	}
	ctx = WithCredentials(ctx, creds)

	got, ok := CredentialsFromContext(ctx)
	if !ok {
		t.Error("should have credentials")
	}
	if got.Token != creds.Token {
		t.Errorf("got Token = %v, want %v", got.Token, creds.Token)
	}
}

func TestPrincipal_GetMetadata(t *testing.T) {
	principal := &Principal{
		Metadata: map[string]any{
			"key1": "value1",
			"key2": 123,
		},
	}

	// 获取存在的 key
	v, ok := principal.GetMetadata("key1")
	if !ok || v != "value1" {
		t.Errorf("GetMetadata(key1) = %v, %v", v, ok)
	}

	// 获取不存在的 key
	_, ok = principal.GetMetadata("key3")
	if ok {
		t.Error("should not have key3")
	}

	// GetMetadataString
	s := principal.GetMetadataString("key1")
	if s != "value1" {
		t.Errorf("GetMetadataString(key1) = %v", s)
	}

	s = principal.GetMetadataString("key2") // 非字符串类型
	if s != "" {
		t.Errorf("GetMetadataString(key2) should return empty, got %v", s)
	}

	// nil metadata
	p2 := &Principal{}
	_, ok = p2.GetMetadata("any")
	if ok {
		t.Error("should return false for nil metadata")
	}
}
