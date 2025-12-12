# Semaphore

信号量并发控制包，用于限制对共享资源的并发访问数量。

## 功能特性

- **本地信号量** - 基于 channel 实现的单机并发控制
- **加权信号量** - 支持一次获取/释放多个许可
- **分布式信号量** - 基于 Redis 的分布式并发控制
- **中间件支持** - Endpoint、HTTP、gRPC 中间件
- **阻塞/非阻塞** - 支持两种获取模式

## 快速开始

### 本地信号量

```go
import "github.com/Tsukikage7/microservice-kit/semaphore"

// 创建信号量，最多允许10个并发
sem := semaphore.NewLocal(10)
defer sem.Close()

// 获取许可
if err := sem.Acquire(ctx); err != nil {
    return err
}
defer sem.Release(ctx)

// 执行需要限制并发的操作
doWork()
```

### 加权信号量

```go
// 创建加权信号量，总共100个许可
sem := semaphore.NewWeightedLocal(100)
defer sem.Close()

// 一次获取10个许可（用于"重型"操作）
if err := sem.AcquireN(ctx, 10); err != nil {
    return err
}
defer sem.ReleaseN(ctx, 10)

doHeavyWork()
```

### 分布式信号量

```go
// 基于 Redis 的分布式信号量
sem := semaphore.NewRedis(redisClient, "api-limit", 100,
    semaphore.WithTTL(30*time.Second),  // 许可过期时间
    semaphore.WithRetryWait(100*time.Millisecond),
)

if err := sem.Acquire(ctx); err != nil {
    return err
}
defer sem.Release(ctx)

doWork()
```

## 中间件

### Endpoint 中间件

```go
sem := semaphore.NewLocal(10)

// 非阻塞模式（默认）：无许可时立即返回错误
endpoint = semaphore.EndpointMiddleware(sem)(endpoint)

// 阻塞模式：无许可时等待
endpoint = semaphore.EndpointMiddleware(sem,
    semaphore.WithBlock(true),
    semaphore.WithMiddlewareLogger(log),
)(endpoint)
```

### HTTP 中间件

```go
sem := semaphore.NewLocal(100)

mux := http.NewServeMux()
mux.HandleFunc("/api", handler)

// 限制 API 并发
limited := semaphore.HTTPMiddleware(sem)(mux)
http.ListenAndServe(":8080", limited)
```

### gRPC 拦截器

```go
sem := semaphore.NewLocal(100)

srv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        semaphore.UnaryServerInterceptor(sem),
    ),
    grpc.ChainStreamInterceptor(
        semaphore.StreamServerInterceptor(sem),
    ),
)
```

## 配置选项

### Redis 信号量选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithTTL(duration)` | 30s | 许可过期时间 |
| `WithRetryWait(duration)` | 100ms | 重试等待时间 |

### 中间件选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithBlock(bool)` | false | 是否阻塞等待 |
| `WithMiddlewareLogger(log)` | - | 日志记录器 |

## 使用场景

### 数据库连接池

```go
// 限制数据库并发查询数
dbSem := semaphore.NewLocal(50)

func QueryDB(ctx context.Context, sql string) (*Result, error) {
    if err := dbSem.Acquire(ctx); err != nil {
        return nil, err
    }
    defer dbSem.Release(ctx)

    return db.Query(ctx, sql)
}
```

### 外部 API 调用限制

```go
// 限制对外部 API 的并发调用（避免被限流）
apiSem := semaphore.NewLocal(10)

func CallExternalAPI(ctx context.Context, req *Request) (*Response, error) {
    if err := apiSem.Acquire(ctx); err != nil {
        return nil, err
    }
    defer apiSem.Release(ctx)

    return externalClient.Call(ctx, req)
}
```

### 文件操作限制

```go
// 限制并发文件操作
fileSem := semaphore.NewWeightedLocal(100)

func ProcessFile(ctx context.Context, file *File) error {
    // 根据文件大小分配不同权重
    weight := file.Size / (1024 * 1024) // MB
    if weight < 1 {
        weight = 1
    }
    if weight > 50 {
        weight = 50
    }

    if err := fileSem.AcquireN(ctx, weight); err != nil {
        return err
    }
    defer fileSem.ReleaseN(ctx, weight)

    return processFile(file)
}
```

### 分布式并发控制

```go
// 跨多个服务实例共享的并发限制
globalSem := semaphore.NewRedis(redisClient, "payment-processor", 50)

func ProcessPayment(ctx context.Context, payment *Payment) error {
    if err := globalSem.Acquire(ctx); err != nil {
        return err
    }
    defer globalSem.Release(ctx)

    return paymentGateway.Process(ctx, payment)
}
```

## 最佳实践

### 1. 合理设置并发数

```go
// 根据下游系统能力设置
// 例如：数据库连接池大小、外部 API 限制等
dbSem := semaphore.NewLocal(dbPoolSize)
```

### 2. 使用 defer 释放

```go
func doWork(ctx context.Context) error {
    if err := sem.Acquire(ctx); err != nil {
        return err
    }
    defer sem.Release(ctx) // 确保释放

    // 即使 panic 也会释放
    return work()
}
```

### 3. 设置合理的超时

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

if err := sem.Acquire(ctx); err != nil {
    if err == context.DeadlineExceeded {
        return ErrTimeout
    }
    return err
}
```

### 4. 分布式场景使用 TTL

```go
// 防止因服务崩溃导致许可无法释放
sem := semaphore.NewRedis(redisClient, "limit", 100,
    semaphore.WithTTL(30*time.Second),
)

// 长时间操作要定期刷新（或使用足够长的 TTL）
```

### 5. 监控并发使用情况

```go
sem := semaphore.NewLocal(100)

// 定期上报指标
go func() {
    ticker := time.NewTicker(time.Minute)
    for range ticker.C {
        available, _ := sem.Available(ctx)
        metrics.Gauge("semaphore_available").Set(float64(available))
        metrics.Gauge("semaphore_used").Set(float64(sem.Size() - available))
    }
}()
```

## 错误处理

| 错误 | 说明 |
|------|------|
| `ErrNoPermit` | 无法获取许可（非阻塞模式） |
| `ErrClosed` | 信号量已关闭 |
| `context.DeadlineExceeded` | 获取许可超时 |
| `context.Canceled` | Context 被取消 |

### HTTP 响应码

| 场景 | 状态码 |
|------|--------|
| 无法获取许可 | 503 Service Unavailable |

### gRPC 状态码

| 场景 | 状态码 |
|------|--------|
| 无法获取许可 | ResourceExhausted |

## 与限流的区别

| 特性 | 信号量 (Semaphore) | 限流 (Rate Limit) |
|------|-------------------|-------------------|
| 控制维度 | 并发数 | 请求速率 |
| 适用场景 | 保护资源（连接池、文件句柄） | 控制流量（QPS、TPS） |
| 阻塞行为 | 可以等待直到有许可 | 通常直接拒绝 |
| 公平性 | FIFO 队列 | 通常无队列 |

**组合使用：**

```go
// 先限流，再限制并发
endpoint = ratelimit.EndpointMiddleware(limiter)(endpoint)  // 100 QPS
endpoint = semaphore.EndpointMiddleware(sem)(endpoint)       // 10 并发
```
