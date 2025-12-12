# microservice-kit

Go 微服务开发工具包，提供构建生产级微服务所需的核心组件。

## 安装

```bash
go get github.com/Tsukikage7/microservice-kit
```

## 包概览

### 可观测性 (observability/)

| 包 | 说明 | Endpoint | HTTP | gRPC |
|---|------|:--------:|:----:|:----:|
| [observability/metrics](./observability/metrics/) | Prometheus 指标收集 | ✅ | ✅ | ✅ |
| [observability/tracing](./observability/tracing/) | OpenTelemetry 链路追踪 | ✅ | ✅ | ✅ |

### 中间件 (middleware/)

| 包 | 说明 | Endpoint | HTTP | gRPC |
|---|------|:--------:|:----:|:----:|
| [middleware/ratelimit](./middleware/ratelimit/) | 限流（令牌桶、滑动窗口、分布式） | ✅ | ✅ | ✅ |
| [middleware/retry](./middleware/retry/) | 重试机制（指数退避） | ✅ | ✅ | ✅ |
| [middleware/recovery](./middleware/recovery/) | Panic 恢复 | ✅ | ✅ | ✅ |
| [middleware/timeout](./middleware/timeout/) | 超时控制 | ✅ | ✅ | ✅ |
| [middleware/idempotency](./middleware/idempotency/) | 幂等性保证 | ✅ | ✅ | - |
| [middleware/semaphore](./middleware/semaphore/) | 并发控制 | ✅ | - | - |

### 请求上下文 (request/)

| 包 | 说明 | HTTP | gRPC |
|---|------|:----:|:----:|
| [request](./request/) | 组合中间件（统一入口） | ✅ | ✅ |
| [request/clientip](./request/clientip/) | 客户端 IP 提取、地理位置、ACL | ✅ | ✅ |
| [request/useragent](./request/useragent/) | User-Agent 解析 | ✅ | ✅ |
| [request/deviceinfo](./request/deviceinfo/) | 设备信息（Client Hints 优先） | ✅ | ✅ |
| [request/botdetect](./request/botdetect/) | 机器人检测 | ✅ | ✅ |
| [request/locale](./request/locale/) | 语言区域设置 | ✅ | ✅ |
| [request/referer](./request/referer/) | 来源页面解析、UTM 参数 | ✅ | ✅ |
| [request/activity](./request/activity/) | 用户活动追踪（Redis + Kafka） | ✅ | ✅ |

### 存储 (storage/)

| 包 | 说明 | 工厂函数 |
|---|------|----------|
| [storage/cache](./storage/cache/) | 缓存（内存、Redis） | `NewCache` / `MustNewCache` |
| [storage/database](./storage/database/) | 数据库（GORM） | `NewDatabase` / `MustNewDatabase` |
| [storage/lock](./storage/lock/) | 分布式锁 | `NewLock` |

### 工具 (util/)

| 包 | 说明 |
|---|------|
| [util/pagination](./util/pagination/) | 分页工具 |
| [util/sorting](./util/sorting/) | 排序工具 |
| [util/collections](./util/collections/) | 集合工具（TreeMap、TreeSet、LinkedList） |
| [util/pbjson](./util/pbjson/) | Protobuf JSON 序列化（零值字段输出） |

### 核心组件

| 包 | 说明 | 工厂函数 |
|---|------|----------|
| [transport](./transport/) | 传输层抽象（Endpoint、Middleware） | - |
| [auth](./auth/) | 认证授权（JWT、API Key、RBAC） | - |
| [logger](./logger/) | 结构化日志（Zap） | `NewLogger` / `MustNewLogger` |
| [config](./config/) | 配置管理（多源、热更新） | `New` |
| [discovery](./discovery/) | 服务发现（Consul、etcd） | `NewDiscovery` / `MustNewDiscovery` |
| [messaging](./messaging/) | 消息队列（Kafka、RabbitMQ） | `NewProducer` / `NewConsumer` |
| [scheduler](./scheduler/) | 定时任务调度 | `NewScheduler` / `MustNewScheduler` |

### 分布式模式

| 包 | 说明 |
|---|------|
| [domain](./domain/) | 领域驱动设计（聚合根、领域事件） |
| [cqrs](./cqrs/) | 命令查询职责分离 |
| [saga](./saga/) | Saga 分布式事务 |

## 快速开始

### 基础组件初始化

```go
package main

import (
    "context"
    "time"

    "github.com/Tsukikage7/microservice-kit/config"
    "github.com/Tsukikage7/microservice-kit/logger"
    "github.com/Tsukikage7/microservice-kit/observability/metrics"
    "github.com/Tsukikage7/microservice-kit/storage/cache"
    "github.com/Tsukikage7/microservice-kit/storage/database"
)

func main() {
    // 1. 加载配置
    cfg, _ := config.New(&config.Options{
        Paths: []string{"config.yaml"},
    })

    // 2. 初始化日志
    log := logger.MustNewLogger(&logger.Config{Level: "info"})
    defer log.Close()

    // 3. 初始化指标收集
    collector := metrics.MustNewMetrics(&metrics.Config{
        Namespace: "my_service",
        Path:      "/metrics",
    })

    // 4. 初始化缓存
    memCache := cache.MustNewCache(cache.NewMemoryConfig(), log)
    defer memCache.Close()

    // 5. 初始化数据库
    db := database.MustNewDatabase(&database.Config{
        Driver: database.DriverMySQL,
        DSN:    "user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True",
    }, log)
    defer db.Close()

    // 使用组件...
}
```

### HTTP 服务示例

```go
package main

import (
    "context"
    "net/http"

    "github.com/Tsukikage7/microservice-kit/auth/jwt"
    "github.com/Tsukikage7/microservice-kit/logger"
    "github.com/Tsukikage7/microservice-kit/middleware/ratelimit"
    "github.com/Tsukikage7/microservice-kit/observability/metrics"
    "github.com/Tsukikage7/microservice-kit/observability/tracing"
)

func main() {
    // 初始化日志
    log := logger.MustNewLogger(logger.DefaultConfig())
    defer log.Close()

    // 初始化指标收集
    collector := metrics.MustNewMetrics(&metrics.Config{
        Namespace: "my_service",
        Path:      "/metrics",
    })

    // 初始化链路追踪
    tp, _ := tracing.NewTracer(&tracing.TracingConfig{
        Enabled:      true,
        SamplingRate: 0.1,
        OTLP:         &tracing.OTLPConfig{Endpoint: "localhost:4318"},
    }, "my-service", "1.0.0")
    defer tp.Shutdown(context.Background())

    // 创建限流器
    limiter := ratelimit.NewTokenBucket(1000, 100)

    // 初始化 JWT 认证
    whitelist := jwt.NewWhitelist().AddHTTPPaths("/health", "/metrics")
    j := jwt.NewJWT(
        jwt.WithSecretKey("your-secret-key"),
        jwt.WithLogger(log),
        jwt.WithWhitelist(whitelist),
    )

    // 创建路由
    mux := http.NewServeMux()
    mux.HandleFunc("/health", healthHandler)
    mux.HandleFunc("/api/users", usersHandler)

    // 应用中间件（从外到内）
    var handler http.Handler = mux
    handler = metrics.HTTPMiddleware(collector)(handler)
    handler = tracing.HTTPMiddleware("my-service")(handler)
    handler = ratelimit.HTTPMiddleware(limiter)(handler)
    handler = jwt.HTTPMiddleware(j)(handler)

    // 暴露指标端点
    http.Handle(collector.GetPath(), collector.GetHandler())

    log.Info("Server starting on :8080")
    http.ListenAndServe(":8080", handler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte(`{"status": "ok"}`))
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
    claims, ok := jwt.ClaimsFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    subject, _ := claims.GetSubject()
    w.Write([]byte(`{"user": "` + subject + `"}`))
}
```

### gRPC 服务示例

```go
package main

import (
    "context"
    "net"

    "github.com/Tsukikage7/microservice-kit/auth/jwt"
    "github.com/Tsukikage7/microservice-kit/logger"
    "github.com/Tsukikage7/microservice-kit/middleware/ratelimit"
    "github.com/Tsukikage7/microservice-kit/observability/metrics"
    "github.com/Tsukikage7/microservice-kit/observability/tracing"
    "google.golang.org/grpc"
)

func main() {
    // 初始化日志
    log := logger.MustNewLogger(logger.DefaultConfig())
    defer log.Close()

    // 初始化组件
    collector := metrics.MustNewMetrics(&metrics.Config{Namespace: "my_service"})
    tp, _ := tracing.NewTracer(&tracing.TracingConfig{
        Enabled: true,
        OTLP:    &tracing.OTLPConfig{Endpoint: "localhost:4318"},
    }, "my-service", "1.0.0")
    defer tp.Shutdown(context.Background())
    limiter := ratelimit.NewTokenBucket(1000, 100)

    // 初始化 JWT 认证
    whitelist := jwt.NewWhitelist().AddGRPCMethods("/grpc.health.v1.Health/")
    j := jwt.NewJWT(
        jwt.WithSecretKey("your-secret-key"),
        jwt.WithLogger(log),
        jwt.WithWhitelist(whitelist),
    )

    // 创建 gRPC 服务器
    server := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            metrics.UnaryServerInterceptor(collector),
            tracing.UnaryServerInterceptor("my-service"),
            ratelimit.UnaryServerInterceptor(limiter),
            jwt.UnaryServerInterceptor(j),
        ),
        grpc.ChainStreamInterceptor(
            metrics.StreamServerInterceptor(collector),
            tracing.StreamServerInterceptor("my-service"),
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

## 中间件使用指南

### Endpoint 中间件

Endpoint 中间件用于 `transport.Endpoint` 层，适合服务内部的横切关注点处理。

```go
import (
    "github.com/Tsukikage7/microservice-kit/transport"
    "github.com/Tsukikage7/microservice-kit/observability/metrics"
    "github.com/Tsukikage7/microservice-kit/observability/tracing"
    "github.com/Tsukikage7/microservice-kit/middleware/ratelimit"
    "github.com/Tsukikage7/microservice-kit/middleware/retry"
    "github.com/Tsukikage7/microservice-kit/auth/jwt"
)

// 定义 Endpoint
var myEndpoint transport.Endpoint = func(ctx context.Context, req any) (any, error) {
    return process(req)
}

// 服务端中间件（从外到内执行）
myEndpoint = transport.Chain(
    metrics.EndpointMiddleware(collector, "my-service", "MyMethod"),
    tracing.EndpointMiddleware("my-service", "MyMethod"),
    ratelimit.EndpointMiddleware(limiter),
    jwt.NewParser(j),
)(myEndpoint)

// 客户端中间件
clientEndpoint = transport.Chain(
    metrics.EndpointMiddleware(collector, "my-service", "MyMethod"),
    tracing.EndpointMiddleware("my-service", "MyMethod"),
    jwt.NewSigner(j),
    retry.EndpointMiddleware(retryConfig),
)(clientEndpoint)
```

### HTTP 中间件

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/users", handleUsers)

// 应用中间件（从外到内执行）
var handler http.Handler = mux
handler = metrics.HTTPMiddleware(collector)(handler)
handler = tracing.HTTPMiddleware("my-service")(handler)
handler = ratelimit.HTTPMiddleware(limiter)(handler)
handler = jwt.HTTPMiddleware(j)(handler)
```

### gRPC 拦截器

```go
// 服务端
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        metrics.UnaryServerInterceptor(collector),
        tracing.UnaryServerInterceptor("my-service"),
        ratelimit.UnaryServerInterceptor(limiter),
        jwt.UnaryServerInterceptor(j),
    ),
    grpc.ChainStreamInterceptor(
        metrics.StreamServerInterceptor(collector),
        tracing.StreamServerInterceptor("my-service"),
        ratelimit.StreamServerInterceptor(limiter),
        jwt.StreamServerInterceptor(j),
    ),
)

// 客户端
conn, _ := grpc.Dial("localhost:50051",
    grpc.WithChainUnaryInterceptor(
        metrics.UnaryClientInterceptor(collector),
        tracing.UnaryClientInterceptor("my-service"),
        retry.UnaryClientInterceptor(retryConfig),
    ),
    grpc.WithChainStreamInterceptor(
        metrics.StreamClientInterceptor(collector),
        tracing.StreamClientInterceptor("my-service"),
        retry.StreamClientInterceptor(retryConfig),
    ),
)
```

### 请求上下文提取中间件

```go
import (
    "github.com/Tsukikage7/microservice-kit/request"
    "github.com/Tsukikage7/microservice-kit/request/clientip"
)

// 方式 1: 使用组合中间件（默认启用 ClientIP, UserAgent, Locale, Referer）
handler = request.HTTPMiddleware()(handler)

// 方式 2: 启用所有解析器（包括 Device, Bot）
handler = request.HTTPMiddleware(request.WithAll())(handler)

// 方式 3: 自定义配置
handler = request.HTTPMiddleware(
    request.WithClientIP(clientip.WithTrustedProxies("10.0.0.0/8")),
    request.WithBot(),
    request.DisableReferer(),
)(handler)

// 方式 4: 单独使用子模块
handler = clientip.HTTPMiddleware()(handler)

// 在 handler 中获取请求信息
func myHandler(w http.ResponseWriter, r *http.Request) {
    // 获取聚合信息
    info := request.FromContext(r.Context())

    // 或单独获取
    ip, _ := clientip.FromContext(r.Context())
    ua, _ := useragent.FromContext(r.Context())
    loc, _ := locale.FromContext(r.Context())
}
```

## 中间件执行顺序

推荐的中间件执行顺序（从外到内）：

1. **Metrics** - 首先记录请求指标
2. **Tracing** - 创建追踪 span
3. **RateLimit** - 限流保护
4. **Request** - 请求上下文提取（ClientIP, UserAgent 等）
5. **Auth/JWT** - 认证验证
6. **Retry** - 重试逻辑（客户端）
7. **Business Logic** - 业务处理

## 基础设施组件

### Logger - 结构化日志

```go
log := logger.MustNewLogger(&logger.Config{
    Level:      "info",
    Format:     "json",
    OutputPath: "stdout",
})
defer log.Close()

log.Info("服务启动", "port", 8080)
log.Error("请求失败", "error", err, "request_id", reqID)
```

### Cache - 缓存

```go
// 内存缓存
memCache := cache.MustNewCache(cache.NewMemoryConfig(), log)

// Redis 缓存
redisCache := cache.MustNewCache(&cache.Config{
    Type: cache.TypeRedis,
    Addr: "localhost:6379",
}, log)

// 使用
ctx := context.Background()
cache.Set(ctx, "key", "value", 5*time.Minute)
value, err := cache.Get(ctx, "key")
```

### Database - 数据库

```go
db := database.MustNewDatabase(&database.Config{
    Driver:        database.DriverMySQL,
    DSN:           "user:pass@tcp(localhost:3306)/dbname",
    AutoMigrate:   true,
    SlowThreshold: 100 * time.Millisecond,
    Pool: database.PoolConfig{
        MaxOpen:     100,
        MaxIdle:     10,
        MaxLifetime: time.Hour,
    },
}, log)
defer db.Close()

// 自动迁移
db.AutoMigrate(&User{})

// 获取 GORM 实例
if gormDB, ok := database.AsGORM(db); ok {
    gormDB.GORM().Create(&User{Name: "John"})
}
```

### Discovery - 服务发现

```go
// Consul
discovery := discovery.MustNewDiscovery(&discovery.Config{
    Type: discovery.TypeConsul,
    Addr: "localhost:8500",
}, log)

// 注册服务
id, _ := discovery.Register(ctx, "my-service", "localhost:8080")
defer discovery.Deregister(ctx, id)

// 发现服务
instances, _ := discovery.Discover(ctx, "other-service")
```

### Messaging - 消息队列

```go
client, _ := messaging.NewClient(
    messaging.WithBrokers([]string{"localhost:9092"}),
    messaging.WithLogger(log),
)

// 生产者
producer, _ := client.Producer()
producer.Send(ctx, "topic", []byte("message"))

// 消费者
consumer, _ := client.Consumer("group-id")
consumer.Subscribe(ctx, "topic", func(msg *messaging.Message) error {
    // 处理消息
    return nil
})
```

### Scheduler - 定时任务

```go
scheduler := scheduler.MustNewScheduler(
    scheduler.WithLogger(log),
)

// 添加任务
scheduler.AddFunc("@every 1m", func() {
    // 每分钟执行
})

scheduler.AddFunc("0 0 * * *", func() {
    // 每天零点执行
})

scheduler.Start()
defer scheduler.Stop()
```

## 工厂函数命名规范

本工具包遵循统一的工厂函数命名规范：

| 模式 | 说明 | 示例 |
|------|------|------|
| `NewXXX` | 返回 `(T, error)` | `NewDatabase(cfg, log)` |
| `MustNewXXX` | 失败时 panic | `MustNewDatabase(cfg, log)` |
| `DefaultConfig` | 返回默认配置 | `logger.DefaultConfig()` |

## 各包详细文档

### 可观测性 (observability/)
- **[observability/metrics](./observability/metrics/)** - Prometheus 指标收集
- **[observability/tracing](./observability/tracing/)** - OpenTelemetry 链路追踪

### 中间件 (middleware/)
- **[middleware/ratelimit](./middleware/ratelimit/)** - 限流（令牌桶、滑动窗口、固定窗口、分布式）
- **[middleware/retry](./middleware/retry/)** - 重试机制（固定/指数/线性退避）
- **[middleware/recovery](./middleware/recovery/)** - Panic 恢复
- **[middleware/timeout](./middleware/timeout/)** - 超时控制
- **[middleware/idempotency](./middleware/idempotency/)** - 幂等性保证
- **[middleware/semaphore](./middleware/semaphore/)** - 并发控制

### 请求上下文 (request/)
- **[request](./request/)** - 请求上下文提取组合层
- **[request/clientip](./request/clientip/)** - 客户端 IP 提取、地理位置、ACL
- **[request/useragent](./request/useragent/)** - User-Agent 解析
- **[request/deviceinfo](./request/deviceinfo/)** - 设备信息（Client Hints 优先）
- **[request/botdetect](./request/botdetect/)** - 机器人检测
- **[request/locale](./request/locale/)** - 语言区域设置
- **[request/referer](./request/referer/)** - 来源页面解析、UTM 参数
- **[request/activity](./request/activity/)** - 用户活动追踪

### 存储 (storage/)
- **[storage/cache](./storage/cache/)** - 缓存（内存、Redis）
- **[storage/database](./storage/database/)** - 数据库（GORM）
- **[storage/lock](./storage/lock/)** - 分布式锁

### 工具 (util/)
- **[util/pagination](./util/pagination/)** - 分页工具
- **[util/sorting](./util/sorting/)** - 排序工具
- **[util/collections](./util/collections/)** - 集合工具（TreeMap、TreeSet、LinkedList）
- **[util/pbjson](./util/pbjson/)** - Protobuf JSON 序列化（零值字段输出）

### 核心组件
- **[transport](./transport/)** - 传输层抽象，定义 Endpoint 和 Middleware
- **[auth](./auth/)** - 认证授权（JWT、API Key、RBAC）
- **[logger](./logger/)** - 结构化日志
- **[config](./config/)** - 配置管理
- **[discovery](./discovery/)** - 服务发现（Consul、etcd）
- **[messaging](./messaging/)** - 消息队列（Kafka）
- **[scheduler](./scheduler/)** - 定时任务调度

### 分布式模式
- **[domain](./domain/)** - 领域驱动设计（聚合根、领域事件）
- **[cqrs](./cqrs/)** - 命令查询职责分离
- **[saga](./saga/)** - Saga 分布式事务

## 设计原则

本工具包遵循以下设计原则：

- **KISS** - 保持简单，避免过度设计
- **DRY** - 抽象通用模式，减少重复代码
- **SOLID** - 单一职责，接口隔离
- **可组合** - 中间件可自由组合
- **可扩展** - 支持自定义实现

## License

MIT License
