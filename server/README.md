# Server

应用服务器框架，提供统一的服务器生命周期管理。

## 特性

- 统一管理多个服务器（HTTP、gRPC 等）
- 生命周期钩子（BeforeStart/AfterStart/BeforeStop/AfterStop）
- 优雅关闭
- 信号处理
- Functional Options 模式

## 快速开始

```go
package main

import (
    "context"
    "net/http"

    "github.com/Tsukikage7/microservice-kit/logger"
    "github.com/Tsukikage7/microservice-kit/server"
)

func main() {
    log := logger.New()

    // 创建 HTTP 服务器
    mux := http.NewServeMux()
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("ok"))
    })

    httpSrv := server.NewHTTP(mux,
        server.WithHTTPAddr(":8080"),
        server.WithHTTPLogger(log),
    )

    // 创建 gRPC 服务器
    grpcSrv := server.NewGRPC(
        server.WithGRPCAddr(":9090"),
        server.WithGRPCLogger(log),
    )
    grpcSrv.Register(userService)

    // 创建应用并运行
    app := server.NewApp(
        server.WithName("my-service"),
        server.WithVersion("1.0.0"),
        server.WithLogger(log),
    )
    app.Use(httpSrv, grpcSrv)

    if err := app.Run(); err != nil {
        panic(err)
    }
}
```

## 架构

```
┌─────────────────────────────────────────────────────────┐
│                         App                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │
│  │    HTTP     │  │    gRPC     │  │   Custom    │      │
│  │   Server    │  │   Server    │  │   Server    │      │
│  └─────────────┘  └─────────────┘  └─────────────┘      │
│                                                          │
│  Lifecycle: BeforeStart → Start → AfterStart            │
│             BeforeStop  → Stop  → AfterStop             │
└─────────────────────────────────────────────────────────┘
```

## 核心接口

### Server 接口

所有服务器都需要实现此接口：

```go
type Server interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Name() string
    Addr() string
}
```

### App 应用

管理多个服务器的生命周期：

```go
app := server.NewApp(
    server.WithName("my-service"),
    server.WithVersion("1.0.0"),
    server.WithLogger(log),  // 必需
    server.WithGracefulTimeout(30 * time.Second),
    server.WithHooks(hooks),
)

app.Use(httpServer, grpcServer)
app.Run()
```

**注意**: `WithLogger` 是必需的，未设置会 panic。

## HTTP 服务器

```go
srv := server.NewHTTP(handler,
    server.WithHTTPName("api-gateway"),  // 可选，默认 "HTTP"
    server.WithHTTPAddr(":8080"),
    server.WithHTTPReadTimeout(30 * time.Second),
    server.WithHTTPWriteTimeout(30 * time.Second),
    server.WithHTTPIdleTimeout(120 * time.Second),
    server.WithHTTPLogger(log),  // 必需
)
```

### 配置选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithHTTPName` | `HTTP` | 服务器名称 |
| `WithHTTPAddr` | `:8080` | 监听地址 |
| `WithHTTPReadTimeout` | `30s` | 读取超时 |
| `WithHTTPWriteTimeout` | `30s` | 写入超时 |
| `WithHTTPIdleTimeout` | `120s` | 空闲超时 |
| `WithHTTPLogger` | - | 日志记录器（必需） |

## gRPC 服务器

```go
srv := server.NewGRPC(
    server.WithGRPCName("user-grpc"),  // 可选，默认 "gRPC"
    server.WithGRPCAddr(":9090"),
    server.WithGRPCReflection(true),
    server.WithGRPCKeepalive(60*time.Second, 20*time.Second),
    server.WithGRPCUnaryInterceptor(loggingInterceptor),
    server.WithGRPCLogger(log),  // 必需
)
srv.Register(userService, orderService)
```

### 配置选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithGRPCName` | `gRPC` | 服务器名称 |
| `WithGRPCAddr` | `:9090` | 监听地址 |
| `WithGRPCReflection` | `true` | 启用反射 |
| `WithGRPCKeepalive` | `60s, 20s` | Keepalive 参数 |
| `WithGRPCUnaryInterceptor` | - | 一元拦截器 |
| `WithGRPCStreamInterceptor` | - | 流拦截器 |
| `WithGRPCServerOption` | - | 自定义选项 |
| `WithGRPCLogger` | - | 日志记录器（必需） |

### 服务注册

实现 `GRPCRegistrar` 接口：

```go
type UserService struct {}

func (s *UserService) RegisterGRPC(server *grpc.Server) {
    pb.RegisterUserServiceServer(server, s)
}
```

## 生命周期钩子

```go
hooks := server.NewHooks().
    BeforeStart(func(ctx context.Context) error {
        log.Info("初始化数据库连接...")
        return nil
    }).
    AfterStart(func(ctx context.Context) error {
        log.Info("服务已启动")
        return nil
    }).
    BeforeStop(func(ctx context.Context) error {
        log.Info("开始关闭...")
        return nil
    }).
    AfterStop(func(ctx context.Context) error {
        log.Info("服务已关闭")
        return nil
    }).
    Build()

app := server.NewApp(
    server.WithLogger(log),
    server.WithHooks(hooks),
)
```

### 钩子执行顺序

```
启动流程：
1. BeforeStart hooks (按添加顺序)
2. 启动所有服务器 (并行)
3. AfterStart hooks (按添加顺序)

关闭流程：
1. 收到信号/调用 Stop
2. BeforeStop hooks (按添加顺序)
3. 停止所有服务器 (并行，带超时)
4. AfterStop hooks (按添加顺序)
```

## 自定义服务器

实现 `Server` 接口即可：

```go
type CustomServer struct {
    name string
    addr string
}

func (s *CustomServer) Start(ctx context.Context) error {
    // 启动逻辑
    return nil
}

func (s *CustomServer) Stop(ctx context.Context) error {
    // 停止逻辑
    return nil
}

func (s *CustomServer) Name() string {
    return s.name
}

func (s *CustomServer) Addr() string {
    return s.addr
}

// 使用
app.Use(&CustomServer{name: "custom", addr: ":8888"})
```

## 优雅关闭

```go
app := server.NewApp(
    server.WithLogger(log),
    server.WithGracefulTimeout(30 * time.Second),
    server.WithSignals(syscall.SIGINT, syscall.SIGTERM),
)
```

- 收到信号后，开始优雅关闭
- 在超时时间内等待所有服务器停止
- 超时后强制退出

## 完整示例

```go
package main

import (
    "context"
    "net/http"
    "time"

    "github.com/Tsukikage7/microservice-kit/logger"
    "github.com/Tsukikage7/microservice-kit/server"
)

func main() {
    log := logger.New()

    // HTTP Handler
    mux := http.NewServeMux()
    mux.HandleFunc("/health", healthHandler)
    mux.HandleFunc("/ready", readyHandler)

    // 创建 HTTP 服务器
    httpSrv := server.NewHTTP(mux,
        server.WithHTTPName("api"),
        server.WithHTTPAddr(":8080"),
        server.WithHTTPReadTimeout(30*time.Second),
        server.WithHTTPLogger(log),
    )

    // 创建 gRPC 服务器
    grpcSrv := server.NewGRPC(
        server.WithGRPCName("rpc"),
        server.WithGRPCAddr(":9090"),
        server.WithGRPCReflection(true),
        server.WithGRPCLogger(log),
    )
    grpcSrv.Register(&UserService{}, &OrderService{})

    // 生命周期钩子
    hooks := server.NewHooks().
        BeforeStart(func(ctx context.Context) error {
            log.Info("连接数据库...")
            return initDB()
        }).
        AfterStop(func(ctx context.Context) error {
            log.Info("关闭数据库连接...")
            return closeDB()
        }).
        Build()

    // 创建并运行应用
    app := server.NewApp(
        server.WithName("example-service"),
        server.WithVersion("1.0.0"),
        server.WithLogger(log),
        server.WithGracefulTimeout(30*time.Second),
        server.WithHooks(hooks),
    )

    app.Use(httpSrv, grpcSrv)

    if err := app.Run(); err != nil {
        log.Error("应用运行失败", "error", err)
    }
}
```

## 错误处理

```go
var (
    ErrServerClosed  = errors.New("server: server is closed")
    ErrServerRunning = errors.New("server: server is already running")
    ErrNoServers     = errors.New("server: no servers registered")
    ErrAddrEmpty     = errors.New("server: address is empty")
    ErrNilHandler    = errors.New("server: handler is nil")
)
```

**注意**: 如果未设置 `logger`，`NewApp`、`NewHTTP`、`NewGRPC` 会 panic。
