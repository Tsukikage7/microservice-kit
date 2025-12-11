package auth

import "context"

// AllowAllAuthorizer 允许所有请求的授权器.
//
// 用于测试或不需要授权的场景.
type AllowAllAuthorizer struct{}

// NewAllowAllAuthorizer 创建允许所有请求的授权器.
func NewAllowAllAuthorizer() *AllowAllAuthorizer {
	return &AllowAllAuthorizer{}
}

// Authorize 实现 Authorizer 接口.
func (a *AllowAllAuthorizer) Authorize(_ context.Context, _ *Principal, _, _ string) error {
	return nil
}

// DenyAllAuthorizer 拒绝所有请求的授权器.
//
// 用于测试或临时禁用访问.
type DenyAllAuthorizer struct{}

// NewDenyAllAuthorizer 创建拒绝所有请求的授权器.
func NewDenyAllAuthorizer() *DenyAllAuthorizer {
	return &DenyAllAuthorizer{}
}

// Authorize 实现 Authorizer 接口.
func (d *DenyAllAuthorizer) Authorize(_ context.Context, _ *Principal, _, _ string) error {
	return ErrForbidden
}

// FuncAuthorizer 函数式授权器.
//
// 将函数包装为授权器.
type FuncAuthorizer struct {
	fn func(ctx context.Context, principal *Principal, action, resource string) error
}

// NewFuncAuthorizer 创建函数式授权器.
func NewFuncAuthorizer(fn func(ctx context.Context, principal *Principal, action, resource string) error) *FuncAuthorizer {
	if fn == nil {
		panic("auth: 函数不能为空")
	}
	return &FuncAuthorizer{fn: fn}
}

// Authorize 实现 Authorizer 接口.
func (f *FuncAuthorizer) Authorize(ctx context.Context, principal *Principal, action, resource string) error {
	return f.fn(ctx, principal, action, resource)
}

// RoleAuthorizer 基于角色的简单授权器.
//
// 检查主体是否具有指定的任一角色.
type RoleAuthorizer struct {
	requiredRoles []string
	requireAll    bool
}

// RoleAuthorizerOption 角色授权器选项.
type RoleAuthorizerOption func(*RoleAuthorizer)

// WithRequireAllRoles 设置是否需要所有角色.
func WithRequireAllRoles(requireAll bool) RoleAuthorizerOption {
	return func(r *RoleAuthorizer) {
		r.requireAll = requireAll
	}
}

// NewRoleAuthorizer 创建角色授权器.
func NewRoleAuthorizer(roles []string, opts ...RoleAuthorizerOption) *RoleAuthorizer {
	r := &RoleAuthorizer{
		requiredRoles: roles,
		requireAll:    false,
	}
	for _, opt := range opts {
		opt(r)
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
//
// 检查主体是否具有指定的权限.
type PermissionAuthorizer struct {
	requiredPermissions []string
	requireAll          bool
}

// PermissionAuthorizerOption 权限授权器选项.
type PermissionAuthorizerOption func(*PermissionAuthorizer)

// WithRequireAllPermissions 设置是否需要所有权限.
func WithRequireAllPermissions(requireAll bool) PermissionAuthorizerOption {
	return func(p *PermissionAuthorizer) {
		p.requireAll = requireAll
	}
}

// NewPermissionAuthorizer 创建权限授权器.
func NewPermissionAuthorizer(permissions []string, opts ...PermissionAuthorizerOption) *PermissionAuthorizer {
	p := &PermissionAuthorizer{
		requiredPermissions: permissions,
		requireAll:          false,
	}
	for _, opt := range opts {
		opt(p)
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

// ChainAuthorizer 链式授权器.
//
// 所有授权器都通过才算通过.
type ChainAuthorizer struct {
	authorizers []Authorizer
}

// NewChainAuthorizer 创建链式授权器.
func NewChainAuthorizer(authorizers ...Authorizer) *ChainAuthorizer {
	return &ChainAuthorizer{
		authorizers: authorizers,
	}
}

// Authorize 实现 Authorizer 接口.
func (c *ChainAuthorizer) Authorize(ctx context.Context, principal *Principal, action, resource string) error {
	for _, auth := range c.authorizers {
		if err := auth.Authorize(ctx, principal, action, resource); err != nil {
			return err
		}
	}
	return nil
}
