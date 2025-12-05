# microservice-kit

Go 微服务开发工具包，提供构建生产级微服务所需的核心组件。

## 安装

```bash
go get github.com/Tsukikage7/microservice-kit
```

## 包概览

| 包 | 说明 | Endpoint | HTTP | gRPC |
|---|------|:--------:|:----:|:----:|
| [transport](./transport/) | 传输层抽象（Endpoint、Middleware） | ✅ 核心 | - | - |
| [metrics](./metrics/) | Prometheus 指标收集 | ✅ | ✅ | ✅ |
| [trace](./trace/) | OpenTelemetry 链路追踪 | ✅ | ✅ | ✅ |
| [ratelimit](./ratelimit/) | 限流（令牌桶、滑动窗口、分布式） | ✅ | ✅ | ✅ |
| [retry](./retry/) | 重试机制（指数退避） | ✅ | ✅ | ✅ |
| [jwt](./jwt/) | JWT 认证 | ✅ | ✅ | ✅ |
| [logger](./logger/) | 结构化日志（Zap） | - | - | - |
| [config](./config/) | 配置管理（多源、热更新） | - | - | - |
| [cache](./cache/) | 缓存（内存、Redis） | - | - | - |
| [discovery](./discovery/) | 服务发现（Consul、etcd） | - | - | - |
| [messaging](./messaging/) | 消息队列（RabbitMQ、Kafka） | - | - | - |
| [scheduler](./scheduler/) | 定时任务调度 | - | - | - |

## 中间件使用指南

### Endpoint 中间件

Endpoint 中间件用于 `transport.Endpoint` 层，适合服务内部的横切关注点处理。

```go
import (
    "github.com/Tsukikage7/microservice-kit/transport"
    "github.com/Tsukikage7/microservice-kit/metrics"
    "github.com/Tsukikage7/microservice-kit/trace"
    "github.com/Tsukikage7/microservice-kit/ratelimit"
    "github.com/Tsukikage7/microservice-kit/retry"
    "github.com/Tsukikage7/microservice-kit/jwt"
)

// 初始化 JWT 服务
j := jwt.New(
    jwt.WithSecretKey("your-secret-key"),
    jwt.WithLogger(log),
)

// 定义 Endpoint
var myEndpoint transport.Endpoint = func(ctx context.Context, req any) (any, error) {
    return process(req)
}

// 服务端中间件（从外到内执行）
myEndpoint = transport.Chain(
    metrics.EndpointMiddleware(collector, "my-service", "MyMethod"),
    trace.EndpointMiddleware("my-service", "MyMethod"),
    ratelimit.EndpointMiddleware(limiter),
    jwt.NewParser(j),  // JWT 验证
)(myEndpoint)

// 客户端中间件（从外到内执行）
clientEndpoint = transport.Chain(
    metrics.EndpointMiddleware(collector, "my-service", "MyMethod"),
    trace.EndpointMiddleware("my-service", "MyMethod"),
    jwt.NewSigner(j),  // JWT 签名
    retry.EndpointMiddleware(retryConfig),
)(clientEndpoint)
```

### HTTP 中间件

HTTP 中间件用于 `http.Handler` 层，适合 HTTP 服务器的请求处理。

```go
import (
    "github.com/Tsukikage7/microservice-kit/metrics"
    "github.com/Tsukikage7/microservice-kit/trace"
    "github.com/Tsukikage7/microservice-kit/ratelimit"
    "github.com/Tsukikage7/microservice-kit/jwt"
)

// 初始化 JWT 服务
j := jwt.New(
    jwt.WithSecretKey("your-secret-key"),
    jwt.WithLogger(log),
)

mux := http.NewServeMux()
mux.HandleFunc("/api/users", handleUsers)

// 应用中间件（从外到内执行）
var handler http.Handler = mux
handler = metrics.HTTPMiddleware(collector)(handler)
handler = trace.HTTPMiddleware("my-service")(handler)
handler = ratelimit.HTTPMiddleware(limiter)(handler)
handler = jwt.HTTPMiddleware(j)(handler)

http.ListenAndServe(":8080", handler)
```

### gRPC 拦截器

gRPC 拦截器用于 gRPC 服务端/客户端，支持一元和流式调用。

```go
import (
    "github.com/Tsukikage7/microservice-kit/metrics"
    "github.com/Tsukikage7/microservice-kit/trace"
    "github.com/Tsukikage7/microservice-kit/ratelimit"
    "github.com/Tsukikage7/microservice-kit/jwt"
    "github.com/Tsukikage7/microservice-kit/retry"
    "google.golang.org/grpc"
)

// 初始化 JWT 服务
j := jwt.New(
    jwt.WithSecretKey("your-secret-key"),
    jwt.WithLogger(log),
)

// 服务端
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        metrics.UnaryServerInterceptor(collector),
        trace.UnaryServerInterceptor("my-service"),
        ratelimit.UnaryServerInterceptor(limiter),
        jwt.UnaryServerInterceptor(j),
    ),
    grpc.ChainStreamInterceptor(
        metrics.StreamServerInterceptor(collector),
        trace.StreamServerInterceptor("my-service"),
        ratelimit.StreamServerInterceptor(limiter),
        jwt.StreamServerInterceptor(j),
    ),
)

// 客户端
conn, _ := grpc.Dial("localhost:50051",
    grpc.WithChainUnaryInterceptor(
        metrics.UnaryClientInterceptor(collector),
        trace.UnaryClientInterceptor("my-service"),
        retry.UnaryClientInterceptor(retryConfig),
    ),
    grpc.WithChainStreamInterceptor(
        metrics.StreamClientInterceptor(collector),
        trace.StreamClientInterceptor("my-service"),
        retry.StreamClientInterceptor(retryConfig),
    ),
)
```

## 快速开始

### 完整 HTTP 服务示例

```go
package main

import (
    "context"
    "net/http"

    "github.com/Tsukikage7/microservice-kit/config"
    "github.com/Tsukikage7/microservice-kit/jwt"
    "github.com/Tsukikage7/microservice-kit/logger"
    "github.com/Tsukikage7/microservice-kit/metrics"
    "github.com/Tsukikage7/microservice-kit/ratelimit"
    "github.com/Tsukikage7/microservice-kit/trace"
)

func main() {
    // 1. 加载配置
    cfg, _ := config.New(&config.Options{
        Paths: []string{"config.yaml"},
    })

    // 2. 初始化日志
    log := logger.MustNew(&logger.Config{Level: "info"})

    // 3. 初始化指标收集
    collector, _ := metrics.New(&metrics.Config{
        Namespace: "my_service",
        Path:      "/metrics",
    })

    // 4. 初始化链路追踪
    tp, _ := trace.NewTracer(&trace.TracingConfig{
        Enabled:      true,
        SamplingRate: 0.1,
        OTLP:         &trace.OTLPConfig{Endpoint: "localhost:4318"},
    }, "my-service", "1.0.0")
    defer tp.Shutdown(context.Background())

    // 5. 创建限流器
    limiter := ratelimit.NewTokenBucket(1000, 100)

    // 6. 初始化 JWT 认证
    whitelist := jwt.NewWhitelist().AddHTTPPaths("/health", "/metrics")
    j := jwt.New(
        jwt.WithSecretKey("your-secret-key"),
        jwt.WithLogger(log),
        jwt.WithWhitelist(whitelist),
    )

    // 7. 创建路由
    mux := http.NewServeMux()
    mux.HandleFunc("/health", healthHandler)
    mux.HandleFunc("/api/users", handleUsers)

    // 8. 应用中间件（从外到内）
    var handler http.Handler = mux
    handler = metrics.HTTPMiddleware(collector)(handler)
    handler = trace.HTTPMiddleware("my-service")(handler)
    handler = ratelimit.HTTPMiddleware(limiter)(handler)
    handler = jwt.HTTPMiddleware(j)(handler)

    // 9. 暴露指标端点
    http.Handle(collector.GetPath(), collector.GetHandler())

    log.Info("Server starting on :8080")
    http.ListenAndServe(":8080", handler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte(`{"status": "ok"}`))
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
    // 获取已验证的用户信息
    claims, ok := jwt.ClaimsFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    subject, _ := claims.GetSubject()
    w.Write([]byte(`{"user": "` + subject + `"}`))
}
```

### 完整 gRPC 服务示例

```go
package main

import (
    "context"
    "log"
    "net"

    "github.com/Tsukikage7/microservice-kit/jwt"
    "github.com/Tsukikage7/microservice-kit/logger"
    "github.com/Tsukikage7/microservice-kit/metrics"
    "github.com/Tsukikage7/microservice-kit/ratelimit"
    "github.com/Tsukikage7/microservice-kit/trace"
    "google.golang.org/grpc"
)

func main() {
    // 初始化日志
    log := logger.MustNew(&logger.Config{Level: "info"})

    // 初始化组件
    collector, _ := metrics.New(&metrics.Config{Namespace: "my_service"})
    tp, _ := trace.NewTracer(&trace.TracingConfig{
        Enabled: true,
        OTLP:    &trace.OTLPConfig{Endpoint: "localhost:4318"},
    }, "my-service", "1.0.0")
    defer tp.Shutdown(context.Background())
    limiter := ratelimit.NewTokenBucket(1000, 100)

    // 初始化 JWT 认证
    whitelist := jwt.NewWhitelist().AddGRPCMethods("/grpc.health.v1.Health/")
    j := jwt.New(
        jwt.WithSecretKey("your-secret-key"),
        jwt.WithLogger(log),
        jwt.WithWhitelist(whitelist),
    )

    // 创建 gRPC 服务器
    server := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            metrics.UnaryServerInterceptor(collector),
            trace.UnaryServerInterceptor("my-service"),
            ratelimit.UnaryServerInterceptor(limiter),
            jwt.UnaryServerInterceptor(j),
        ),
        grpc.ChainStreamInterceptor(
            metrics.StreamServerInterceptor(collector),
            trace.StreamServerInterceptor("my-service"),
            ratelimit.StreamServerInterceptor(limiter),
            jwt.StreamServerInterceptor(j),
        ),
    )

    // 注册服务...
    // pb.RegisterMyServiceServer(server, &myService{})

    lis, _ := net.Listen("tcp", ":50051")
    log.Info("gRPC server starting on :50051")
    server.Serve(lis)
}
```

## 中间件执行顺序

推荐的中间件执行顺序（从外到内）：

1. **Metrics** - 首先记录请求指标
2. **Tracing** - 创建追踪 span
3. **RateLimit** - 限流保护
4. **Auth/JWT** - 认证验证
5. **Retry** - 重试逻辑（通常用于客户端）
6. **Business Logic** - 业务处理

```go
// 服务端推荐顺序
handler = metrics.HTTPMiddleware(collector)(handler)
handler = trace.HTTPMiddleware("my-service")(handler)
handler = ratelimit.HTTPMiddleware(limiter)(handler)
handler = jwt.HTTPMiddleware(j)(handler)

// 客户端推荐顺序
conn, _ := grpc.Dial("target",
    grpc.WithChainUnaryInterceptor(
        metrics.UnaryClientInterceptor(collector),
        trace.UnaryClientInterceptor("my-service"),
        retry.UnaryClientInterceptor(retryConfig),
    ),
)
```

## 各包详细文档

- **[transport](./transport/)** - 传输层抽象，定义 Endpoint 和 Middleware
- **[metrics](./metrics/)** - Prometheus 指标收集
- **[trace](./trace/)** - OpenTelemetry 链路追踪
- **[ratelimit](./ratelimit/)** - 限流（令牌桶、滑动窗口、固定窗口、分布式）
- **[retry](./retry/)** - 重试机制（固定/指数/线性退避）
- **[jwt](./jwt/)** - JWT 认证
- **[logger](./logger/)** - 结构化日志
- **[config](./config/)** - 配置管理
- **[cache](./cache/)** - 缓存（内存、Redis）
- **[discovery](./discovery/)** - 服务发现（Consul、etcd）
- **[messaging](./messaging/)** - 消息队列（RabbitMQ、Kafka）
- **[scheduler](./scheduler/)** - 定时任务调度

## 设计原则

本工具包遵循以下设计原则：

- **KISS** - 保持简单，避免过度设计
- **DRY** - 抽象通用模式，减少重复代码
- **SOLID** - 单一职责，接口隔离
- **可组合** - 中间件可自由组合
- **可扩展** - 支持自定义实现

## License

MIT License
