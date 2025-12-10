package messaging

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// rabbitMQConsumer RabbitMQ 消费者.
type rabbitMQConsumer struct {
	conn    *rabbitMQConnection
	channel *amqp.Channel
	mu      sync.RWMutex
	closed  atomic.Bool
	groupID string

	consuming  atomic.Bool
	cancelFunc context.CancelFunc
	delivery   *amqp.Delivery

	exchange      string
	exchangeType  exchangeType
	queueDurable  bool
	queueExcl     bool
	autoDelete    bool
	autoAck       bool
	prefetchCount int
	prefetchSize  int
	logger        logger.Logger
}

func newRabbitMQConsumer(cfg *Config, groupID string, log logger.Logger) (*rabbitMQConsumer, error) {
	if groupID == "" {
		return nil, ErrEmptyGroupID
	}

	if cfg.URL == "" {
		return nil, ErrNoBrokers
	}

	c := &rabbitMQConsumer{
		groupID:       groupID,
		exchange:      "",
		exchangeType:  exchangeDirect,
		queueDurable:  true,
		autoAck:       false,
		prefetchCount: 10,
		logger:        log,
	}

	// 应用 RabbitMQ 特定配置
	if cfg.RabbitMQ != nil {
		if cfg.RabbitMQ.Exchange != "" {
			c.exchange = cfg.RabbitMQ.Exchange
		}
		if cfg.RabbitMQ.ExchangeType != "" {
			c.exchangeType = exchangeType(cfg.RabbitMQ.ExchangeType)
		}
		c.queueDurable = cfg.RabbitMQ.Durable
		c.autoAck = cfg.RabbitMQ.AutoAck
		if cfg.RabbitMQ.PrefetchCount > 0 {
			c.prefetchCount = cfg.RabbitMQ.PrefetchCount
		}
	}

	// 创建连接
	var connOpts []rabbitMQConnectionOption
	if log != nil {
		connOpts = append(connOpts, withRabbitMQConnectionLogger(log))
	}

	conn, err := newRabbitMQConnection(cfg.URL, connOpts...)
	if err != nil {
		return nil, err
	}
	c.conn = conn

	if err := c.setupChannel(); err != nil {
		conn.Close()
		return nil, err
	}

	return c, nil
}

func (c *rabbitMQConsumer) setupChannel() error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCreateConsumer, err)
	}

	if err := ch.Qos(c.prefetchCount, c.prefetchSize, false); err != nil {
		ch.Close()
		return fmt.Errorf("设置 QoS 失败: %w", err)
	}

	if c.exchange != "" {
		err = ch.ExchangeDeclare(
			c.exchange,
			string(c.exchangeType),
			c.queueDurable,
			c.autoDelete,
			false,
			false,
			nil,
		)
		if err != nil {
			ch.Close()
			return fmt.Errorf("声明交换机失败: %w", err)
		}
	}

	c.mu.Lock()
	c.channel = ch
	c.mu.Unlock()

	return nil
}

func (c *rabbitMQConsumer) Consume(ctx context.Context, topics []string, handler MessageHandler) error {
	if c.closed.Load() {
		return ErrClientClosed
	}

	if len(topics) == 0 {
		return ErrNoTopics
	}

	if handler == nil {
		return ErrNilHandler
	}

	if c.consuming.Swap(true) {
		return fmt.Errorf("消费者已在运行")
	}
	defer c.consuming.Store(false)

	ctx, cancel := context.WithCancel(ctx)
	c.cancelFunc = cancel
	defer cancel()

	queueName, deliveries, err := c.setupQueue(topics)
	if err != nil {
		return err
	}

	c.log("开始消费队列: %s", queueName)

	go c.handleReconnect(ctx, topics, handler)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case delivery, ok := <-deliveries:
			if !ok {
				c.log("消费 channel 关闭，等待重连...")
				time.Sleep(time.Second)
				continue
			}

			msg := c.convertMessage(&delivery)

			c.mu.Lock()
			c.delivery = &delivery
			c.mu.Unlock()

			if err := handler(msg); err != nil {
				c.log("消息处理失败: %v", err)
				if !c.autoAck {
					delivery.Nack(false, true)
				}
				continue
			}

			if !c.autoAck {
				delivery.Ack(false)
			}
		}
	}
}

func (c *rabbitMQConsumer) setupQueue(topics []string) (string, <-chan amqp.Delivery, error) {
	c.mu.RLock()
	ch := c.channel
	c.mu.RUnlock()

	if ch == nil {
		return "", nil, ErrNoBrokersAvailable
	}

	queueName := c.groupID
	if c.exchange == "" && len(topics) > 0 {
		queueName = topics[0]
	}

	queue, err := ch.QueueDeclare(
		queueName,
		c.queueDurable,
		c.autoDelete,
		c.queueExcl,
		false,
		nil,
	)
	if err != nil {
		return "", nil, fmt.Errorf("声明队列失败: %w", err)
	}

	if c.exchange != "" {
		for _, topic := range topics {
			err = ch.QueueBind(
				queue.Name,
				topic,
				c.exchange,
				false,
				nil,
			)
			if err != nil {
				return "", nil, fmt.Errorf("绑定队列失败: %w", err)
			}
		}
	}

	deliveries, err := ch.Consume(
		queue.Name,
		c.groupID,
		c.autoAck,
		c.queueExcl,
		false,
		false,
		nil,
	)
	if err != nil {
		return "", nil, fmt.Errorf("启动消费失败: %w", err)
	}

	return queue.Name, deliveries, nil
}

func (c *rabbitMQConsumer) handleReconnect(ctx context.Context, topics []string, handler MessageHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.conn.ReconnectNotify():
			if c.closed.Load() {
				return
			}

			c.log("检测到重连，重新设置消费者...")

			c.mu.Lock()
			if c.channel != nil {
				c.channel.Close()
			}
			c.mu.Unlock()

			if err := c.setupChannel(); err != nil {
				c.log("重建 channel 失败: %v", err)
				continue
			}

			if _, _, err := c.setupQueue(topics); err != nil {
				c.log("重新设置队列失败: %v", err)
			}
		}
	}
}

func (c *rabbitMQConsumer) convertMessage(delivery *amqp.Delivery) *Message {
	msg := &Message{
		Topic:     delivery.RoutingKey,
		Key:       []byte(delivery.MessageId),
		Value:     delivery.Body,
		Timestamp: delivery.Timestamp,
		Offset:    int64(delivery.DeliveryTag),
	}

	if len(delivery.Headers) > 0 {
		msg.Headers = make(map[string]string)
		for k, v := range delivery.Headers {
			if str, ok := v.(string); ok {
				msg.Headers[k] = str
			}
		}
	}

	return msg
}

func (c *rabbitMQConsumer) CommitMessage(msg *Message) error {
	c.mu.RLock()
	delivery := c.delivery
	c.mu.RUnlock()

	if delivery == nil {
		return ErrNoActiveSession
	}

	return delivery.Ack(false)
}

func (c *rabbitMQConsumer) Close() error {
	if c.closed.Swap(true) {
		return nil
	}

	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.channel != nil {
		c.channel.Close()
	}

	return c.conn.Close()
}

func (c *rabbitMQConsumer) log(format string, args ...any) {
	if c.logger != nil {
		c.logger.Info(fmt.Sprintf(format, args...))
	}
}
