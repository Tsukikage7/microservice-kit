# Idempotency

请求幂等性控制包，保证同一请求多次执行的效果与执行一次相同。

## 功能特性

- **HTTP 幂等性中间件** - 基于请求头的幂等控制
- **gRPC 幂等性拦截器** - 基于元数据的幂等控制
- **Endpoint 幂等性中间件** - 通用幂等控制
- **多种存储后端** - 支持内存和 Redis
- **可配置过期时间** - 控制幂等键的有效期
- **并发安全** - 支持分布式环境下的并发控制

## 工作原理

```
1. 客户端发送请求，携带幂等键（Idempotency-Key）
2. 服务端检查幂等键是否已存在
   - 如果存在且已完成：返回之前的结果
   - 如果存在且处理中：返回 409 Conflict
   - 如果不存在：获取锁，执行请求，保存结果
3. 后续相同幂等键的请求直接返回缓存结果
```

## 快速开始

### 内存存储

```go
import "github.com/Tsukikage7/microservice-kit/idempotency"

store := idempotency.NewMemoryStore()
defer store.Close()
```

### Redis 存储

```go
import "github.com/Tsukikage7/microservice-kit/idempotency"

store := idempotency.NewRedisStore(redisClient,
    idempotency.WithKeyPrefix("myapp:idempotency:"),
)
```

### HTTP 中间件

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/orders", createOrderHandler)

// 使用幂等性中间件
handler := idempotency.HTTPMiddleware(store)(mux)
http.ListenAndServe(":8080", handler)
```

**客户端请求：**

```bash
curl -X POST http://localhost:8080/api/orders \
  -H "Idempotency-Key: order-12345" \
  -H "Content-Type: application/json" \
  -d '{"item":"book","quantity":1}'
```

### gRPC 拦截器

```go
srv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        idempotency.UnaryServerInterceptor(store),
    ),
)
```

**客户端请求：**

```go
ctx := metadata.AppendToOutgoingContext(ctx, "x-idempotency-key", "order-12345")
resp, err := client.CreateOrder(ctx, req)
```

### Endpoint 中间件

```go
// 方式1: 实现 IdempotentRequest 接口
type CreateOrderRequest struct {
    OrderID string
    Item    string
}

func (r *CreateOrderRequest) IdempotencyKey() string {
    return r.OrderID
}

endpoint := idempotency.EndpointMiddleware(store)(createOrderEndpoint)

// 方式2: 自定义键提取
endpoint := idempotency.EndpointMiddleware(store,
    idempotency.WithKeyExtractor(func(ctx any) string {
        if req, ok := ctx.(*CreateOrderRequest); ok {
            return req.OrderID
        }
        return ""
    }),
)(createOrderEndpoint)
```

## 配置选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithKeyExtractor(fn)` | 默认提取规则 | 自定义幂等键提取函数 |
| `WithTTL(duration)` | 24h | 幂等键过期时间 |
| `WithLogger(log)` | - | 设置日志记录器 |
| `WithSkipOnError(bool)` | false | 存储错误时是否跳过检查 |
| `WithLockTimeout(duration)` | 30s | 处理锁超时时间 |

### Redis 存储选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithKeyPrefix(prefix)` | `idempotency:` | Redis 键前缀 |

## 使用场景

### 支付接口

```go
func createPayment(w http.ResponseWriter, r *http.Request) {
    // 幂等性已在中间件中处理
    // 即使用户重复点击，也只会扣款一次
    payment, err := paymentService.Create(r.Context(), req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(payment)
}

// 配置
handler := idempotency.HTTPMiddleware(store,
    idempotency.WithTTL(24*time.Hour), // 24小时内重复请求返回相同结果
)(paymentHandler)
```

### 订单创建

```go
type CreateOrderRequest struct {
    OrderNo string `json:"order_no"` // 客户端生成的订单号
    Items   []Item `json:"items"`
}

func (r *CreateOrderRequest) IdempotencyKey() string {
    return "order:" + r.OrderNo
}

endpoint := idempotency.EndpointMiddleware(store)(createOrderEndpoint)
```

### 消息发送

```go
// 使用消息ID作为幂等键，防止重复发送
handler := idempotency.HTTPMiddleware(store,
    idempotency.WithKeyExtractor(func(ctx any) string {
        if r, ok := ctx.(*http.Request); ok {
            return r.Header.Get("X-Message-ID")
        }
        return ""
    }),
    idempotency.WithTTL(1*time.Hour),
)(sendMessageHandler)
```

## 最佳实践

### 1. 选择合适的幂等键

```go
// ✅ 好的幂等键：具有业务含义，客户端生成
"order:ORD-2024-001"
"payment:PAY-UUID-123"
"message:MSG-HASH-ABC"

// ❌ 不好的幂等键：随机生成，每次不同
uuid.New().String()
```

### 2. 设置合理的过期时间

```go
// 支付场景：较长的过期时间
idempotency.WithTTL(24 * time.Hour)

// 消息推送：较短的过期时间
idempotency.WithTTL(1 * time.Hour)

// 数据同步：根据同步频率设置
idempotency.WithTTL(5 * time.Minute)
```

### 3. 处理并发请求

```go
// 当同一幂等键的请求正在处理中时，新请求会收到 409 Conflict
// 客户端应该重试

func createOrderWithRetry(ctx context.Context, req *Request) (*Response, error) {
    for i := 0; i < 3; i++ {
        resp, err := client.CreateOrder(ctx, req)
        if err == nil {
            return resp, nil
        }

        // 如果是请求进行中，等待后重试
        if status.Code(err) == codes.Aborted {
            time.Sleep(time.Second)
            continue
        }

        return nil, err
    }
    return nil, errors.New("max retries exceeded")
}
```

### 4. 分布式环境使用 Redis

```go
// 单机测试可用内存存储
store := idempotency.NewMemoryStore()

// 生产环境使用 Redis
store := idempotency.NewRedisStore(redisClient,
    idempotency.WithKeyPrefix("prod:idempotency:"),
)
```

### 5. 错误处理策略

```go
// 严格模式：存储错误时拒绝请求
handler := idempotency.HTTPMiddleware(store,
    idempotency.WithSkipOnError(false), // 默认
)

// 宽松模式：存储错误时跳过幂等检查
handler := idempotency.HTTPMiddleware(store,
    idempotency.WithSkipOnError(true),
    idempotency.WithLogger(log), // 记录跳过的情况
)
```

## 响应码说明

### HTTP

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功（首次或缓存命中） |
| 409 | 相同幂等键的请求正在处理中 |
| 500 | 存储操作失败 |

### gRPC

| 状态码 | 说明 |
|--------|------|
| OK | 请求成功 |
| Aborted | 相同幂等键的请求正在处理中 |
| Internal | 存储操作失败 |

## 注意事项

1. **幂等键必须唯一**：不同请求使用相同的幂等键会返回第一次请求的结果
2. **结果会被序列化**：确保响应对象可以被 JSON 序列化
3. **内存存储不持久化**：重启后数据丢失，生产环境建议使用 Redis
4. **GET 请求默认跳过**：只对 POST/PUT/PATCH 方法启用幂等检查
