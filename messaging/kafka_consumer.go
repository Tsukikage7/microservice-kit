package messaging

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel/trace"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// ConsumerOption 消费者配置选项.
//
// 使用函数选项模式配置消费者.
type ConsumerOption func(*consumerOptions)

type consumerOptions struct {
	logger            logger.Logger
	maxRetries        int           // 最大重试次数
	retryInterval     time.Duration // 重试间隔（指数退避的基数）
	reconnectInterval time.Duration // 消费循环重连间隔
	deadLetterTopic   string        // 死信队列主题
	dlqProducer       *KafkaProducer // 死信队列生产者
}

// WithConsumerLogger 设置日志记录器.
//
// 用于记录消费者启动、消费错误等日志.
func WithConsumerLogger(log logger.Logger) ConsumerOption {
	return func(o *consumerOptions) {
		o.logger = log
	}
}

// WithRetry 设置重试策略.
//
// 当消息处理失败时，会按指数退避策略重试.
// 重试间隔 = retryInterval * 2^(重试次数-1)
//
// 参数:
//   - maxRetries: 最大重试次数，0 表示不重试
//   - retryInterval: 重试间隔基数
//
// 示例:
//
//	WithRetry(3, time.Second) // 重试3次，间隔 1s, 2s, 4s
func WithRetry(maxRetries int, retryInterval time.Duration) ConsumerOption {
	return func(o *consumerOptions) {
		o.maxRetries = maxRetries
		o.retryInterval = retryInterval
	}
}

// WithDeadLetterQueue 设置死信队列.
//
// 当消息重试耗尽后，会发送到死信队列而不是丢弃.
// 死信队列消息包含原始消息内容和错误信息.
//
// 参数:
//   - topic: 死信队列主题名称
//   - producer: 用于发送死信的生产者实例
//
// 示例:
//
//	WithDeadLetterQueue("orders-dlq", dlqProducer)
func WithDeadLetterQueue(topic string, producer *KafkaProducer) ConsumerOption {
	return func(o *consumerOptions) {
		o.deadLetterTopic = topic
		o.dlqProducer = producer
	}
}

// WithReconnectInterval 设置消费循环重连间隔.
//
// 当消费循环发生错误时，等待指定时间后重试.
// 默认为 1 秒.
//
// 参数:
//   - interval: 重连间隔时间
//
// 示例:
//
//	WithReconnectInterval(5 * time.Second)
func WithReconnectInterval(interval time.Duration) ConsumerOption {
	return func(o *consumerOptions) {
		o.reconnectInterval = interval
	}
}

// KafkaConsumer Kafka 消费者.
//
// 使用消费者组模式，支持自动重平衡.
// 内置最佳实践配置：
//   - AutoCommit: 禁用 (手动提交，保证消息处理完成后再确认)
//   - Offsets.Initial: Newest (从最新消息开始消费)
//
// 示例:
//
//	consumer, err := NewKafkaConsumer(
//	    []string{"localhost:9092"},
//	    "order-service",
//	    WithConsumerLogger(log),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer consumer.Close()
//
//	err = consumer.Consume(ctx, []string{"orders"}, func(msg *Message) error {
//	    fmt.Printf("收到消息: %s\n", msg.Value)
//	    return nil
//	})
type KafkaConsumer struct {
	consumerGroup     sarama.ConsumerGroup
	groupID           string
	handler           MessageHandler
	topics            []string
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	currentSession    sarama.ConsumerGroupSession
	sessionMu         sync.RWMutex
	logger            logger.Logger
	maxRetries        int
	retryInterval     time.Duration
	reconnectInterval time.Duration
	deadLetterTopic   string
	dlqProducer       *KafkaProducer
	metrics           *messagingMetrics
	tracer            *messagingTracer
}

// NewKafkaConsumer 创建 Kafka 消费者.
//
// 参数:
//   - brokers: Kafka 服务器地址列表
//   - groupID: 消费者组ID，同组消费者共享消息
//   - opts: 可选配置项
//
// 返回创建的消费者实例，使用完毕后需调用 Close 关闭.
func NewKafkaConsumer(brokers []string, groupID string, opts ...ConsumerOption) (*KafkaConsumer, error) {
	if groupID == "" {
		return nil, ErrEmptyGroupID
	}

	options := &consumerOptions{}
	for _, opt := range opts {
		opt(options)
	}

	config := sarama.NewConfig()
	config.Version = sarama.V3_8_0_0
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Offsets.AutoCommit.Enable = false

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, errors.Join(ErrCreateConsumer, err)
	}

	// 设置默认重连间隔
	reconnectInterval := options.reconnectInterval
	if reconnectInterval <= 0 {
		reconnectInterval = time.Second
	}

	c := &KafkaConsumer{
		consumerGroup:     consumerGroup,
		groupID:           groupID,
		logger:            options.logger,
		maxRetries:        options.maxRetries,
		retryInterval:     options.retryInterval,
		reconnectInterval: reconnectInterval,
		deadLetterTopic:   options.deadLetterTopic,
		dlqProducer:       options.dlqProducer,
	}

	if c.logger != nil {
		c.logger.With(
			logger.Any("brokers", brokers),
			logger.String("groupID", groupID),
		).Debug("[Messaging] Kafka消费者启动")
	}

	return c, nil
}

// Consume 开始消费消息.
//
// 该方法会启动后台 goroutine 消费消息，调用后立即返回.
// 消息处理成功（handler 返回 nil）后会自动标记偏移量.
//
// 参数:
//   - ctx: 上下文，取消时会停止消费
//   - topics: 要消费的主题列表
//   - handler: 消息处理函数，返回 nil 表示处理成功
//
// 注意：
//   - 如果 handler 返回错误，消息不会被确认，会在下次重平衡后重新消费
//   - 调用 Close 会等待所有 goroutine 退出
func (c *KafkaConsumer) Consume(ctx context.Context, topics []string, handler MessageHandler) error {
	if len(topics) == 0 {
		return ErrNoTopics
	}
	if handler == nil {
		return ErrNilHandler
	}

	c.topics = topics
	c.handler = handler
	ctx, c.cancel = context.WithCancel(ctx)

	// 消费循环
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer c.recoverPanic("消费循环")
		for {
			if err := c.consumerGroup.Consume(ctx, c.topics, c); err != nil {
				if ctx.Err() != nil {
					return
				}
				if c.logger != nil {
					c.logger.With(logger.Err(err)).Error("[Messaging] 消费失败")
				}
				time.Sleep(c.reconnectInterval)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// 错误监听
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer c.recoverPanic("错误监听")
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-c.consumerGroup.Errors():
				if !ok {
					return
				}
				if c.logger != nil {
					c.logger.With(logger.Err(err)).Warn("[Messaging] 消费者错误")
				}
			}
		}
	}()

	return nil
}

// CommitMessage 手动提交消息偏移量.
//
// 当需要精确控制提交时机时使用，例如批量处理场景.
// 正常情况下，消息处理成功后会自动提交，无需手动调用.
func (c *KafkaConsumer) CommitMessage(msg *Message) error {
	c.sessionMu.RLock()
	session := c.currentSession
	c.sessionMu.RUnlock()

	if session == nil {
		return ErrNoActiveSession
	}

	session.MarkOffset(msg.Topic, msg.Partition, msg.Offset+1, "")
	session.Commit()
	return nil
}

// Close 关闭消费者.
//
// 停止消费并等待所有 goroutine 退出，释放资源.
// 重复调用是安全的.
func (c *KafkaConsumer) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()

	if c.consumerGroup != nil {
		return c.consumerGroup.Close()
	}
	return nil
}

// Setup 实现 sarama.ConsumerGroupHandler 接口.
// 在消费开始前调用.
func (c *KafkaConsumer) Setup(session sarama.ConsumerGroupSession) error {
	c.sessionMu.Lock()
	c.currentSession = session
	c.sessionMu.Unlock()
	return nil
}

// Cleanup 实现 sarama.ConsumerGroupHandler 接口.
// 在消费结束后调用.
func (c *KafkaConsumer) Cleanup(session sarama.ConsumerGroupSession) error {
	session.Commit()
	c.sessionMu.Lock()
	c.currentSession = nil
	c.sessionMu.Unlock()
	return nil
}

// ConsumeClaim 实现 sarama.ConsumerGroupHandler 接口.
// 处理分区消息.
func (c *KafkaConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			c.processMessage(session, msg)

		case <-session.Context().Done():
			return nil
		}
	}
}

// processMessage 处理单条消息.
// 提取为独立方法以确保 defer span.End() 在每条消息处理完成后执行.
func (c *KafkaConsumer) processMessage(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) {
	startTime := time.Now()

	// 构建消息
	message := &Message{
		Topic:     msg.Topic,
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   make(map[string]string, len(msg.Headers)),
		Partition: msg.Partition,
		Offset:    msg.Offset,
		Timestamp: msg.Timestamp,
	}
	for _, header := range msg.Headers {
		message.Headers[string(header.Key)] = string(header.Value)
	}

	// Tracing: 从 Headers 提取追踪上下文并开始 span
	ctx := session.Context()
	var span trace.Span
	if c.tracer != nil {
		ctx = c.tracer.extractContext(ctx, message.Headers)
		ctx, span = c.tracer.startConsumerSpan(ctx, msg.Topic, msg.Partition, msg.Offset)
		defer span.End() // 现在会在 processMessage 返回时正确关闭
	}

	// 处理消息（带重试）
	var lastErr error
	maxAttempts := c.maxRetries + 1
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := c.handler(message); err != nil {
			lastErr = err
			// Metrics: 记录重试
			if c.metrics != nil && attempt < maxAttempts {
				c.metrics.RecordRetry(msg.Topic)
			}
			if attempt < maxAttempts {
				// 指数退避：interval * 2^(attempt-1)
				backoff := c.retryInterval * time.Duration(1<<(attempt-1))
				if c.logger != nil {
					c.logger.With(
						logger.Duration("backoff", backoff),
						logger.Int("attempt", attempt),
						logger.Int("maxRetries", c.maxRetries),
						logger.String("topic", msg.Topic),
						logger.Int64("offset", msg.Offset),
						logger.Err(err),
					).Warn("[Messaging] 消息处理失败，即将重试")
				}
				time.Sleep(backoff)
				continue
			}
		} else {
			lastErr = nil
			break
		}
	}

	// 处理结果
	if lastErr != nil {
		// Tracing: 记录错误
		if c.tracer != nil {
			c.tracer.setError(span, lastErr)
		}
		// Metrics: 记录消费错误
		if c.metrics != nil {
			c.metrics.RecordConsumeError(msg.Topic, c.groupID)
		}
		if c.logger != nil {
			c.logger.With(
				logger.String("topic", msg.Topic),
				logger.Int64("offset", msg.Offset),
				logger.Err(lastErr),
			).Error("[Messaging] 消息处理失败，重试耗尽")
		}
		// 发送到死信队列
		if c.dlqProducer != nil && c.deadLetterTopic != "" {
			c.sendToDeadLetterQueue(ctx, message, lastErr)
			// Metrics: 记录 DLQ
			if c.metrics != nil {
				c.metrics.RecordDLQ(msg.Topic)
			}
		}
	} else {
		// Metrics: 记录成功消费
		if c.metrics != nil {
			c.metrics.RecordConsume(msg.Topic, c.groupID, time.Since(startTime))
		}
	}
	// 无论成功或失败（发送到DLQ后），都标记消息已处理
	session.MarkMessage(msg, "")
}

// recoverPanic 恢复 goroutine panic 并记录日志.
func (c *KafkaConsumer) recoverPanic(goroutineName string) {
	if r := recover(); r != nil {
		if c.logger != nil {
			c.logger.With(
				logger.String("goroutine", goroutineName),
				logger.Any("panic", r),
			).Error("[Messaging] goroutine panic")
		}
	}
}

// sendToDeadLetterQueue 发送消息到死信队列.
func (c *KafkaConsumer) sendToDeadLetterQueue(ctx context.Context, msg *Message, err error) {
	dlqMsg := &Message{
		Topic: c.deadLetterTopic,
		Key:   msg.Key,
		Value: msg.Value,
		Headers: map[string]string{
			"x-original-topic":     msg.Topic,
			"x-original-partition": strconv.FormatInt(int64(msg.Partition), 10),
			"x-original-offset":    strconv.FormatInt(msg.Offset, 10),
			"x-error-message":      err.Error(),
			"x-consumer-group":     c.groupID,
		},
	}
	// 保留原始 headers
	for k, v := range msg.Headers {
		if _, exists := dlqMsg.Headers[k]; !exists {
			dlqMsg.Headers[k] = v
		}
	}

	if _, sendErr := c.dlqProducer.SendMessage(ctx, dlqMsg); sendErr != nil {
		if c.logger != nil {
			c.logger.With(
				logger.String("topic", msg.Topic),
				logger.Int64("offset", msg.Offset),
				logger.Err(sendErr),
			).Error("[Messaging] 发送死信队列失败")
		}
	} else if c.logger != nil {
		c.logger.With(
			logger.String("originalTopic", msg.Topic),
			logger.String("dlqTopic", c.deadLetterTopic),
		).Warn("[Messaging] 消息已发送到死信队列")
	}
}
