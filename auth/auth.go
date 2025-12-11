// Package auth 提供统一的认证授权框架.
//
// 特性:
//   - 可扩展的认证器接口
//   - 简单的角色/权限授权
//   - HTTP/gRPC/Endpoint 中间件
//   - 内置 JWT 支持（auth/jwt 子包）
//
// 基本用法:
//
//	// 使用 JWT 认证
//	jwtAuth := jwt.NewAuthenticator(jwtSrv)
//	endpoint = auth.Middleware(jwtAuth)(endpoint)
//
//	// HTTP 中间件
//	handler = auth.HTTPMiddleware(jwtAuth)(handler)
//
//	// gRPC 拦截器
//	srv := grpc.NewServer(
//	    grpc.UnaryInterceptor(auth.UnaryServerInterceptor(jwtAuth)),
//	)
//
// 在业务逻辑中使用:
//
//	func CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
//	    principal, ok := auth.FromContext(ctx)
//	    if !ok {
//	        return auth.ErrUnauthenticated
//	    }
//	    order.UserID = principal.ID
//	    return nil
//	}
package auth

import (
	"context"
	"time"
)

// ============================================================================
// 核心类型
// ============================================================================

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

	// Type 主体类型: user, service.
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

// ============================================================================
// 接口定义
// ============================================================================

// Authenticator 认证器接口.
type Authenticator interface {
	// Authenticate 验证凭据，返回身份主体.
	Authenticate(ctx context.Context, creds Credentials) (*Principal, error)
}

// Authorizer 授权器接口.
type Authorizer interface {
	// Authorize 检查主体是否有权限执行操作.
	Authorize(ctx context.Context, principal *Principal, action string, resource string) error
}

// CredentialsExtractor 凭据提取器函数.
type CredentialsExtractor func(ctx context.Context, request any) (*Credentials, error)

// Skipper 跳过检查函数.
type Skipper func(ctx context.Context, request any) bool

// ErrorHandler 错误处理函数.
type ErrorHandler func(ctx context.Context, err error) error
