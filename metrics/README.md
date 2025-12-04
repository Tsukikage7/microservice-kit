# metrics

提供基于 Prometheus 的指标收集功能，支持 HTTP、gRPC 和自定义指标。

## 安装

```go
import "github.com/Tsukikage7/microservice-kit/metrics"
```

## 快速开始

```go
// 1. 创建指标收集器
cfg := &metrics.Config{
    Namespace: "my_service",
    Path:      "/metrics",
}

collector, err := metrics.New(cfg)
if err != nil {
    log.Fatal(err)
}

// 2. 使用中间件自动采集（推荐）
mux := http.NewServeMux()
mux.HandleFunc("/api/users", handleUsers)

handler := metrics.HTTPMiddleware(collector)(mux)

// 3. 暴露指标端点
http.Handle(collector.GetPath(), collector.GetHandler())
http.ListenAndServe(":8080", handler)
```

## API

### 创建收集器

#### New

创建指标收集器。

```go
func New(cfg *Config) (Collector, error)
```

#### MustNew

创建指标收集器，失败时 panic。

```go
func MustNew(cfg *Config) Collector
```

### HTTP 指标

#### RecordHTTPRequest

记录 HTTP 请求指标。

```go
func (c *Collector) RecordHTTPRequest(method, path, statusCode string, duration time.Duration, requestSize, responseSize float64)
```

### gRPC 指标

#### RecordGRPCRequest

记录 gRPC 请求指标。

```go
func (c *Collector) RecordGRPCRequest(method, service, statusCode string, duration time.Duration)
```

### 中间件/拦截器

#### HTTPMiddleware

返回 HTTP 指标采集中间件，自动记录请求指标。

```go
func HTTPMiddleware(collector *PrometheusCollector) func(http.Handler) http.Handler
```

#### UnaryServerInterceptor

返回 gRPC 一元服务端指标拦截器。

```go
func UnaryServerInterceptor(collector *PrometheusCollector) grpc.UnaryServerInterceptor
```

#### StreamServerInterceptor

返回 gRPC 流式服务端指标拦截器。

```go
func StreamServerInterceptor(collector *PrometheusCollector) grpc.StreamServerInterceptor
```

#### UnaryClientInterceptor

返回 gRPC 一元客户端指标拦截器。

```go
func UnaryClientInterceptor(collector *PrometheusCollector) grpc.UnaryClientInterceptor
```

#### StreamClientInterceptor

返回 gRPC 流式客户端指标拦截器。

```go
func StreamClientInterceptor(collector *PrometheusCollector) grpc.StreamClientInterceptor
```

### 系统指标

#### RecordPanic

记录 panic 事件。

```go
func (c *Collector) RecordPanic(service, method, endpoint string)
```

#### UpdateGoroutineCount

更新 goroutine 数量。

```go
func (c *Collector) UpdateGoroutineCount(count int)
```

#### UpdateMemoryUsage

更新内存使用量。

```go
func (c *Collector) UpdateMemoryUsage(bytes int64)
```

### 自定义指标

#### IncrementCounter

增加自定义计数器。

```go
func (c *Collector) IncrementCounter(name string, labels map[string]string)
```

#### ObserveHistogram

观察自定义直方图。

```go
func (c *Collector) ObserveHistogram(name string, value float64, labels map[string]string)
```

#### SetGauge

设置自定义仪表盘。

```go
func (c *Collector) SetGauge(name string, value float64, labels map[string]string)
```

### Handler

#### GetHandler

返回 metrics 的 HTTP 处理器。

```go
func (c *Collector) GetHandler() http.Handler
```

#### GetPath

返回 metrics 路径。

```go
func (c *Collector) GetPath() string
```

### 错误

| 错误 | 说明 |
|------|------|
| `ErrNilConfig` | 指标配置为空 |
| `ErrRegisterMetric` | 注册指标失败 |
| `ErrEmptyNamespace` | 命名空间为空 |

## 使用示例

### HTTP 服务（使用中间件）

```go
package main

import (
    "log"
    "net/http"
    "runtime"
    "time"

    "github.com/Tsukikage7/microservice-kit/metrics"
)

func main() {
    // 创建指标收集器
    cfg := &metrics.Config{
        Namespace: "user_service",
        Path:      "/metrics",
    }

    collector, err := metrics.New(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // 创建路由
    mux := http.NewServeMux()
    mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
        // 业务逻辑，无需关心指标采集
        w.Write([]byte(`{"users": []}`))
    })

    // 使用中间件自动采集 HTTP 指标
    handler := metrics.HTTPMiddleware(collector)(mux)

    // 暴露指标端点
    http.Handle(collector.GetPath(), collector.GetHandler())

    // 启动系统指标采集
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        for range ticker.C {
            var m runtime.MemStats
            runtime.ReadMemStats(&m)
            collector.UpdateGoroutineCount(runtime.NumGoroutine())
            collector.UpdateMemoryUsage(int64(m.Alloc))
        }
    }()

    log.Println("Server starting on :8080")
    http.ListenAndServe(":8080", handler)
}
```

### gRPC 服务（使用拦截器）

```go
package main

import (
    "log"
    "net"

    "github.com/Tsukikage7/microservice-kit/metrics"
    "google.golang.org/grpc"
)

func main() {
    cfg := &metrics.Config{
        Namespace: "order_service",
    }

    collector, err := metrics.New(cfg)
    if err != nil {
        panic(err)
    }

    // 创建 gRPC 服务器，使用拦截器自动采集指标
    server := grpc.NewServer(
        grpc.UnaryInterceptor(metrics.UnaryServerInterceptor(collector)),
        grpc.StreamInterceptor(metrics.StreamServerInterceptor(collector)),
    )

    // 注册服务...
    // pb.RegisterOrderServiceServer(server, &orderService{})

    lis, _ := net.Listen("tcp", ":50051")
    log.Println("gRPC server starting on :50051")
    server.Serve(lis)
}
```

### gRPC 客户端（使用拦截器）

```go
package main

import (
    "github.com/Tsukikage7/microservice-kit/metrics"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    cfg := &metrics.Config{
        Namespace: "user_service",
    }

    collector, _ := metrics.New(cfg)

    // 创建带指标拦截器的 gRPC 连接
    conn, _ := grpc.Dial("order-service:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithUnaryInterceptor(metrics.UnaryClientInterceptor(collector)),
        grpc.WithStreamInterceptor(metrics.StreamClientInterceptor(collector)),
    )
    defer conn.Close()

    // 使用连接，指标自动采集
    // client := pb.NewOrderServiceClient(conn)
}
```

### 自定义指标

```go
// 计数器：统计事件次数
collector.IncrementCounter("orders_created", map[string]string{
    "type":   "standard",
    "region": "cn-east",
})

// 直方图：记录延迟分布
collector.ObserveHistogram("db_query_duration_seconds", 0.05, map[string]string{
    "query": "select_users",
    "db":    "primary",
})

// 仪表盘：记录当前值
collector.SetGauge("active_connections", 42, map[string]string{
    "server": "main",
    "pool":   "default",
})
```

### Panic 恢复中间件（需手动实现）

```go
func recoveryMiddleware(collector *metrics.PrometheusCollector, serviceName string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            defer func() {
                if err := recover(); err != nil {
                    collector.RecordPanic(serviceName, r.Method, r.URL.Path)
                    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                }
            }()
            next.ServeHTTP(w, r)
        })
    }
}
```

### 配置文件示例

```yaml
metrics:
  namespace: my_service
  path: /metrics
```

## 内置指标

### HTTP 指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `{namespace}_http_requests_total` | Counter | method, path, status_code | HTTP 请求总数 |
| `{namespace}_http_request_duration_seconds` | Histogram | method, path | HTTP 请求耗时 |
| `{namespace}_http_request_size_bytes` | Histogram | method, path | HTTP 请求大小 |
| `{namespace}_http_response_size_bytes` | Histogram | method, path | HTTP 响应大小 |

### gRPC 指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `{namespace}_grpc_requests_total` | Counter | method, service, status_code | gRPC 请求总数 |
| `{namespace}_grpc_request_duration_seconds` | Histogram | method, service | gRPC 请求耗时 |

### 系统指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `{namespace}_system_goroutines` | Gauge | - | Goroutine 数量 |
| `{namespace}_system_memory_usage_bytes` | Gauge | - | 内存使用量 |
| `{namespace}_system_panic_total` | Counter | service, method, endpoint | Panic 次数 |

## 特性

- **Prometheus 标准**: 完全兼容 Prometheus 格式
- **中间件/拦截器**: HTTP 和 gRPC 自动采集，业务代码零侵入
- **自定义指标**: 支持动态创建 Counter、Histogram、Gauge
- **线程安全**: 并发安全的指标操作
- **独立注册表**: 使用独立注册表，避免全局冲突
