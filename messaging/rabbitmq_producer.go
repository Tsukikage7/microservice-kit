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

// exchangeType 交换机类型.
type exchangeType string

const (
	exchangeDirect  exchangeType = "direct"
	exchangeFanout  exchangeType = "fanout"
	exchangeTopic   exchangeType = "topic"
	exchangeHeaders exchangeType = "headers"
)

// rabbitMQProducer RabbitMQ 生产者.
type rabbitMQProducer struct {
	conn     *rabbitMQConnection
	channel  *amqp.Channel
	mu       sync.RWMutex
	closed   atomic.Bool
	confirms chan amqp.Confirmation

	exchange     string
	exchangeType exchangeType
	mandatory    bool
	immediate    bool
	durable      bool
	autoDelete   bool
	confirm      bool
	logger       logger.Logger
}

func newRabbitMQProducer(cfg *Config, log logger.Logger) (*rabbitMQProducer, error) {
	if cfg.URL == "" {
		return nil, ErrNoBrokers
	}

	p := &rabbitMQProducer{
		exchange:     "",
		exchangeType: exchangeDirect,
		durable:      true,
		confirm:      true,
		logger:       log,
	}

	// 应用 RabbitMQ 特定配置
	if cfg.RabbitMQ != nil {
		if cfg.RabbitMQ.Exchange != "" {
			p.exchange = cfg.RabbitMQ.Exchange
		}
		if cfg.RabbitMQ.ExchangeType != "" {
			p.exchangeType = exchangeType(cfg.RabbitMQ.ExchangeType)
		}
		p.durable = cfg.RabbitMQ.Durable
		p.confirm = cfg.RabbitMQ.Confirm
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
	p.conn = conn

	if err := p.setupChannel(); err != nil {
		conn.Close()
		return nil, err
	}

	go p.handleReconnect()

	return p, nil
}

func (p *rabbitMQProducer) setupChannel() error {
	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCreateProducer, err)
	}

	if p.exchange != "" {
		err = ch.ExchangeDeclare(
			p.exchange,
			string(p.exchangeType),
			p.durable,
			p.autoDelete,
			false,
			false,
			nil,
		)
		if err != nil {
			ch.Close()
			return fmt.Errorf("声明交换机失败: %w", err)
		}
	}

	if p.confirm {
		if err := ch.Confirm(false); err != nil {
			ch.Close()
			return fmt.Errorf("启用发布确认失败: %w", err)
		}
		p.confirms = ch.NotifyPublish(make(chan amqp.Confirmation, 100))
	}

	p.mu.Lock()
	p.channel = ch
	p.mu.Unlock()

	return nil
}

func (p *rabbitMQProducer) handleReconnect() {
	for range p.conn.ReconnectNotify() {
		if p.closed.Load() {
			return
		}

		p.log("检测到重连，重新创建 channel...")

		p.mu.Lock()
		if p.channel != nil {
			p.channel.Close()
		}
		p.mu.Unlock()

		if err := p.setupChannel(); err != nil {
			p.log("重建 channel 失败: %v", err)
		} else {
			p.log("channel 重建成功")
		}
	}
}

func (p *rabbitMQProducer) SendMessage(ctx context.Context, msg *Message) (*Message, error) {
	if p.closed.Load() {
		return nil, ErrProducerClosed
	}

	if msg == nil {
		return nil, ErrNilMessage
	}

	if msg.Topic == "" {
		return nil, ErrEmptyTopic
	}

	p.mu.RLock()
	ch := p.channel
	p.mu.RUnlock()

	if ch == nil {
		return nil, ErrNoBrokersAvailable
	}

	publishing := amqp.Publishing{
		ContentType:  "application/json",
		Body:         msg.Value,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		MessageId:    string(msg.Key),
	}

	if len(msg.Headers) > 0 {
		publishing.Headers = make(amqp.Table)
		for k, v := range msg.Headers {
			publishing.Headers[k] = v
		}
	}

	err := ch.PublishWithContext(
		ctx,
		p.exchange,
		msg.Topic,
		p.mandatory,
		p.immediate,
		publishing,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSendMessage, err)
	}

	if p.confirm && p.confirms != nil {
		select {
		case confirm := <-p.confirms:
			if !confirm.Ack {
				return nil, fmt.Errorf("%w: 消息被拒绝", ErrSendMessage)
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	msg.Timestamp = publishing.Timestamp
	return msg, nil
}

func (p *rabbitMQProducer) SendBatch(ctx context.Context, msgs []*Message) ([]*Message, error) {
	if p.closed.Load() {
		return nil, ErrProducerClosed
	}

	results := make([]*Message, 0, len(msgs))
	var errs []error

	for _, msg := range msgs {
		result, err := p.SendMessage(ctx, msg)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		results = append(results, result)
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("%w: %d/%d 消息发送失败", ErrBatchSend, len(errs), len(msgs))
	}

	return results, nil
}

func (p *rabbitMQProducer) Close() error {
	if p.closed.Swap(true) {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil {
		p.channel.Close()
	}

	return p.conn.Close()
}

func (p *rabbitMQProducer) log(format string, args ...any) {
	if p.logger != nil {
		p.logger.Info(fmt.Sprintf(format, args...))
	}
}
