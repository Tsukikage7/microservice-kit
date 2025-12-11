# Auth 认证授权包

提供统一的认证授权框架，支持多种认证方式和授权策略。

## 特性

- 可扩展的认证器接口，支持 JWT、API Key 等多种认证方式
- 可扩展的授权器接口，支持 RBAC 等授权策略
- 链式认证器，支持多种认证方式组合
- 带缓存的认证器，提升性能
- HTTP/gRPC/Endpoint 中间件
- Context 操作便捷函数

## 安装

```go
import "github.com/Tsukikage7/microservice-kit/auth"
```

## 核心概念

### Principal (身份主体)

表示已认证的用户或服务：

```go
type Principal struct {
    ID          string            // 唯一标识
    Type        string            // 类型: user, service, api_key
    Name        string            // 名称
    Roles       []string          // 角色列表
    Permissions []string          // 权限列表
    Metadata    map[string]any    // 扩展元数据
    ExpiresAt   *time.Time        // 过期时间
}
```

### Credentials (凭据)

认证凭据：

```go
type Credentials struct {
    Type  string            // bearer, api_key, basic
    Token string            // 凭据令牌
    Extra map[string]string // 额外信息
}
```

### Authenticator (认证器)

验证凭据并返回身份主体：

```go
type Authenticator interface {
    Authenticate(ctx context.Context, creds Credentials) (*Principal, error)
}
```

### Authorizer (授权器)

检查主体是否有权限执行操作：

```go
type Authorizer interface {
    Authorize(ctx context.Context, principal *Principal, action string, resource string) error
}
```

## 使用示例

### 1. 基础 JWT 认证

```go
import (
    "github.com/Tsukikage7/microservice-kit/auth"
    "github.com/Tsukikage7/microservice-kit/auth/jwt"
)

// 创建 JWT 服务并获取认证器
jwtService := jwt.NewJWT(
    jwt.WithSecretKey("your-secret-key"),
    jwt.WithLogger(log),
)
authenticator := jwt.NewAuthenticator(jwtService)

// 应用到 Endpoint
endpoint = auth.Middleware(authenticator)(endpoint)

// 应用到 HTTP
handler = auth.HTTPMiddleware(authenticator)(handler)

// 应用到 gRPC
srv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        auth.UnaryServerInterceptor(authenticator),
    ),
)
```

### 2. API Key 认证

```go
import (
    "github.com/Tsukikage7/microservice-kit/auth"
    "github.com/Tsukikage7/microservice-kit/auth/apikey"
)

// 创建 API Key 存储
store := apikey.NewMemoryStore()
store.Register("sk_live_xxx", &auth.Principal{
    ID:   "service-1",
    Type: auth.PrincipalTypeService,
    Roles: []string{"service"},
})

// 创建认证器
authenticator := apikey.NewAuthenticator(store)

// 使用哈希存储（更安全）
hashedKey := apikey.HashKey("sk_live_xxx")
store.Register(hashedKey, principal)
authenticator := apikey.NewAuthenticator(store, apikey.WithHashKeys(true))
```

### 3. 链式认证（JWT 或 API Key）

```go
chainAuth := auth.NewChainAuthenticator(
    jwt.NewAuthenticator(jwtService),
    apikey.NewAuthenticator(apiKeyStore),
)

handler = auth.HTTPMiddleware(chainAuth)(handler)
```

### 4. RBAC 授权

```go
import "github.com/Tsukikage7/microservice-kit/auth/rbac"

// 定义角色和权限
rbacAuth := rbac.New().
    AddRole(&rbac.Role{
        Name: "admin",
        Permissions: []rbac.Permission{
            {Action: "*", Resource: "*"},
        },
    }).
    AddRole(&rbac.Role{
        Name: "user",
        Permissions: []rbac.Permission{
            {Action: "read", Resource: "orders"},
            {Action: "create", Resource: "orders"},
        },
    }).
    AddRole(&rbac.Role{
        Name: "manager",
        Parents: []string{"user"}, // 继承 user 角色
        Permissions: []rbac.Permission{
            {Action: "update", Resource: "orders"},
            {Action: "delete", Resource: "orders"},
        },
    })

// 应用认证 + 授权
endpoint = auth.Middleware(authenticator,
    auth.WithAuthorizer(rbacAuth),
    auth.WithActionResource("read", "orders"),
)(endpoint)
```

### 5. 在业务逻辑中使用

```go
func CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
    // 获取当前用户
    principal, ok := auth.FromContext(ctx)
    if !ok {
        return auth.ErrUnauthenticated
    }

    // 检查权限
    if !auth.HasPermission(ctx, "orders:create") {
        return auth.ErrForbidden
    }

    // 检查角色
    if !auth.HasRole(ctx, "user") {
        return auth.ErrForbidden
    }

    // 使用用户 ID
    order.UserID = principal.ID
    order.CreatedBy = principal.Name

    return nil
}
```

### 6. 跳过某些路径

```go
// HTTP 跳过健康检查
handler = auth.HTTPMiddleware(authenticator,
    auth.WithSkipper(auth.HTTPSkipPaths("/health", "/ready", "/metrics")),
)(handler)

// gRPC 跳过某些方法
srv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        auth.UnaryServerInterceptor(authenticator,
            auth.WithSkipper(auth.GRPCSkipMethods(
                "/grpc.health.v1.Health/Check",
            )),
        ),
    ),
)
```

### 7. 自定义凭据提取

```go
// HTTP: 从自定义 header 提取
handler = auth.HTTPMiddleware(authenticator,
    auth.WithCredentialsExtractor(auth.APIKeyExtractor("X-Custom-Key")),
)(handler)

// gRPC: 从自定义 metadata 提取
auth.UnaryServerInterceptor(authenticator,
    auth.WithCredentialsExtractor(auth.GRPCAPIKeyExtractor("x-custom-key")),
)
```

## 内置认证器

| 认证器 | 说明 |
|-------|------|
| `jwt.Authenticator` | JWT 认证，整合 jwt 包 |
| `apikey.Authenticator` | API Key 认证 |
| `auth.ChainAuthenticator` | 链式认证，按顺序尝试 |
| `auth.CachingAuthenticator` | 带缓存的认证器 |
| `auth.MemoryAuthenticator` | 内存认证器，用于测试 |
| `auth.FuncAuthenticator` | 函数式认证器 |

## 内置授权器

| 授权器 | 说明 |
|-------|------|
| `rbac.RBAC` | 基于角色的访问控制 |
| `auth.RoleAuthorizer` | 简单角色检查 |
| `auth.PermissionAuthorizer` | 简单权限检查 |
| `auth.ChainAuthorizer` | 链式授权，全部通过 |
| `auth.AllowAllAuthorizer` | 允许所有（测试用） |
| `auth.DenyAllAuthorizer` | 拒绝所有（测试用） |
| `auth.FuncAuthorizer` | 函数式授权器 |

## 中间件选项

| 选项 | 说明 |
|-----|------|
| `WithAuthorizer` | 设置授权器 |
| `WithCredentialsExtractor` | 自定义凭据提取 |
| `WithSkipper` | 设置跳过条件 |
| `WithErrorHandler` | 自定义错误处理 |
| `WithLogger` | 设置日志记录器 |
| `WithActionResource` | 设置授权的操作和资源 |

## 错误类型

| 错误 | 说明 |
|-----|------|
| `ErrUnauthenticated` | 未认证 |
| `ErrForbidden` | 无权限 |
| `ErrInvalidCredentials` | 无效凭据 |
| `ErrCredentialsExpired` | 凭据已过期 |
| `ErrCredentialsNotFound` | 凭据未找到 |

## Context 便捷函数

```go
// 获取主体
principal, ok := auth.FromContext(ctx)
principal := auth.MustFromContext(ctx) // 不存在则 panic

// 检查角色/权限
auth.HasRole(ctx, "admin")
auth.HasAnyRole(ctx, "admin", "manager")
auth.HasAllRoles(ctx, "user", "verified")
auth.HasPermission(ctx, "orders:create")

// 获取 ID
id, ok := auth.GetPrincipalID(ctx)
```

## 最佳实践

1. **使用 RBAC 管理权限** - 比直接检查权限更易维护
2. **启用凭据哈希** - API Key 使用 `WithHashKeys(true)`
3. **缓存认证结果** - 使用 `CachingAuthenticator` 提升性能
4. **跳过公开端点** - 配置 Skipper 跳过不需要认证的路径
5. **记录日志** - 配置 Logger 便于调试和审计
