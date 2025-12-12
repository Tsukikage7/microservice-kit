# Transport

传输层包，提供 HTTP 和 gRPC 的客户端与服务器实现。

## 目录结构

```
transport/
├── grpc/
│   ├── client/     # gRPC 客户端（服务发现）
│   └── server/     # gRPC 服务器（反射、Keepalive、拦截器链）
├── http/
│   ├── client/     # HTTP 客户端（服务发现）
│   └── server/     # HTTP 服务器（超时配置、优雅关闭）
├── gateway/
│   └── server/     # gRPC-Gateway 双协议服务器
├── health/         # 健康检查（HTTP + gRPC）
├── response/       # 统一响应格式
├── endpoint.go     # Endpoint 核心抽象
├── server.go       # 统一应用管理器（Application）
├── hooks.go        # 生命周期钩子
├── errors.go       # 共享错误定义
└── transport.go    # 包文档
```

## 快速开始

### 应用管理器

```go
package main

import (
    "github.com/Tsukikage7/microservice-kit/transport"
    grpcserver "github.com/Tsukikage7/microservice-kit/transport/grpc/server"
    httpserver "github.com/Tsukikage7/microservice-kit/transport/http/server"
)

func main() {
    log := logger.New()

    // 创建 HTTP 服务器
    httpSrv := httpserver.New(mux,
        httpserver.Addr(":8080"),
        httpserver.Logger(log),
    )

    // 创建 gRPC 服务器
    grpcSrv := grpcserver.New(
        grpcserver.WithAddr(":9090"),
        grpcserver.WithLogger(log),
    )
    grpcSrv.Register(userService)

    // 创建应用并运行
    app := transport.NewApplication(
        transport.WithName("my-service"),
        transport.WithLogger(log),
    )
    app.Use(httpSrv, grpcSrv)

    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### 生命周期钩子

```go
hooks := transport.NewHooks().
    BeforeStart(func(ctx context.Context) error {
        log.Info("服务启动前")
        return nil
    }).
    AfterStart(func(ctx context.Context) error {
        log.Info("服务启动后")
        return nil
    }).
    BeforeStop(func(ctx context.Context) error {
        log.Info("服务停止前")
        return nil
    }).
    AfterStop(func(ctx context.Context) error {
        log.Info("服务停止后")
        return nil
    }).
    Build()

app := transport.NewApplication(
    transport.WithName("my-service"),
    transport.WithLogger(log),
    transport.WithHooks(hooks),
)
```

### 从配置文件创建应用

```go
// 配置文件 (config.yaml)
// app:
//   name: "my-service"
//   version: "1.0.0"
//   graceful_timeout: "30s"
//   http:
//     addr: ":8080"
//     read_timeout: "30s"
//     write_timeout: "30s"
//     idle_timeout: "120s"
//   grpc:
//     addr: ":9090"
//     enable_reflection: true
//     keepalive_time: "60s"
//     keepalive_timeout: "20s"

var cfg transport.ApplicationConfig
viper.UnmarshalKey("app", &cfg)

app := transport.NewApplication(
    transport.WithConfig(cfg),
    transport.WithLogger(log),
)

// 使用配置创建服务器
httpSrv := httpserver.New(mux,
    httpserver.Logger(log),
    httpserver.Addr(cfg.HTTP.Addr),
)
grpcSrv := grpcserver.New(
    grpcserver.WithConfig(cfg.GRPC),
    grpcserver.WithLogger(log),
)
```

### ApplicationConfig 结构

```go
type ApplicationConfig struct {
    Name            string        // 应用名称
    Version         string        // 应用版本
    GracefulTimeout time.Duration // 优雅关闭超时
    HTTP            HTTPConfig    // HTTP 服务器配置
    GRPC            GRPCConfig    // gRPC 服务器配置
}

type HTTPConfig struct {
    Name         string        // 服务器名称
    Addr         string        // 监听地址
    ReadTimeout  time.Duration // 读取超时
    WriteTimeout time.Duration // 写入超时
    IdleTimeout  time.Duration // 空闲超时
}

type GRPCConfig struct {
    Name             string        // 服务器名称
    Addr             string        // 监听地址
    EnableReflection bool          // 是否启用反射
    KeepaliveTime    time.Duration // Keepalive 间隔
    KeepaliveTimeout time.Duration // Keepalive 超时
}
```

### Application 配置选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithName` | `app` | 应用名称 |
| `WithVersion` | `1.0.0` | 应用版本 |
| `WithLogger` | - | 日志实例（必需） |
| `WithHooks` | - | 生命周期钩子 |
| `WithGracefulTimeout` | `30s` | 优雅关闭超时 |
| `WithSignals` | `SIGINT, SIGTERM` | 监听的系统信号 |
| `WithConfig` | - | 从 ApplicationConfig 加载配置 |

## gRPC 客户端

```go
import grpcclient "github.com/Tsukikage7/microservice-kit/transport/grpc/client"

client, err := grpcclient.New(
    grpcclient.WithServiceName("user-service"),
    grpcclient.WithDiscovery(disc),
    grpcclient.WithLogger(log),
)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// 使用连接
userClient := userv1.NewUserServiceClient(client.Conn())
resp, err := userClient.GetUser(ctx, &userv1.GetUserRequest{Id: 123})
```

### gRPC 客户端配置选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithName` | `gRPC-Client` | 客户端名称（用于日志） |
| `WithServiceName` | - | 目标服务名称（必需） |
| `WithDiscovery` | - | 服务发现实例（必需） |
| `WithLogger` | - | 日志实例（必需） |
| `WithInterceptors` | - | 自定义拦截器 |
| `WithDialOptions` | - | 自定义 gRPC DialOption |

## gRPC 服务器

```go
import grpcserver "github.com/Tsukikage7/microservice-kit/transport/grpc/server"

srv := grpcserver.New(
    grpcserver.WithAddr(":9090"),
    grpcserver.WithLogger(log),
)
srv.Register(userService, orderService)

// 单独启动
if err := srv.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### gRPC 服务器配置选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithName` | `gRPC` | 服务器名称 |
| `WithAddr` | `:9090` | 监听地址 |
| `WithLogger` | - | 日志实例（必需） |
| `WithReflection` | `true` | 是否启用反射 |
| `WithKeepalive` | `60s, 20s` | Keepalive 参数 |
| `WithUnaryInterceptor` | - | 一元拦截器 |
| `WithStreamInterceptor` | - | 流拦截器 |
| `WithServerOption` | - | 自定义 gRPC 选项 |
| `WithTrace` | - | 启用链路追踪 |
| `WithRecovery` | - | 启用 panic 恢复 |

## HTTP 客户端

```go
import httpclient "github.com/Tsukikage7/microservice-kit/transport/http/client"

client, err := httpclient.New(
    httpclient.WithServiceName("api-gateway"),
    httpclient.WithDiscovery(disc),
    httpclient.WithLogger(log),
)
if err != nil {
    log.Fatal(err)
}

// 发送请求
resp, err := client.Get(ctx, "/api/users/123")
resp, err := client.Post(ctx, "/api/users", body)
```

### HTTP 客户端配置选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithName` | `HTTP-Client` | 客户端名称 |
| `WithServiceName` | - | 目标服务名称（必需） |
| `WithDiscovery` | - | 服务发现实例（必需） |
| `WithLogger` | - | 日志实例（必需） |
| `WithScheme` | `http` | URL scheme |
| `WithTimeout` | `30s` | 请求超时 |
| `WithHeader` | - | 添加默认请求头 |
| `WithTransport` | - | 自定义 Transport |

## HTTP 服务器

```go
import httpserver "github.com/Tsukikage7/microservice-kit/transport/http/server"

mux := http.NewServeMux()
mux.HandleFunc("/health", healthHandler)

srv := httpserver.New(mux,
    httpserver.Addr(":8080"),
    httpserver.Logger(log),
)

// 单独启动
if err := srv.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### HTTP 服务器配置选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `Name` | `HTTP` | 服务器名称 |
| `Addr` | `:8080` | 监听地址 |
| `Logger` | - | 日志实例（必需） |
| `Timeout` | `30s/30s/120s` | 超时设置(read/write/idle) |
| `Trace` | - | 启用链路追踪 |
| `Recovery` | - | 启用 panic 恢复 |
| `Feature` | - | 特性开关管理器 |
| `Auth` | - | 认证器 |
| `Profiling` | - | pprof 端点 |

## 可观测性

### 链路追踪

启用链路追踪后，所有请求会自动生成 traceId/spanId，并传播到下游服务：

```go
import "github.com/Tsukikage7/microservice-kit/tracing"

// 1. 初始化全局 TracerProvider（应用启动时）
tp, err := tracing.NewTracer("my-service",
    tracing.WithJaegerExporter("http://localhost:14268/api/traces"),
)
defer tp.Shutdown(ctx)

// 2. 启用服务器链路追踪
httpSrv := httpserver.New(mux,
    httpserver.Logger(log),
    httpserver.Trace("my-service"),  // 启用 HTTP 追踪
)

grpcSrv := grpcserver.New(
    grpcserver.WithLogger(log),
    grpcserver.WithTrace("my-service"),  // 启用 gRPC 追踪
)
```

### Panic 恢复

启用 panic 恢复后，handler 中的 panic 会被捕获并记录，避免服务崩溃：

```go
// HTTP 服务器
httpSrv := httpserver.New(mux,
    httpserver.Logger(log),
    httpserver.Recovery(),  // 启用 panic 恢复
)

// gRPC 服务器
grpcSrv := grpcserver.New(
    grpcserver.WithLogger(log),
    grpcserver.WithRecovery(),  // 启用 panic 恢复
)
```

**Panic 日志输出示例：**

```json
{
  "level": "ERROR",
  "timestamp": "2025-12-11 15:30:46",
  "msg": "http panic recovered",
  "traceId": "abc123def456",
  "spanId": "xyz789",
  "panic": "runtime error: index out of range [5] with length 3",
  "method": "GET",
  "path": "/api/users/123",
  "stack": "goroutine 42 [running]:..."
}
```

### 推荐配置

生产环境建议同时启用链路追踪和 panic 恢复：

```go
httpSrv := httpserver.New(mux,
    httpserver.Logger(log),
    httpserver.Trace("my-service"),
    httpserver.Recovery(),
)

grpcSrv := grpcserver.New(
    grpcserver.WithLogger(log),
    grpcserver.WithTrace("my-service"),
    grpcserver.WithRecovery(),
)
```

**中间件执行顺序（从外到内）：**

```
Recovery → Trace → Health → 业务逻辑
```

Recovery 在最外层，确保能捕获所有内层 panic（包括 Trace 中间件）。

## Gateway 服务器

gRPC-Gateway 双协议服务器，同时提供 gRPC 和 HTTP/JSON API：

```go
import gateway "github.com/Tsukikage7/microservice-kit/transport/gateway/server"

srv := gateway.New(
    gateway.WithGRPCAddr(":9090"),
    gateway.WithHTTPAddr(":8080"),
    gateway.WithLogger(log),
    gateway.WithTrace("my-service"),
    gateway.WithRecovery(),
    gateway.WithResponse(),  // 统一响应格式
)
srv.Register(userService, orderService)

if err := srv.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### Gateway 配置选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithName` | `Gateway` | 服务器名称 |
| `WithGRPCAddr` | `:9090` | gRPC 监听地址 |
| `WithHTTPAddr` | `:8080` | HTTP 监听地址 |
| `WithLogger` | - | 日志实例（必需） |
| `WithReflection` | `true` | 是否启用 gRPC 反射 |
| `WithKeepalive` | `60s, 20s` | gRPC Keepalive 参数 |
| `WithUnaryInterceptor` | - | gRPC 一元拦截器 |
| `WithStreamInterceptor` | - | gRPC 流拦截器 |
| `WithHTTPTimeout` | `30s, 30s, 120s` | HTTP 超时（读/写/空闲） |
| `WithTrace` | - | 启用链路追踪（gRPC + HTTP） |
| `WithRecovery` | - | 启用 panic 恢复（gRPC + HTTP） |
| `WithResponse` | - | 启用统一响应格式 |
