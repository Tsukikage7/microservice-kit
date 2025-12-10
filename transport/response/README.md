# Response 统一响应体包

提供微服务 API 的统一响应格式，支持 HTTP 和 gRPC 协议，内置错误码体系和分页支持。

## 特性

- ✅ 泛型响应体 `Response[T]`
- ✅ 数字错误码体系
- ✅ HTTP/gRPC 状态码自动映射
- ✅ 内置分页响应 `PagedResponse[T]`
- ✅ 与 `gateway` 包无缝集成
- ✅ 内部错误信息自动隐藏

## 响应格式

### 标准响应

```json
{
  "code": 0,
  "message": "成功",
  "data": { ... }
}
```

### 分页响应

```json
{
  "code": 0,
  "message": "成功",
  "data": [...],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

### 错误响应

```json
{
  "code": 40001,
  "message": "用户不存在"
}
```

## 快速开始

### 与 Gateway 集成（推荐）

```go
import (
    gateway "github.com/Tsukikage7/microservice-kit/transport/gateway/server"
)

server := gateway.New(
    gateway.WithName("user-service"),
    gateway.WithLogger(log),
    gateway.WithTrace("user-service"),   // 启用链路追踪
    gateway.WithResponse(),              // 启用统一响应格式
)
```

启用 `WithResponse()` 后：
- **gRPC 错误** → 自动映射到正确的状态码
- **HTTP 错误** → 自动转换为 `{"code": xxx, "message": "xxx"}` 格式
- **内部错误** → 详细信息自动隐藏

### 独立 HTTP Server

```go
import (
    httpserver "github.com/Tsukikage7/microservice-kit/transport/http/server"
)

handler := httpserver.NewEndpointHandler(
    getUserEndpoint,
    decodeGetUserRequest,
    httpserver.EncodeJSONResponse,
    httpserver.WithResponse(),  // 启用统一响应格式
)
```

### 独立 gRPC Server

```go
import (
    grpcserver "github.com/Tsukikage7/microservice-kit/transport/grpc/server"
)

handler := grpcserver.NewEndpointHandler(
    getUserEndpoint,
    decodeGetUserRequest,
    encodeGetUserResponse,
    grpcserver.WithResponse(),  // 启用统一响应格式
)
```

### 错误码规范

| 范围 | 类型 | 示例 | 返回详情 |
|------|------|------|----------|
| 0 | 成功 | `CodeSuccess` | ✅ |
| 1xxxx | 通用错误 | `CodeUnknown`, `CodeTimeout` | ✅ |
| 2xxxx | 认证/授权 | `CodeUnauthorized`, `CodeForbidden` | ✅ |
| 3xxxx | 参数错误 | `CodeInvalidParam`, `CodeMissingParam` | ✅ |
| 4xxxx | 资源错误 | `CodeNotFound`, `CodeAlreadyExists` | ✅ |
| 5xxxx | 内部错误 | `CodeInternal`, `CodeDatabaseError` | ❌ 隐藏 |
| 6xxxx | 外部服务 | `CodeServiceUnavailable` | ❌ 隐藏 |

### Service 层使用

```go
import "github.com/Tsukikage7/microservice-kit/transport/response"

func (s *UserService) GetByID(ctx context.Context, id int) (*User, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            // 业务错误：返回给客户端
            return nil, response.NewErrorWithMessage(response.CodeNotFound, "用户不存在")
        }
        // 内部错误：详情会被隐藏
        return nil, response.Wrap(response.CodeDatabaseError, err)
    }
    return user, nil
}
```

### 自定义错误码

```go
var CodeUserDisabled = response.NewCode(
    40101,                      // 数字码
    "用户已被禁用",              // 默认消息
    http.StatusForbidden,       // HTTP 状态码
    codes.PermissionDenied,     // gRPC 状态码
)
```

### 直接构建响应

```go
// 成功响应
resp := response.OK(user)

// 带消息的成功响应
resp := response.OKWithMessage(user, "创建成功")

// 失败响应
resp := response.Fail[User](response.CodeNotFound)

// 带自定义消息的失败响应
resp := response.FailWithMessage[User](response.CodeNotFound, "用户不存在")

// 从 error 创建失败响应
resp := response.FailWithError[User](err)

// 分页响应
result := pagination.NewResult(users, total, pag)
resp := response.Paged(result)
```

### HTTP Handler 直接写入

```go
func GetUser(w http.ResponseWriter, r *http.Request) {
    user, err := userService.GetByID(ctx, id)
    if err != nil {
        response.WriteError(w, err)
        return
    }
    response.WriteSuccess(w, user)
}
```

## 状态码映射

| 业务错误码 | HTTP 状态码 | gRPC 状态码 |
|-----------|-------------|-------------|
| `CodeSuccess` | 200 | OK |
| `CodeInvalidParam` | 400 | InvalidArgument |
| `CodeUnauthorized` | 401 | Unauthenticated |
| `CodeForbidden` | 403 | PermissionDenied |
| `CodeNotFound` | 404 | NotFound |
| `CodeAlreadyExists` | 409 | AlreadyExists |
| `CodeResourceExhausted` | 429 | ResourceExhausted |
| `CodeInternal` | 500 | Internal |
| `CodeServiceUnavailable` | 503 | Unavailable |

## API 参考

### 响应构建

| 函数 | 说明 |
|------|------|
| `OK[T](data)` | 成功响应 |
| `OKWithMessage[T](data, msg)` | 带消息的成功响应 |
| `Fail[T](code)` | 失败响应 |
| `FailWithMessage[T](code, msg)` | 带消息的失败响应 |
| `FailWithError[T](err)` | 从 error 创建失败响应 |
| `Paged[T](result)` | 分页响应 |
| `PagedFail[T](code)` | 分页失败响应 |

### 错误处理

| 函数 | 说明 |
|------|------|
| `NewError(code)` | 创建业务错误 |
| `NewErrorWithMessage(code, msg)` | 带消息的业务错误 |
| `Wrap(code, err)` | 包装错误 |
| `ExtractCode(err)` | 提取错误码 |
| `ExtractMessage(err)` | 提取错误消息（内部错误隐藏详情） |
| `ExtractMessageUnsafe(err)` | 提取完整错误消息（仅用于日志） |

### gRPC 集成

| 函数 | 说明 |
|------|------|
| `GRPCError(err)` | 转换为 gRPC error |
| `UnaryServerInterceptor()` | gRPC 一元拦截器 |
| `StreamServerInterceptor()` | gRPC 流拦截器 |
| `FromGRPCError(err)` | 从 gRPC error 提取 Code |

### HTTP 集成

| 函数 | 说明 |
|------|------|
| `WriteSuccess[T](w, data)` | 写入成功响应 |
| `WriteFail(w, code)` | 写入失败响应 |
| `WriteError(w, err)` | 写入错误响应 |
| `WritePaged[T](w, resp)` | 写入分页响应 |
