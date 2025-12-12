package activity

import (
	"context"
	"encoding/json"

	"github.com/Tsukikage7/microservice-kit/messaging"
)

// KafkaProducer Kafka 消息生产者.
type KafkaProducer struct {
	producer messaging.Producer
}

// NewKafkaProducer 创建 Kafka 生产者.
func NewKafkaProducer(producer messaging.Producer) *KafkaProducer {
	return &KafkaProducer{producer: producer}
}

// Publish 发布活跃事件到 Kafka.
func (p *KafkaProducer) Publish(ctx context.Context, topic string, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := &messaging.Message{
		Topic: topic,
		Key:   []byte(event.UserID), // 使用 user_id 作为分区键，保证同一用户的消息有序
		Value: data,
		Headers: map[string]string{
			"event_type": string(event.EventType),
			"user_id":    event.UserID,
		},
	}

	_, err = p.producer.SendMessage(ctx, msg)
	return err
}

// 确保 KafkaProducer 实现了 Producer 接口.
var _ Producer = (*KafkaProducer)(nil)
