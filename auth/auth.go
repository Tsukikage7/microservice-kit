// Package auth 提供统一的认证授权框架.
//
// 特性:
//   - 可扩展的认证器接口，支持 JWT、API Key 等多种认证方式
//   - 可扩展的授权器接口，支持 RBAC 等授权策略
//   - 链式认证器，支持多种认证方式组合
//   - 带缓存的认证器，提升性能
//   - HTTP/gRPC/Endpoint 中间件
//
// 基本用法:
//
//	// 1. 创建认证器
//	jwtAuth := authjwt.NewAuthenticator(jwtService)
//
//	// 2. 创建授权器（可选）
//	rbacAuth := rbac.New().
//	    AddRole(&rbac.Role{Name: "admin", Permissions: []rbac.Permission{{Action: "*", Resource: "*"}}})
//
//	// 3. 应用中间件
//	endpoint = auth.Middleware(jwtAuth,
//	    auth.WithAuthorizer(rbacAuth),
//	)(endpoint)
//
// 在业务逻辑中使用:
//
//	func CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
//	    principal, ok := auth.FromContext(ctx)
//	    if !ok {
//	        return auth.ErrUnauthenticated
//	    }
//
//	    // 使用用户 ID
//	    order.UserID = principal.ID
//	    return nil
//	}
package auth

import (
	"context"
	"time"
)

// Credentials 认证凭据.
type Credentials struct {
	// Type 凭据类型: bearer, api_key, basic.
	Type string

	// Token 凭据令牌.
	Token string

	// Extra 额外信息.
	Extra map[string]string
}

// CredentialType 凭据类型常量.
const (
	CredentialTypeBearer = "bearer"
	CredentialTypeAPIKey = "api_key"
	CredentialTypeBasic  = "basic"
)

// Principal 身份主体，表示已认证的用户/服务.
type Principal struct {
	// ID 唯一标识.
	ID string

	// Type 主体类型: user, service, api_key.
	Type string

	// Name 主体名称（可选）.
	Name string

	// Roles 角色列表.
	Roles []string

	// Permissions 权限列表.
	Permissions []string

	// Metadata 扩展元数据.
	Metadata map[string]any

	// ExpiresAt 过期时间.
	ExpiresAt *time.Time
}

// PrincipalType 主体类型常量.
const (
	PrincipalTypeUser    = "user"
	PrincipalTypeService = "service"
	PrincipalTypeAPIKey  = "api_key"
)

// HasRole 检查主体是否具有指定角色.
func (p *Principal) HasRole(role string) bool {
	for _, r := range p.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission 检查主体是否具有指定权限.
func (p *Principal) HasPermission(permission string) bool {
	for _, perm := range p.Permissions {
		if perm == permission {
			return true
		}
	}
	return false
}

// HasAnyRole 检查主体是否具有任一指定角色.
func (p *Principal) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if p.HasRole(role) {
			return true
		}
	}
	return false
}

// HasAllRoles 检查主体是否具有所有指定角色.
func (p *Principal) HasAllRoles(roles ...string) bool {
	for _, role := range roles {
		if !p.HasRole(role) {
			return false
		}
	}
	return true
}

// IsExpired 检查主体是否已过期.
func (p *Principal) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*p.ExpiresAt)
}

// GetMetadata 获取元数据值.
func (p *Principal) GetMetadata(key string) (any, bool) {
	if p.Metadata == nil {
		return nil, false
	}
	v, ok := p.Metadata[key]
	return v, ok
}

// GetMetadataString 获取字符串类型的元数据值.
func (p *Principal) GetMetadataString(key string) string {
	v, ok := p.GetMetadata(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// Authenticator 认证器接口.
//
// 实现此接口以支持不同的认证方式（JWT、API Key 等）.
type Authenticator interface {
	// Authenticate 验证凭据，返回身份主体.
	//
	// 如果凭据无效，返回 ErrInvalidCredentials.
	// 如果凭据已过期，返回 ErrCredentialsExpired.
	Authenticate(ctx context.Context, creds Credentials) (*Principal, error)
}

// Authorizer 授权器接口.
//
// 实现此接口以支持不同的授权策略（RBAC、ABAC 等）.
type Authorizer interface {
	// Authorize 检查主体是否有权限执行操作.
	//
	// action: 操作类型，如 "read", "write", "delete".
	// resource: 资源标识，如 "orders", "users".
	//
	// 如果无权限，返回 ErrForbidden.
	Authorize(ctx context.Context, principal *Principal, action string, resource string) error
}

// CredentialsExtractor 凭据提取器函数.
//
// 用于从请求中提取认证凭据.
type CredentialsExtractor func(ctx context.Context, request any) (*Credentials, error)

// Skipper 跳过检查函数.
//
// 返回 true 表示跳过认证/授权检查.
type Skipper func(ctx context.Context, request any) bool

// ErrorHandler 错误处理函数.
//
// 用于自定义认证/授权错误的处理方式.
type ErrorHandler func(ctx context.Context, err error) error
