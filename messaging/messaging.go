// Package messaging 提供消息队列客户端.
//
// 支持多种消息队列实现，通过配置切换。目前支持 Kafka.
//
// 示例:
//
//	// 创建生产者
//	producer, _ := messaging.NewProducer(cfg, messaging.WithIdempotent(), messaging.WithProducerLogger(log))
//	defer producer.Close()
//
//	// 发送消息
//	msg, _ := producer.SendMessage(ctx, &messaging.Message{
//	    Topic: "orders",
//	    Key:   []byte("order-123"),
//	    Value: []byte(`{"id": "123"}`),
//	})
//
//	// 创建消费者
//	consumer, _ := messaging.NewConsumer(cfg, "order-service", messaging.WithConsumerLogger(log))
//	defer consumer.Close()
//
//	// 消费消息
//	consumer.Consume(ctx, []string{"orders"}, func(msg *messaging.Message) error {
//	    fmt.Printf("收到消息: %s\n", msg.Value)
//	    return nil
//	})
package messaging

import (
	"context"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// MessageHandler 消息处理函数.
type MessageHandler func(*Message) error

// Producer 生产者接口.
type Producer interface {
	// SendMessage 发送单条消息，返回包含分区和偏移量的消息.
	SendMessage(ctx context.Context, msg *Message) (*Message, error)
	// SendBatch 批量发送消息，返回包含分区和偏移量的消息列表.
	SendBatch(ctx context.Context, msgs []*Message) ([]*Message, error)
	// Close 关闭生产者.
	Close() error
}

// Consumer 消费者接口.
type Consumer interface {
	// Consume 开始消费消息，handler 处理每条消息.
	Consume(ctx context.Context, topics []string, handler MessageHandler) error
	// CommitMessage 手动提交消息偏移量.
	CommitMessage(msg *Message) error
	// Close 关闭消费者.
	Close() error
}

// NewProducer 根据配置创建生产者.
func NewProducer(cfg *Config, opts ...ProducerOption) (Producer, error) {
	// 提取 logger
	var log logger.Logger
	for _, opt := range opts {
		po := &producerOptions{}
		opt(po)
		if po.logger != nil {
			log = po.logger
		}
	}

	switch cfg.Type {
	case "kafka", "":
		return NewKafkaProducer(cfg.Brokers, opts...)
	case "rabbitmq":
		return newRabbitMQProducer(cfg, log)
	default:
		return nil, ErrUnsupportedType
	}
}

// NewConsumer 根据配置创建消费者.
func NewConsumer(cfg *Config, groupID string, opts ...ConsumerOption) (Consumer, error) {
	// 提取 logger
	var log logger.Logger
	for _, opt := range opts {
		co := &consumerOptions{}
		opt(co)
		if co.logger != nil {
			log = co.logger
		}
	}

	switch cfg.Type {
	case "kafka", "":
		return NewKafkaConsumer(cfg.Brokers, groupID, opts...)
	case "rabbitmq":
		return newRabbitMQConsumer(cfg, groupID, log)
	default:
		return nil, ErrUnsupportedType
	}
}
