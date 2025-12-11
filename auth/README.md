# Auth

统一的认证授权框架，提供 HTTP/gRPC/Endpoint 中间件。

## 结构

```
auth/
├── auth.go        # 核心类型 (Principal, Credentials, 接口定义)
├── errors.go      # 错误定义
├── context.go     # Context 操作
├── authorizer.go  # Authorizer 实现 (RoleAuthorizer, PermissionAuthorizer)
├── options.go     # 中间件配置选项
├── middleware.go  # Endpoint 中间件
├── http.go        # HTTP 中间件
├── grpc.go        # gRPC 拦截器
└── jwt/           # JWT 认证（可独立使用）
```

## 快速开始

### 使用 JWT 认证

```go
import (
    "github.com/Tsukikage7/microservice-kit/auth"
    "github.com/Tsukikage7/microservice-kit/auth/jwt"
)

// 1. 创建 JWT 服务和认证器
jwtSrv := jwt.NewJWT(
    jwt.WithSecretKey("your-secret"),
    jwt.WithLogger(log),
)
authenticator := jwt.NewAuthenticator(jwtSrv)

// 2. 应用中间件
// HTTP
handler = auth.HTTPMiddleware(authenticator)(handler)

// gRPC
srv := grpc.NewServer(
    grpc.UnaryInterceptor(auth.UnaryServerInterceptor(authenticator)),
)

// Endpoint
endpoint = auth.Middleware(authenticator)(endpoint)
```

### 在业务逻辑中使用

```go
func CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
    // 获取当前用户
    principal, ok := auth.FromContext(ctx)
    if !ok {
        return auth.ErrUnauthenticated
    }

    // 使用用户信息
    order.UserID = principal.ID

    // 检查角色
    if !principal.HasRole("admin") {
        return auth.ErrForbidden
    }

    return nil
}
```

## 配置选项

```go
auth.HTTPMiddleware(authenticator,
    // 设置日志
    auth.WithLogger(log),

    // 设置授权器（角色检查）
    auth.WithAuthorizer(auth.NewRoleAuthorizer([]string{"admin"})),

    // 跳过某些路径
    auth.WithSkipper(auth.HTTPSkipPaths("/health", "/ready")),

    // 自定义凭据提取
    auth.WithCredentialsExtractor(auth.BearerExtractor),

    // 自定义错误处理
    auth.WithErrorHandler(func(ctx context.Context, err error) error {
        return customError(err)
    }),
)
```

## 授权器

### 角色授权

```go
// 需要任一角色
authorizer := auth.NewRoleAuthorizer([]string{"admin", "editor"})

// 需要所有角色
authorizer := auth.NewRoleAuthorizer([]string{"admin", "editor"}, true)
```

### 权限授权

```go
// 需要任一权限
authorizer := auth.NewPermissionAuthorizer([]string{"read:orders", "write:orders"})

// 需要所有权限
authorizer := auth.NewPermissionAuthorizer([]string{"read:orders", "write:orders"}, true)
```

### 便捷函数

```go
// 需要指定角色
endpoint = auth.RequireRoles(authenticator, []string{"admin"})(endpoint)

// 需要指定权限
endpoint = auth.RequirePermissions(authenticator, []string{"read:orders"})(endpoint)
```

## 上下文操作

```go
// 获取身份主体
principal, ok := auth.FromContext(ctx)

// 获取身份主体（不存在则 panic）
principal := auth.MustFromContext(ctx)

// 检查角色
if auth.HasRole(ctx, "admin") { ... }

// 检查权限
if auth.HasPermission(ctx, "read:orders") { ... }

// 获取用户 ID
id, ok := auth.GetPrincipalID(ctx)
```

## 错误处理

```go
var (
    auth.ErrUnauthenticated    // 未认证
    auth.ErrForbidden          // 无权限
    auth.ErrInvalidCredentials // 无效凭据
    auth.ErrCredentialsExpired // 凭据已过期
    auth.ErrCredentialsNotFound // 凭据未找到
)

// 错误检查
if auth.IsUnauthenticated(err) { ... }
if auth.IsForbidden(err) { ... }
```

## JWT 子包

JWT 子包可以独立使用，详见 [jwt/README.md](jwt/README.md)。

```go
// 直接使用 JWT（不通过 auth 包）
j := jwt.NewJWT(jwt.WithSecretKey("secret"), jwt.WithLogger(log))
handler = jwt.HTTPMiddleware(j)(handler)
```
