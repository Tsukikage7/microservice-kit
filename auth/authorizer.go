package auth

import "context"

// RoleAuthorizer 基于角色的简单授权器.
type RoleAuthorizer struct {
	requiredRoles []string
	requireAll    bool
}

// NewRoleAuthorizer 创建角色授权器.
//
// 默认只需要任一角色匹配即可，设置 requireAll=true 需要所有角色.
func NewRoleAuthorizer(roles []string, requireAll ...bool) *RoleAuthorizer {
	r := &RoleAuthorizer{
		requiredRoles: roles,
		requireAll:    false,
	}
	if len(requireAll) > 0 {
		r.requireAll = requireAll[0]
	}
	return r
}

// Authorize 实现 Authorizer 接口.
func (r *RoleAuthorizer) Authorize(_ context.Context, principal *Principal, _, _ string) error {
	if principal == nil {
		return ErrUnauthenticated
	}
	if len(r.requiredRoles) == 0 {
		return nil
	}
	if r.requireAll {
		if principal.HasAllRoles(r.requiredRoles...) {
			return nil
		}
	} else {
		if principal.HasAnyRole(r.requiredRoles...) {
			return nil
		}
	}
	return ErrForbidden
}

// PermissionAuthorizer 基于权限的简单授权器.
type PermissionAuthorizer struct {
	requiredPermissions []string
	requireAll          bool
}

// NewPermissionAuthorizer 创建权限授权器.
//
// 默认只需要任一权限匹配即可，设置 requireAll=true 需要所有权限.
func NewPermissionAuthorizer(permissions []string, requireAll ...bool) *PermissionAuthorizer {
	p := &PermissionAuthorizer{
		requiredPermissions: permissions,
		requireAll:          false,
	}
	if len(requireAll) > 0 {
		p.requireAll = requireAll[0]
	}
	return p
}

// Authorize 实现 Authorizer 接口.
func (p *PermissionAuthorizer) Authorize(_ context.Context, principal *Principal, _, _ string) error {
	if principal == nil {
		return ErrUnauthenticated
	}
	if len(p.requiredPermissions) == 0 {
		return nil
	}
	if p.requireAll {
		for _, perm := range p.requiredPermissions {
			if !principal.HasPermission(perm) {
				return ErrForbidden
			}
		}
		return nil
	}
	for _, perm := range p.requiredPermissions {
		if principal.HasPermission(perm) {
			return nil
		}
	}
	return ErrForbidden
}
