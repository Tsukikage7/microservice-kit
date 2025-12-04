package messaging

import (
	"context"
	"errors"
	"sync"

	"github.com/IBM/sarama"

	"github.com/Tsukikage7/microservice-kit/logger"
	"github.com/Tsukikage7/microservice-kit/metrics"
)

// Client 消息队列客户端.
//
// 统一管理生产者和消费者的生命周期，提供：
//   - 统一配置入口
//   - 共享连接管理
//   - 优雅关闭
//   - 健康检查
//
// 示例:
//
//	client, err := messaging.NewClient(
//	    messaging.WithBrokers([]string{"localhost:9092"}),
//	    messaging.WithLogger(log),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// 创建生产者
//	producer := client.Producer()
//	producer.SendMessage(ctx, msg)
//
//	// 创建消费者
//	client.Consumer("order-service").Consume(ctx, topics, handler)
type Client struct {
	brokers   []string
	logger    logger.Logger
	clientID  string
	mqType    string
	producers []*KafkaProducer
	consumers []*KafkaConsumer
	mu        sync.Mutex
	closed    bool

	// Metrics 相关（使用 metrics 包的 PrometheusCollector）
	metrics *messagingMetrics

	// Tracing 相关（使用 trace 包初始化的全局 TracerProvider）
	tracer *messagingTracer

	// sarama 客户端（用于健康检查）
	saramaClient sarama.Client
}

// ClientOption 客户端配置选项.
type ClientOption func(*Client)

// WithBrokers 设置服务器地址.
func WithBrokers(brokers []string) ClientOption {
	return func(c *Client) {
		c.brokers = brokers
	}
}

// WithLogger 设置日志记录器.
func WithLogger(log logger.Logger) ClientOption {
	return func(c *Client) {
		c.logger = log
	}
}

// WithClientID 设置客户端ID.
//
// 用于标识客户端，便于在 Kafka 中追踪.
func WithClientID(clientID string) ClientOption {
	return func(c *Client) {
		c.clientID = clientID
	}
}

// WithType 设置消息队列类型.
//
// 支持: kafka（默认）
func WithType(mqType string) ClientOption {
	return func(c *Client) {
		c.mqType = mqType
	}
}

// WithMetrics 启用指标监控.
//
// 直接使用 metrics 包的 PrometheusCollector，与 HTTP/gRPC 指标统一管理.
//
// 示例:
//
//	collector := metrics.MustNew(&metrics.Config{Namespace: "myapp"})
//	client, _ := messaging.NewClient(
//	    messaging.WithBrokers(brokers),
//	    messaging.WithMetrics(collector),
//	)
func WithMetrics(collector *metrics.PrometheusCollector) ClientOption {
	return func(c *Client) {
		c.metrics = newMessagingMetrics(collector)
	}
}

// WithTracing 启用链路追踪.
//
// 使用 trace 包初始化的全局 TracerProvider，与 HTTP/gRPC 追踪统一管理.
// 需要先调用 trace.NewTracer 初始化全局 TracerProvider.
//
// 示例:
//
//	// 初始化全局 TracerProvider
//	tp, _ := trace.NewTracer(cfg, "my-service", "1.0.0")
//	defer tp.Shutdown(context.Background())
//
//	client, _ := messaging.NewClient(
//	    messaging.WithBrokers(brokers),
//	    messaging.WithTracing("my-service"),
//	)
func WithTracing(serviceName string) ClientOption {
	return func(c *Client) {
		c.tracer = newMessagingTracer(serviceName)
	}
}


// NewClient 创建消息队列客户端.
//
// 必须设置 WithBrokers 选项.
func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		mqType: "kafka",
	}

	for _, opt := range opts {
		opt(c)
	}

	if len(c.brokers) == 0 {
		return nil, ErrNoBrokers
	}

	// 创建 sarama 客户端用于健康检查
	config := sarama.NewConfig()
	config.Version = sarama.V3_8_0_0
	if c.clientID != "" {
		config.ClientID = c.clientID
	}

	client, err := sarama.NewClient(c.brokers, config)
	if err != nil {
		return nil, errors.Join(ErrCreateClient, err)
	}
	c.saramaClient = client

	if c.logger != nil {
		c.logger.Debugf("[Messaging] 客户端已创建: brokers=%v, clientID=%s", c.brokers, c.clientID)
	}

	return c, nil
}

// Producer 创建生产者.
//
// 生产者由 Client 管理生命周期，调用 Client.Close() 时自动关闭.
func (c *Client) Producer(opts ...ProducerOption) (*KafkaProducer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	// 添加默认 logger
	if c.logger != nil {
		opts = append([]ProducerOption{WithProducerLogger(c.logger)}, opts...)
	}

	producer, err := NewKafkaProducer(c.brokers, opts...)
	if err != nil {
		return nil, err
	}

	// 设置 metrics
	if c.metrics != nil {
		producer.metrics = c.metrics
	}

	// 设置 trace
	if c.tracer != nil {
		producer.tracer = c.tracer
	}

	c.producers = append(c.producers, producer)
	return producer, nil
}

// Consumer 创建消费者.
//
// 消费者由 Client 管理生命周期，调用 Client.Close() 时自动关闭.
func (c *Client) Consumer(groupID string, opts ...ConsumerOption) (*KafkaConsumer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	// 添加默认 logger
	if c.logger != nil {
		opts = append([]ConsumerOption{WithConsumerLogger(c.logger)}, opts...)
	}

	consumer, err := NewKafkaConsumer(c.brokers, groupID, opts...)
	if err != nil {
		return nil, err
	}

	// 设置 metrics
	if c.metrics != nil {
		consumer.metrics = c.metrics
	}

	// 设置 trace
	if c.tracer != nil {
		consumer.tracer = c.tracer
	}

	c.consumers = append(c.consumers, consumer)
	return consumer, nil
}

// HealthCheck 健康检查.
//
// 检查与 Kafka 集群的连接是否正常.
// 可用于 K8s liveness/readiness probe.
func (c *Client) HealthCheck(ctx context.Context) error {
	c.mu.Lock()
	closed := c.closed
	client := c.saramaClient
	c.mu.Unlock()

	if closed {
		return ErrClientClosed
	}

	if client == nil {
		return ErrClientClosed
	}

	// 检查是否能获取 broker 列表
	brokers := client.Brokers()
	if len(brokers) == 0 {
		return ErrNoBrokersAvailable
	}

	// 检查至少有一个 broker 可连接
	for _, broker := range brokers {
		connected, _ := broker.Connected()
		if connected {
			return nil
		}
	}

	// 尝试刷新元数据
	if err := client.RefreshMetadata(); err != nil {
		return errors.Join(ErrHealthCheck, err)
	}

	return nil
}

// Close 关闭客户端.
//
// 关闭所有生产者和消费者，释放资源.
func (c *Client) Close() error {
	return c.Shutdown(context.Background())
}

// Shutdown 优雅关闭客户端.
//
// 等待进行中的消息处理完成（受 ctx 超时限制）.
// 建议配合 context.WithTimeout 使用.
//
// 示例:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	client.Shutdown(ctx)
func (c *Client) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true

	producers := c.producers
	consumers := c.consumers
	saramaClient := c.saramaClient
	c.mu.Unlock()

	if c.logger != nil {
		c.logger.Debugf("[Messaging] 开始优雅关闭，等待 %d 个生产者和 %d 个消费者...",
			len(producers), len(consumers))
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(producers)+len(consumers)+1)

	// 先关闭消费者（停止消费新消息）
	for _, consumer := range consumers {
		wg.Add(1)
		go func(cons *KafkaConsumer) {
			defer wg.Done()
			if err := cons.Close(); err != nil {
				errCh <- err
			}
		}(consumer)
	}

	// 再关闭生产者（等待发送完成）
	for _, producer := range producers {
		wg.Add(1)
		go func(prod *KafkaProducer) {
			defer wg.Done()
			if err := prod.Close(); err != nil {
				errCh <- err
			}
		}(producer)
	}

	// 等待所有关闭完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 所有关闭完成
	case <-ctx.Done():
		if c.logger != nil {
			c.logger.Warnf("[Messaging] 优雅关闭超时，强制关闭")
		}
	}

	// 关闭 sarama 客户端
	if saramaClient != nil {
		if err := saramaClient.Close(); err != nil {
			errCh <- err
		}
	}

	close(errCh)

	// 收集所有错误
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if c.logger != nil {
		c.logger.Debugf("[Messaging] 客户端已关闭")
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Brokers 返回当前配置的 broker 地址列表的副本.
func (c *Client) Brokers() []string {
	result := make([]string, len(c.brokers))
	copy(result, c.brokers)
	return result
}

// Stats 返回客户端统计信息.
func (c *Client) Stats() ClientStats {
	c.mu.Lock()
	defer c.mu.Unlock()

	return ClientStats{
		ProducerCount: len(c.producers),
		ConsumerCount: len(c.consumers),
		Closed:        c.closed,
	}
}

// ClientStats 客户端统计信息.
type ClientStats struct {
	ProducerCount int
	ConsumerCount int
	Closed        bool
}
