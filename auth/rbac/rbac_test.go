package rbac

import (
	"context"
	"testing"

	"github.com/Tsukikage7/microservice-kit/auth"
)

func TestPermission_Match(t *testing.T) {
	tests := []struct {
		name     string
		perm     Permission
		action   string
		resource string
		want     bool
	}{
		{
			name:     "exact match",
			perm:     Permission{Action: "read", Resource: "orders"},
			action:   "read",
			resource: "orders",
			want:     true,
		},
		{
			name:     "wildcard action",
			perm:     Permission{Action: "*", Resource: "orders"},
			action:   "write",
			resource: "orders",
			want:     true,
		},
		{
			name:     "wildcard resource",
			perm:     Permission{Action: "read", Resource: "*"},
			action:   "read",
			resource: "anything",
			want:     true,
		},
		{
			name:     "all wildcard",
			perm:     Permission{Action: "*", Resource: "*"},
			action:   "delete",
			resource: "users",
			want:     true,
		},
		{
			name:     "no match",
			perm:     Permission{Action: "read", Resource: "orders"},
			action:   "write",
			resource: "orders",
			want:     false,
		},
		{
			name:     "prefix wildcard",
			perm:     Permission{Action: "read", Resource: "orders:*"},
			action:   "read",
			resource: "orders:123",
			want:     true,
		},
		{
			name:     "prefix wildcard no match",
			perm:     Permission{Action: "read", Resource: "orders:*"},
			action:   "read",
			resource: "users:123",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.perm.Match(tt.action, tt.resource); got != tt.want {
				t.Errorf("Permission.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRBAC_AddRole(t *testing.T) {
	rbac := New()
	rbac.AddRole(&Role{
		Name:        "admin",
		Description: "管理员",
		Permissions: []Permission{
			{Action: "*", Resource: "*"},
		},
	})

	role, ok := rbac.GetRole("admin")
	if !ok {
		t.Fatal("role should exist")
	}
	if role.Name != "admin" {
		t.Errorf("got name = %v, want admin", role.Name)
	}
}

func TestRBAC_AddRole_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("should panic with empty role name")
		}
	}()

	rbac := New()
	rbac.AddRole(&Role{Name: ""})
}

func TestRBAC_RemoveRole(t *testing.T) {
	rbac := New()
	rbac.AddRole(&Role{Name: "admin"})
	rbac.RemoveRole("admin")

	_, ok := rbac.GetRole("admin")
	if ok {
		t.Error("role should be removed")
	}
}

func TestRBAC_Authorize(t *testing.T) {
	ctx := context.Background()

	rbac := New().
		AddRole(&Role{
			Name: "admin",
			Permissions: []Permission{
				{Action: "*", Resource: "*"},
			},
		}).
		AddRole(&Role{
			Name: "user",
			Permissions: []Permission{
				{Action: "read", Resource: "orders"},
				{Action: "create", Resource: "orders"},
			},
		})

	tests := []struct {
		name      string
		principal *auth.Principal
		action    string
		resource  string
		wantErr   bool
	}{
		{
			name: "admin has all permissions",
			principal: &auth.Principal{
				ID:    "admin-1",
				Roles: []string{"admin"},
			},
			action:   "delete",
			resource: "users",
			wantErr:  false,
		},
		{
			name: "user has read orders",
			principal: &auth.Principal{
				ID:    "user-1",
				Roles: []string{"user"},
			},
			action:   "read",
			resource: "orders",
			wantErr:  false,
		},
		{
			name: "user cannot delete orders",
			principal: &auth.Principal{
				ID:    "user-1",
				Roles: []string{"user"},
			},
			action:   "delete",
			resource: "orders",
			wantErr:  true,
		},
		{
			name:      "nil principal",
			principal: nil,
			action:    "read",
			resource:  "orders",
			wantErr:   true,
		},
		{
			name: "unknown role",
			principal: &auth.Principal{
				ID:    "user-1",
				Roles: []string{"unknown"},
			},
			action:   "read",
			resource: "orders",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rbac.Authorize(ctx, tt.principal, tt.action, tt.resource)
			if (err != nil) != tt.wantErr {
				t.Errorf("RBAC.Authorize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRBAC_Authorize_DirectPermission(t *testing.T) {
	ctx := context.Background()
	rbac := New()

	// 主体有直接权限，不通过角色
	principal := &auth.Principal{
		ID:          "user-1",
		Permissions: []string{"read:orders"},
	}

	if err := rbac.Authorize(ctx, principal, "read", "orders"); err != nil {
		t.Errorf("should authorize via direct permission: %v", err)
	}

	if err := rbac.Authorize(ctx, principal, "write", "orders"); err == nil {
		t.Error("should not authorize without permission")
	}
}

func TestRBAC_RoleInheritance(t *testing.T) {
	ctx := context.Background()

	rbac := New().
		AddRole(&Role{
			Name: "user",
			Permissions: []Permission{
				{Action: "read", Resource: "orders"},
			},
		}).
		AddRole(&Role{
			Name:    "manager",
			Parents: []string{"user"},
			Permissions: []Permission{
				{Action: "update", Resource: "orders"},
			},
		}).
		AddRole(&Role{
			Name:    "admin",
			Parents: []string{"manager"},
			Permissions: []Permission{
				{Action: "delete", Resource: "orders"},
			},
		})

	// manager 继承 user 的权限
	manager := &auth.Principal{
		ID:    "manager-1",
		Roles: []string{"manager"},
	}

	if err := rbac.Authorize(ctx, manager, "read", "orders"); err != nil {
		t.Errorf("manager should inherit read from user: %v", err)
	}
	if err := rbac.Authorize(ctx, manager, "update", "orders"); err != nil {
		t.Errorf("manager should have update: %v", err)
	}
	if err := rbac.Authorize(ctx, manager, "delete", "orders"); err == nil {
		t.Error("manager should not have delete")
	}

	// admin 继承 manager（和 user）的权限
	admin := &auth.Principal{
		ID:    "admin-1",
		Roles: []string{"admin"},
	}

	if err := rbac.Authorize(ctx, admin, "read", "orders"); err != nil {
		t.Errorf("admin should inherit read: %v", err)
	}
	if err := rbac.Authorize(ctx, admin, "update", "orders"); err != nil {
		t.Errorf("admin should inherit update: %v", err)
	}
	if err := rbac.Authorize(ctx, admin, "delete", "orders"); err != nil {
		t.Errorf("admin should have delete: %v", err)
	}
}

func TestRBAC_HasPermission(t *testing.T) {
	rbac := New().
		AddRole(&Role{
			Name: "admin",
			Permissions: []Permission{
				{Action: "*", Resource: "*"},
			},
		})

	if !rbac.HasPermission("admin", "read", "orders") {
		t.Error("admin should have read:orders")
	}
}

func TestRBAC_GetAllRoles(t *testing.T) {
	rbac := New().
		AddRole(&Role{Name: "admin"}).
		AddRole(&Role{Name: "user"})

	roles := rbac.GetAllRoles()
	if len(roles) != 2 {
		t.Errorf("got %d roles, want 2", len(roles))
	}
}

func TestRBAC_CircularInheritance(t *testing.T) {
	ctx := context.Background()

	// 测试循环继承不会导致无限循环
	rbac := New().
		AddRole(&Role{
			Name:    "a",
			Parents: []string{"b"},
			Permissions: []Permission{
				{Action: "read", Resource: "a"},
			},
		}).
		AddRole(&Role{
			Name:    "b",
			Parents: []string{"a"},
			Permissions: []Permission{
				{Action: "read", Resource: "b"},
			},
		})

	principal := &auth.Principal{
		ID:    "user-1",
		Roles: []string{"a"},
	}

	// 应该能正常完成，不会无限循环
	_ = rbac.Authorize(ctx, principal, "read", "a")
	_ = rbac.Authorize(ctx, principal, "read", "b")
}
