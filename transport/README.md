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
        httpserver.WithAddr(":8080"),
        httpserver.WithLogger(log),
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
    httpserver.WithConfig(cfg.HTTP),
    httpserver.WithLogger(log),
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
    httpserver.WithAddr(":8080"),
    httpserver.WithLogger(log),
)

// 单独启动
if err := srv.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### HTTP 服务器配置选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithName` | `HTTP` | 服务器名称 |
| `WithAddr` | `:8080` | 监听地址 |
| `WithLogger` | - | 日志实例（必需） |
| `WithReadTimeout` | `30s` | 读取超时 |
| `WithWriteTimeout` | `30s` | 写入超时 |
| `WithIdleTimeout` | `120s` | 空闲超时 |
