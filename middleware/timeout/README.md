# Timeout

超时控制包，提供统一的请求超时管理，支持 Endpoint、HTTP 和 gRPC 三种级别。

## 功能特性

- **Endpoint 超时中间件** - 为任意 Endpoint 添加超时控制
- **HTTP 超时中间件** - HTTP 请求超时控制
- **gRPC 超时拦截器** - gRPC 服务端和客户端超时控制
- **级联超时** - 调用下游时自动减去已用时间
- **超时回调** - 支持自定义超时处理函数
- **降级支持** - 超时时返回降级响应

## 快速开始

### Endpoint 中间件

```go
import "github.com/Tsukikage7/microservice-kit/timeout"

// 基本用法
endpoint := myEndpoint
endpoint = timeout.EndpointMiddleware(5*time.Second)(endpoint)

// 带日志
endpoint = timeout.EndpointMiddleware(5*time.Second,
    timeout.WithLogger(log),
)(endpoint)

// 带降级
endpoint = timeout.EndpointMiddlewareWithFallback(
    5*time.Second,
    func(ctx context.Context, request any) (any, error) {
        return &DefaultResponse{}, nil
    },
)(endpoint)
```

### HTTP 中间件

```go
mux := http.NewServeMux()
mux.HandleFunc("/api", handler)

// 使用中间件
wrapped := timeout.HTTPMiddleware(10*time.Second)(mux)
http.ListenAndServe(":8080", wrapped)

// 或使用 TimeoutHandler（带自定义超时消息）
handler := timeout.HTTPTimeoutHandler(myHandler, 10*time.Second, "请求超时，请稍后重试")
```

### gRPC 拦截器

```go
// 服务端
srv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        timeout.UnaryServerInterceptor(5*time.Second),
    ),
    grpc.ChainStreamInterceptor(
        timeout.StreamServerInterceptor(30*time.Second),
    ),
)

// 客户端
conn, _ := grpc.Dial(target,
    grpc.WithUnaryInterceptor(timeout.UnaryClientInterceptor(5*time.Second)),
    grpc.WithStreamInterceptor(timeout.StreamClientInterceptor(30*time.Second)),
)
```

## 级联超时

级联超时用于在调用下游服务时自动减去已用时间，防止总超时超过预期。

```go
// 假设原始请求有 10s 超时，已经用了 3s
// Cascade 会使用 min(剩余7s, 指定的2s) = 2s 作为新超时

ctx, cancel := timeout.Cascade(ctx, 2*time.Second)
defer cancel()

resp, err := downstreamClient.Call(ctx, req)
```

### 预留处理时间

使用 `ShrinkBy` 预留处理超时响应的时间：

```go
// 预留 500ms 处理超时后的清理工作
ctx, cancel := timeout.ShrinkBy(ctx, 500*time.Millisecond)
defer cancel()

resp, err := slowOperation(ctx)
```

## 工具函数

### 查询剩余时间

```go
remaining, hasDeadline := timeout.Remaining(ctx)
if hasDeadline {
    log.Info("剩余时间", logger.Duration("remaining", remaining))
}
```

### 安全创建超时 Context

```go
ctx, cancel, err := timeout.WithTimeout(ctx, 5*time.Second)
if err != nil {
    // 处理无效超时参数
}
defer cancel()
```

## 配置选项

| 选项 | 说明 |
|------|------|
| `WithLogger(log)` | 设置日志记录器，超时时记录日志 |
| `WithOnTimeout(fn)` | 设置超时回调函数 |

## 超时回调

```go
endpoint = timeout.EndpointMiddleware(5*time.Second,
    timeout.WithLogger(log),
    timeout.WithOnTimeout(func(ctx any, duration time.Duration) {
        // ctx 类型取决于中间件类型：
        // - Endpoint: context.Context
        // - HTTP: *http.Request
        // - gRPC: context.Context
        metrics.TimeoutCounter.Inc()
    }),
)(endpoint)
```

## 最佳实践

### 1. 设置合理的超时时间

```go
// 根据 SLA 设置超时
// 一般建议：超时 = P99 延迟 * 2
httpSrv := httpserver.New(mux,
    httpserver.Logger(log),
    httpserver.Timeout(30*time.Second, 30*time.Second, 0),
)

// 为特定端点设置更短的超时
fastEndpoint = timeout.EndpointMiddleware(1*time.Second)(fastEndpoint)
slowEndpoint = timeout.EndpointMiddleware(30*time.Second)(slowEndpoint)
```

### 2. 级联超时避免累积

```go
func (s *Service) GetUserWithOrders(ctx context.Context, userID int64) (*Response, error) {
    // 获取用户信息，最多使用 2s
    userCtx, cancel := timeout.Cascade(ctx, 2*time.Second)
    defer cancel()
    user, err := s.userClient.Get(userCtx, userID)
    if err != nil {
        return nil, err
    }

    // 获取订单列表，使用剩余时间或最多 3s
    orderCtx, cancel := timeout.Cascade(ctx, 3*time.Second)
    defer cancel()
    orders, err := s.orderClient.List(orderCtx, userID)
    if err != nil {
        return nil, err
    }

    return &Response{User: user, Orders: orders}, nil
}
```

### 3. 结合熔断器使用

```go
// 超时 + 熔断 + 重试 的典型配置
endpoint := myEndpoint
endpoint = timeout.EndpointMiddleware(5*time.Second)(endpoint)
endpoint = circuitbreaker.EndpointMiddleware(cb)(endpoint)
endpoint = retry.EndpointMiddleware(3, 100*time.Millisecond)(endpoint)
```

### 4. 在 Handler 中检查超时

```go
func (s *Service) LongRunningTask(ctx context.Context, req *Request) (*Response, error) {
    for i := 0; i < 100; i++ {
        // 定期检查 context 是否已取消
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }

        // 执行一小步操作
        if err := s.processStep(ctx, i); err != nil {
            return nil, err
        }
    }
    return &Response{}, nil
}
```

## 错误处理

| 错误 | 说明 |
|------|------|
| `ErrTimeout` | 请求超时 |
| `ErrInvalidTimeout` | 超时时间无效（必须 > 0） |
| `context.DeadlineExceeded` | Context 超时 |
| `codes.DeadlineExceeded` | gRPC 超时错误码 |

## 与 transport 集成

```go
// HTTP 服务器
httpSrv := httpserver.New(mux,
    httpserver.Logger(log),
    httpserver.Timeout(30*time.Second, 30*time.Second, 0),  // read/write/idle 超时
)

// gRPC 服务器
grpcSrv := grpcserver.New(
    grpcserver.WithLogger(log),
    grpcserver.WithUnaryInterceptor(
        timeout.UnaryServerInterceptor(10*time.Second),
    ),
)
```
