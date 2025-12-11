// Package rbac 提供基于角色的访问控制 (RBAC) 授权器实现.
//
// RBAC 允许通过角色来管理权限，支持角色继承.
//
// 示例:
//
//	rbacAuth := rbac.New().
//	    AddRole(&rbac.Role{
//	        Name: "admin",
//	        Permissions: []rbac.Permission{
//	            {Action: "*", Resource: "*"},
//	        },
//	    }).
//	    AddRole(&rbac.Role{
//	        Name: "user",
//	        Permissions: []rbac.Permission{
//	            {Action: "read", Resource: "orders"},
//	            {Action: "create", Resource: "orders"},
//	        },
//	    }).
//	    AddRole(&rbac.Role{
//	        Name: "manager",
//	        Parents: []string{"user"},
//	        Permissions: []rbac.Permission{
//	            {Action: "update", Resource: "orders"},
//	            {Action: "delete", Resource: "orders"},
//	        },
//	    })
//
//	// 在中间件中使用
//	endpoint = auth.Middleware(authenticator,
//	    auth.WithAuthorizer(rbacAuth),
//	    auth.WithActionResource("read", "orders"),
//	)(endpoint)
package rbac

import (
	"context"
	"strings"
	"sync"

	"github.com/Tsukikage7/microservice-kit/auth"
)

// Permission 权限定义.
type Permission struct {
	// Action 操作类型: create, read, update, delete, * (全部).
	Action string

	// Resource 资源标识: orders, users, * (全部).
	Resource string
}

// Match 检查权限是否匹配.
func (p Permission) Match(action, resource string) bool {
	actionMatch := p.Action == "*" || p.Action == action
	resourceMatch := p.Resource == "*" || p.Resource == resource

	// 支持通配符前缀匹配，如 "orders:*" 匹配 "orders:123"
	if !resourceMatch && strings.HasSuffix(p.Resource, ":*") {
		prefix := strings.TrimSuffix(p.Resource, ":*")
		resourceMatch = strings.HasPrefix(resource, prefix+":")
	}

	return actionMatch && resourceMatch
}

// Role 角色定义.
type Role struct {
	// Name 角色名称.
	Name string

	// Description 角色描述.
	Description string

	// Permissions 角色拥有的权限.
	Permissions []Permission

	// Parents 继承的父角色名称.
	Parents []string
}

// RBAC 基于角色的访问控制授权器.
type RBAC struct {
	mu    sync.RWMutex
	roles map[string]*Role

	// 缓存角色的所有权限（包括继承的）
	permCache map[string][]Permission
}

// New 创建 RBAC 授权器.
func New() *RBAC {
	return &RBAC{
		roles:     make(map[string]*Role),
		permCache: make(map[string][]Permission),
	}
}

// AddRole 添加角色.
func (r *RBAC) AddRole(role *Role) *RBAC {
	if role == nil || role.Name == "" {
		panic("rbac: 角色名称不能为空")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.roles[role.Name] = role

	// 清除权限缓存
	r.permCache = make(map[string][]Permission)

	return r
}

// RemoveRole 移除角色.
func (r *RBAC) RemoveRole(name string) *RBAC {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.roles, name)

	// 清除权限缓存
	r.permCache = make(map[string][]Permission)

	return r
}

// GetRole 获取角色.
func (r *RBAC) GetRole(name string) (*Role, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	role, ok := r.roles[name]
	return role, ok
}

// Authorize 实现 auth.Authorizer 接口.
func (r *RBAC) Authorize(_ context.Context, principal *auth.Principal, action, resource string) error {
	if principal == nil {
		return auth.ErrUnauthenticated
	}

	// 检查主体的直接权限
	for _, perm := range principal.Permissions {
		if matchPermissionString(perm, action, resource) {
			return nil
		}
	}

	// 检查主体的角色权限
	for _, roleName := range principal.Roles {
		permissions := r.getRolePermissions(roleName)
		for _, perm := range permissions {
			if perm.Match(action, resource) {
				return nil
			}
		}
	}

	return auth.ErrForbidden
}

// getRolePermissions 获取角色的所有权限（包括继承的）.
func (r *RBAC) getRolePermissions(roleName string) []Permission {
	r.mu.RLock()

	// 检查缓存
	if cached, ok := r.permCache[roleName]; ok {
		r.mu.RUnlock()
		return cached
	}
	r.mu.RUnlock()

	// 收集权限
	permissions := r.collectPermissions(roleName, make(map[string]bool))

	// 缓存结果
	r.mu.Lock()
	r.permCache[roleName] = permissions
	r.mu.Unlock()

	return permissions
}

// collectPermissions 递归收集角色权限.
func (r *RBAC) collectPermissions(roleName string, visited map[string]bool) []Permission {
	// 防止循环引用
	if visited[roleName] {
		return nil
	}
	visited[roleName] = true

	r.mu.RLock()
	role, ok := r.roles[roleName]
	r.mu.RUnlock()

	if !ok {
		return nil
	}

	// 收集当前角色的权限
	permissions := make([]Permission, len(role.Permissions))
	copy(permissions, role.Permissions)

	// 收集父角色的权限
	for _, parent := range role.Parents {
		parentPerms := r.collectPermissions(parent, visited)
		permissions = append(permissions, parentPerms...)
	}

	return permissions
}

// matchPermissionString 匹配权限字符串.
//
// 支持格式: "action:resource" 或 "*:*" 或 "action:*" 或 "*:resource".
func matchPermissionString(perm, action, resource string) bool {
	// 格式: "action:resource"
	if strings.Contains(perm, ":") {
		parts := strings.SplitN(perm, ":", 2)
		if len(parts) == 2 {
			p := Permission{Action: parts[0], Resource: parts[1]}
			return p.Match(action, resource)
		}
	}

	// 简单权限名匹配
	return perm == "*" || perm == action+":"+resource
}

// HasPermission 检查角色是否有指定权限.
func (r *RBAC) HasPermission(roleName, action, resource string) bool {
	permissions := r.getRolePermissions(roleName)
	for _, perm := range permissions {
		if perm.Match(action, resource) {
			return true
		}
	}
	return false
}

// GetAllRoles 获取所有角色名称.
func (r *RBAC) GetAllRoles() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	roles := make([]string, 0, len(r.roles))
	for name := range r.roles {
		roles = append(roles, name)
	}
	return roles
}
