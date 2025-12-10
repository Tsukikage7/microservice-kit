# Messaging

消息队列客户端库，提供统一的生产者和消费者接口，支持多种消息队列实现。

## 特性

- **统一入口**: Client 统一管理生产者和消费者生命周期
- **Options 模式**: 灵活的配置方式，支持链式调用
- **最佳实践内置**: Kafka 实现内置生产级最佳实践配置
- **类型安全**: Key/Value 使用 `[]byte`，序列化由调用方控制
- **重试与死信队列**: 内置消息重试和死信队列支持
- **优雅关闭**: 支持超时控制的优雅关闭
- **健康检查**: 内置健康检查，支持 K8s probe
- **指标监控**: 内置 Metrics 收集
- **链路追踪**: 内置 Tracing 支持
- **批量发送**: 支持批量消息发送

## 支持的消息队列

- [x] Apache Kafka
- [x] RabbitMQ
- [ ] RocketMQ (计划中)

## 安装

```bash
go get github.com/Tsukikage7/microservice-kit/messaging
```

## 快速开始

### 使用 Client（推荐）

```go
// 创建客户端
client, err := messaging.NewClient(
    messaging.WithBrokers([]string{"localhost:9092"}),
    messaging.WithLogger(log),
    messaging.WithMetrics(messaging.NewMetrics()),
    messaging.WithTracing(),
)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// 创建生产者
producer, err := client.Producer()
if err != nil {
    log.Fatal(err)
}

// 发送消息
msg, err := producer.SendMessage(ctx, &messaging.Message{
    Topic: "orders",
    Key:   []byte("order-123"),
    Value: []byte(`{"id":"123","amount":100}`),
})

// 创建消费者
consumer, err := client.Consumer("order-service",
    messaging.WithRetry(3, time.Second),
)
if err != nil {
    log.Fatal(err)
}

// 消费消息
consumer.Consume(ctx, []string{"orders"}, func(msg *messaging.Message) error {
    fmt.Printf("收到消息: %s\n", msg.Value)
    return nil
})
```

### 直接使用 Producer/Consumer

```go
cfg := &messaging.Config{
    Type:    "kafka",
    Brokers: []string{"localhost:9092"},
}

// 创建生产者
producer, err := messaging.NewProducer(cfg,
    messaging.WithProducerLogger(log),
)
if err != nil {
    log.Fatal(err)
}
defer producer.Close()

// 创建消费者
consumer, err := messaging.NewConsumer(cfg, "order-service",
    messaging.WithConsumerLogger(log),
)
if err != nil {
    log.Fatal(err)
}
defer consumer.Close()
```

## Client API

### Client Options

| 选项 | 说明 |
|------|------|
| `WithBrokers(brokers)` | 设置服务器地址（必填） |
| `WithLogger(log)` | 设置日志记录器 |
| `WithClientID(id)` | 设置客户端ID |
| `WithType(type)` | 设置消息队列类型，默认 "kafka" |
| `WithMetrics(metrics)` | 启用指标监控 |
| `WithTracing()` | 启用链路追踪 |

### Client 方法

```go
// 创建生产者
producer, err := client.Producer(opts...)

// 创建消费者
consumer, err := client.Consumer(groupID, opts...)

// 健康检查
err := client.HealthCheck(ctx)

// 优雅关闭（带超时）
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
client.Shutdown(ctx)

// 关闭（等同于 Shutdown(context.Background())）
client.Close()

// 获取统计信息
stats := client.Stats()
```

## Producer API

### Producer Options

| 选项 | 说明 |
|------|------|
| `WithProducerLogger(log)` | 设置日志记录器 |

### Producer 方法

```go
// 发送单条消息
msg, err := producer.SendMessage(ctx, &messaging.Message{
    Topic: "orders",
    Key:   []byte("order-123"),
    Value: []byte(`{"id":"123"}`),
    Headers: map[string]string{
        "trace-id": "abc-123",
    },
})

// 批量发送消息
msgs, err := producer.SendBatch(ctx, []*messaging.Message{msg1, msg2, msg3})
```

## Consumer API

### Consumer Options

| 选项 | 说明 |
|------|------|
| `WithConsumerLogger(log)` | 设置日志记录器 |
| `WithRetry(maxRetries, interval)` | 设置重试策略（指数退避） |
| `WithDeadLetterQueue(topic, producer)` | 设置死信队列 |

### Consumer 方法

```go
// 消费消息
err := consumer.Consume(ctx, []string{"orders"}, func(msg *messaging.Message) error {
    // 处理消息...
    return nil  // 返回 nil 表示成功，自动提交
})

// 手动提交（特殊场景）
err := consumer.CommitMessage(msg)
```

## Message

| 字段 | 类型 | 说明 |
|------|------|------|
| Topic | string | 消息主题（必填） |
| Key | []byte | 消息键，用于分区路由 |
| Value | []byte | 消息内容 |
| Headers | map[string]string | 消息头 |
| Partition | int32 | 分区号（发送后填充） |
| Offset | int64 | 偏移量（发送后填充） |
| Timestamp | time.Time | 时间戳 |

## 健康检查

```go
// 用于 K8s liveness/readiness probe
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    if err := client.HealthCheck(r.Context()); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
})
```

## 指标监控

```go
// 创建指标收集器
metrics := messaging.NewMetrics()

// 创建客户端时启用
client, _ := messaging.NewClient(
    messaging.WithBrokers(brokers),
    messaging.WithMetrics(metrics),
)

// 获取指标快照
snapshot := metrics.Snapshot()
fmt.Printf("发送总数: %d\n", snapshot.MessagesSent)
fmt.Printf("消费总数: %d\n", snapshot.MessagesConsumed)
fmt.Printf("发送错误: %d\n", snapshot.SendErrors)
fmt.Printf("消费错误: %d\n", snapshot.ConsumeErrors)
fmt.Printf("重试次数: %d\n", snapshot.RetryCount)
fmt.Printf("死信队列: %d\n", snapshot.DLQCount)
fmt.Printf("发送延迟(avg): %v\n", snapshot.SendLatencyAvg)
fmt.Printf("发送延迟(P99): %v\n", snapshot.SendLatencyP99)

// 按 topic 统计
for topic, count := range snapshot.TopicSent {
    fmt.Printf("Topic %s 发送: %d\n", topic, count)
}

// 重置指标
metrics.Reset()
```

## 链路追踪

```go
// 启用追踪
client, _ := messaging.NewClient(
    messaging.WithBrokers(brokers),
    messaging.WithTracing(),
)

// 使用自定义追踪器
tracer := messaging.NewSimpleTracer()
producer.SetTracer(tracer)
consumer.SetTracer(tracer)

// 追踪信息会自动通过 Headers 传递:
// - x-trace-id
// - x-span-id
// - x-parent-span-id
// - x-sampled
```

### 集成 OpenTelemetry

```go
// 实现 Tracer 接口即可集成 OpenTelemetry
type OTelTracer struct {
    tracer trace.Tracer
}

func (t *OTelTracer) Extract(ctx context.Context) *messaging.TraceContext { ... }
func (t *OTelTracer) Inject(ctx context.Context, tc *messaging.TraceContext) context.Context { ... }
func (t *OTelTracer) StartSpan(ctx context.Context, name string) (context.Context, messaging.Span) { ... }
```

## 重试与死信队列

```go
// 创建带重试和死信队列的消费者
dlqProducer, _ := client.Producer()

consumer, _ := client.Consumer("order-service",
    messaging.WithRetry(3, time.Second),                      // 重试3次，间隔 1s, 2s, 4s
    messaging.WithDeadLetterQueue("orders-dlq", dlqProducer), // 死信队列
)

consumer.Consume(ctx, []string{"orders"}, func(msg *messaging.Message) error {
    // 处理失败会重试
    // 重试耗尽后发送到死信队列
    return processOrder(msg)
})
```

### 死信队列消息格式

| Header | 说明 |
|--------|------|
| `x-original-topic` | 原始消息主题 |
| `x-original-partition` | 原始消息分区 |
| `x-original-offset` | 原始消息偏移量 |
| `x-error-message` | 最后一次错误信息 |
| `x-consumer-group` | 消费者组ID |

## 优雅关闭

```go
// 监听信号
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

// 等待信号
<-ctx.Done()

// 优雅关闭（最多等待30秒）
shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
client.Shutdown(shutdownCtx)
```

## 错误处理

```go
import "errors"

if errors.Is(err, messaging.ErrProducerClosed) {
    // 生产者已关闭
}

if errors.Is(err, messaging.ErrClientClosed) {
    // 客户端已关闭
}
```

| 错误 | 说明 |
|------|------|
| ErrNoBrokers | 未配置服务器地址 |
| ErrCreateClient | 创建客户端失败 |
| ErrClientClosed | 客户端已关闭 |
| ErrNoBrokersAvailable | 没有可用的服务器 |
| ErrHealthCheck | 健康检查失败 |
| ErrEmptyGroupID | 消费者组ID为空 |
| ErrProducerClosed | 生产者已关闭 |
| ErrNilMessage | 消息为空 |
| ErrEmptyTopic | 消息主题为空 |
| ErrNoTopics | 未指定消费主题 |
| ErrNilHandler | 消息处理器为空 |
| ErrNoActiveSession | 没有活跃的消费者会话 |
| ErrUnsupportedType | 不支持的消息队列类型 |
| ErrCreateProducer | 创建生产者失败 |
| ErrCreateConsumer | 创建消费者失败 |
| ErrSendMessage | 消息发送失败 |
| ErrBatchSend | 批量发送失败 |

## Kafka 最佳实践配置

### 生产者

- `Idempotent`: true - 幂等性，保证消息不重复
- `RequiredAcks`: WaitForAll - 等待所有副本确认
- `Retry.Max`: 3 - 最多重试3次
- `Compression`: Snappy - 使用Snappy压缩

### 消费者

- `AutoCommit`: 禁用 - 手动提交，保证消息处理完成后再确认
- `Offsets.Initial`: Newest - 从最新消息开始消费

## RabbitMQ

### 配置

```go
cfg := &messaging.Config{
    Type: "rabbitmq",
    URL:  "amqp://user:pass@localhost:5672/vhost",
    RabbitMQ: &messaging.RabbitMQConfig{
        Exchange:      "orders",      // 交换机名称
        ExchangeType:  "direct",      // 交换机类型: direct, fanout, topic, headers
        Durable:       true,          // 持久化
        AutoAck:       false,         // 自动确认（建议关闭）
        PrefetchCount: 10,            // 预取数量
        Confirm:       true,          // 发布确认
    },
}

// 创建生产者
producer, err := messaging.NewProducer(cfg, messaging.WithProducerLogger(log))
if err != nil {
    log.Fatal(err)
}
defer producer.Close()

// 发送消息
msg, err := producer.SendMessage(ctx, &messaging.Message{
    Topic: "order.created",  // 作为 routing key
    Key:   []byte("order-123"),
    Value: []byte(`{"id":"123"}`),
})

// 创建消费者
consumer, err := messaging.NewConsumer(cfg, "order-service", messaging.WithConsumerLogger(log))
if err != nil {
    log.Fatal(err)
}
defer consumer.Close()

// 消费消息
consumer.Consume(ctx, []string{"order.created"}, func(msg *messaging.Message) error {
    fmt.Printf("收到消息: %s\n", msg.Value)
    return nil
})
```

### 配置选项

| 选项 | 类型 | 说明 |
|------|------|------|
| Exchange | string | 交换机名称 |
| ExchangeType | string | 交换机类型: direct, fanout, topic, headers |
| Durable | bool | 是否持久化（建议开启） |
| AutoAck | bool | 是否自动确认（建议关闭） |
| PrefetchCount | int | 预取数量（QoS） |
| Confirm | bool | 是否启用发布确认 |

### 交换机类型

| 类型 | 说明 |
|------|------|
| direct | 精确匹配 routing key |
| fanout | 广播到所有绑定队列 |
| topic | 模式匹配 routing key（支持 * 和 #） |
| headers | 基于消息头匹配 |

### 特性

- **自动重连**: 连接断开后自动重连
- **发布确认**: 支持 publisher confirms 模式
- **预取控制**: 支持 QoS 设置
- **手动/自动确认**: 支持手动和自动消息确认

## 许可证

MIT License
