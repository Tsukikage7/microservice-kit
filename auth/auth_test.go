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

func TestMemoryAuthenticator(t *testing.T) {
	auth := NewMemoryAuthenticator()
	ctx := context.Background()

	principal := &Principal{
		ID:   "user-123",
		Type: PrincipalTypeUser,
	}

	// 注册令牌
	auth.Register("token-123", principal)

	// 认证成功
	creds := Credentials{Token: "token-123"}
	got, err := auth.Authenticate(ctx, creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != principal.ID {
		t.Errorf("got ID = %v, want %v", got.ID, principal.ID)
	}

	// 认证失败 - 无效令牌
	_, err = auth.Authenticate(ctx, Credentials{Token: "invalid"})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}

	// 取消注册
	auth.Unregister("token-123")
	_, err = auth.Authenticate(ctx, creds)
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials after unregister, got %v", err)
	}
}

func TestMemoryAuthenticator_Expired(t *testing.T) {
	auth := NewMemoryAuthenticator()
	ctx := context.Background()

	expired := time.Now().Add(-time.Hour)
	principal := &Principal{
		ID:        "user-123",
		ExpiresAt: &expired,
	}

	auth.Register("token-123", principal)

	_, err := auth.Authenticate(ctx, Credentials{Token: "token-123"})
	if err != ErrCredentialsExpired {
		t.Errorf("expected ErrCredentialsExpired, got %v", err)
	}
}

func TestChainAuthenticator(t *testing.T) {
	ctx := context.Background()

	auth1 := NewMemoryAuthenticator()
	auth1.Register("token-1", &Principal{ID: "user-1"})

	auth2 := NewMemoryAuthenticator()
	auth2.Register("token-2", &Principal{ID: "user-2"})

	chain := NewChainAuthenticator(auth1, auth2)

	// 第一个认证器成功
	got, err := chain.Authenticate(ctx, Credentials{Token: "token-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "user-1" {
		t.Errorf("got ID = %v, want user-1", got.ID)
	}

	// 第二个认证器成功
	got, err = chain.Authenticate(ctx, Credentials{Token: "token-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "user-2" {
		t.Errorf("got ID = %v, want user-2", got.ID)
	}

	// 都失败
	_, err = chain.Authenticate(ctx, Credentials{Token: "invalid"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestChainAuthenticator_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("should panic with no authenticators")
		}
	}()

	NewChainAuthenticator()
}

func TestFuncAuthenticator(t *testing.T) {
	ctx := context.Background()

	auth := NewFuncAuthenticator(func(_ context.Context, creds Credentials) (*Principal, error) {
		if creds.Token == "valid" {
			return &Principal{ID: "user-123"}, nil
		}
		return nil, ErrInvalidCredentials
	})

	// 成功
	got, err := auth.Authenticate(ctx, Credentials{Token: "valid"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "user-123" {
		t.Errorf("got ID = %v, want user-123", got.ID)
	}

	// 失败
	_, err = auth.Authenticate(ctx, Credentials{Token: "invalid"})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
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
	auth := NewRoleAuthorizer([]string{"admin", "editor"}, WithRequireAllRoles(true))

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

func TestAllowAllAuthorizer(t *testing.T) {
	ctx := context.Background()
	auth := NewAllowAllAuthorizer()

	if err := auth.Authorize(ctx, nil, "any", "any"); err != nil {
		t.Errorf("should always allow: %v", err)
	}
}

func TestDenyAllAuthorizer(t *testing.T) {
	ctx := context.Background()
	auth := NewDenyAllAuthorizer()

	if err := auth.Authorize(ctx, &Principal{ID: "user"}, "any", "any"); err == nil {
		t.Error("should always deny")
	}
}

func TestChainAuthorizer(t *testing.T) {
	ctx := context.Background()

	roleAuth := NewRoleAuthorizer([]string{"admin"})
	permAuth := NewPermissionAuthorizer([]string{"read:orders"})

	chain := NewChainAuthorizer(roleAuth, permAuth)

	// 都通过
	principal := &Principal{
		Roles:       []string{"admin"},
		Permissions: []string{"read:orders"},
	}
	if err := chain.Authorize(ctx, principal, "", ""); err != nil {
		t.Errorf("should authorize: %v", err)
	}

	// 角色不通过
	principal = &Principal{
		Roles:       []string{"user"},
		Permissions: []string{"read:orders"},
	}
	if err := chain.Authorize(ctx, principal, "", ""); err == nil {
		t.Error("should not authorize - missing role")
	}

	// 权限不通过
	principal = &Principal{
		Roles:       []string{"admin"},
		Permissions: []string{"write:orders"},
	}
	if err := chain.Authorize(ctx, principal, "", ""); err == nil {
		t.Error("should not authorize - missing permission")
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
