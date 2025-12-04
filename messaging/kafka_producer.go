package messaging

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel/trace"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// ProducerOption 生产者配置选项.
//
// 使用函数选项模式配置生产者.
type ProducerOption func(*producerOptions)

type producerOptions struct {
	logger logger.Logger
}

// WithProducerLogger 设置日志记录器.
//
// 用于记录生产者启动、错误等日志.
func WithProducerLogger(log logger.Logger) ProducerOption {
	return func(o *producerOptions) {
		o.logger = log
	}
}

// KafkaProducer Kafka 生产者.
//
// 使用同步发送模式，保证消息可靠投递.
// 内置最佳实践配置：
//   - Idempotent: true (幂等性，保证消息不重复)
//   - RequiredAcks: WaitForAll (等待所有副本确认)
//   - Retry.Max: 3 (最多重试3次)
//   - Compression: Snappy (使用Snappy压缩)
//
// 示例:
//
//	producer, err := NewKafkaProducer(
//	    []string{"localhost:9092"},
//	    WithProducerLogger(log),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer producer.Close()
//
//	msg, err := producer.SendMessage(ctx, &Message{
//	    Topic: "orders",
//	    Value: []byte(`{"id":"123"}`),
//	})
type KafkaProducer struct {
	producer sarama.SyncProducer
	closed   bool
	mu       sync.RWMutex
	logger   logger.Logger
	metrics  *messagingMetrics
	tracer   *messagingTracer
}

// NewKafkaProducer 创建 Kafka 生产者.
//
// 参数:
//   - brokers: Kafka 服务器地址列表
//   - opts: 可选配置项
//
// 返回创建的生产者实例，使用完毕后需调用 Close 关闭.
func NewKafkaProducer(brokers []string, opts ...ProducerOption) (*KafkaProducer, error) {
	options := &producerOptions{}
	for _, opt := range opts {
		opt(options)
	}

	config := sarama.NewConfig()
	config.Version = sarama.V3_8_0_0
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.Idempotent = true
	config.Net.MaxOpenRequests = 1

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, errors.Join(ErrCreateProducer, err)
	}

	p := &KafkaProducer{
		producer: producer,
		logger:   options.logger,
	}

	if p.logger != nil {
		p.logger.Debugf("[Messaging] Kafka生产者启动: brokers=%v", brokers)
	}

	return p, nil
}

// SendMessage 发送消息.
//
// 同步发送消息并等待确认，返回包含分区和偏移量信息的消息.
//
// 参数:
//   - ctx: 上下文，用于取消操作
//   - msg: 要发送的消息，Topic 字段必填
//
// 返回:
//   - *Message: 发送成功后的消息，包含 Partition 和 Offset
//   - error: 发送失败时返回错误
func (p *KafkaProducer) SendMessage(ctx context.Context, msg *Message) (*Message, error) {
	startTime := time.Now()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()
	if closed {
		return nil, ErrProducerClosed
	}

	if msg == nil {
		return nil, ErrNilMessage
	}
	if msg.Topic == "" {
		return nil, ErrEmptyTopic
	}

	// Tracing: 开始 span
	var span trace.Span
	if p.tracer != nil {
		ctx, span = p.tracer.startProducerSpan(ctx, msg.Topic)
		defer span.End()
	}

	// 构建 sarama 消息
	saramaMsg := p.buildSaramaMessage(ctx, msg)

	partition, offset, err := p.producer.SendMessage(saramaMsg)
	if err != nil {
		// Tracing: 记录错误
		if p.tracer != nil {
			p.tracer.setError(span, err)
		}
		// Metrics: 记录错误
		if p.metrics != nil {
			p.metrics.RecordSendError(msg.Topic)
		}
		return nil, errors.Join(ErrSendMessage, err)
	}

	// Metrics: 记录成功发送
	if p.metrics != nil {
		p.metrics.RecordSend(msg.Topic, time.Since(startTime))
	}

	return &Message{
		Topic:     msg.Topic,
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   msg.Headers,
		Partition: partition,
		Offset:    offset,
		Timestamp: time.Now(),
	}, nil
}

// SendBatch 批量发送消息.
//
// 一次性发送多条消息，提高吞吐量.
// 注意：批量发送是原子操作，要么全部成功，要么全部失败.
//
// 参数:
//   - ctx: 上下文，用于取消操作
//   - msgs: 要发送的消息列表
//
// 返回:
//   - []*Message: 发送成功后的消息列表，包含 Partition 和 Offset
//   - error: 发送失败时返回错误
func (p *KafkaProducer) SendBatch(ctx context.Context, msgs []*Message) ([]*Message, error) {
	if len(msgs) == 0 {
		return nil, nil
	}

	startTime := time.Now()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()
	if closed {
		return nil, ErrProducerClosed
	}

	// 构建 sarama 消息列表
	saramaMsgs := make([]*sarama.ProducerMessage, 0, len(msgs))
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		if msg.Topic == "" {
			return nil, ErrEmptyTopic
		}
		saramaMsgs = append(saramaMsgs, p.buildSaramaMessage(ctx, msg))
	}

	// 批量发送
	if err := p.producer.SendMessages(saramaMsgs); err != nil {
		// Metrics: 记录错误
		if p.metrics != nil {
			for _, msg := range msgs {
				if msg != nil {
					p.metrics.RecordSendError(msg.Topic)
				}
			}
		}
		return nil, errors.Join(ErrBatchSend, err)
	}

	// 构建返回结果
	results := make([]*Message, 0, len(saramaMsgs))
	for i, saramaMsg := range saramaMsgs {
		results = append(results, &Message{
			Topic:     msgs[i].Topic,
			Key:       msgs[i].Key,
			Value:     msgs[i].Value,
			Headers:   msgs[i].Headers,
			Partition: saramaMsg.Partition,
			Offset:    saramaMsg.Offset,
			Timestamp: time.Now(),
		})

		// Metrics: 记录成功发送
		if p.metrics != nil {
			p.metrics.RecordSend(msgs[i].Topic, time.Since(startTime)/time.Duration(len(msgs)))
		}
	}

	return results, nil
}


// buildSaramaMessage 构建 sarama 消息.
func (p *KafkaProducer) buildSaramaMessage(ctx context.Context, msg *Message) *sarama.ProducerMessage {
	headers := msg.Headers
	if p.tracer != nil {
		headers = p.tracer.injectHeaders(ctx, headers)
	}

	saramaMsg := &sarama.ProducerMessage{
		Topic:     msg.Topic,
		Key:       sarama.ByteEncoder(msg.Key),
		Value:     sarama.ByteEncoder(msg.Value),
		Timestamp: time.Now(),
	}
	for k, v := range headers {
		saramaMsg.Headers = append(saramaMsg.Headers, sarama.RecordHeader{Key: []byte(k), Value: []byte(v)})
	}
	return saramaMsg
}

// Close 关闭生产者.
//
// 关闭与 Kafka 的连接，释放资源.
// 关闭后不能再发送消息，重复调用是安全的.
func (p *KafkaProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	if p.producer != nil {
		return p.producer.Close()
	}
	return nil
}
