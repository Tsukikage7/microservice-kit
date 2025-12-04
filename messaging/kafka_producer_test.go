package messaging

import (
	"context"
	"errors"
	"testing"
)

func TestProducerOptions(t *testing.T) {
	t.Run("WithProducerLogger", func(t *testing.T) {
		opts := &producerOptions{}
		log := &mockLogger{}
		WithProducerLogger(log)(opts)
		if opts.logger == nil {
			t.Error("expected logger to be set")
		}
	})
}

func TestKafkaProducer_SendMessage_Validation(t *testing.T) {
	// 创建一个模拟的 producer 用于测试验证逻辑
	p := &KafkaProducer{
		producer: nil, // 不需要真实连接来测试验证逻辑
		closed:   false,
	}

	ctx := context.Background()

	t.Run("nil message", func(t *testing.T) {
		_, err := p.SendMessage(ctx, nil)
		if !errors.Is(err, ErrNilMessage) {
			t.Errorf("expected ErrNilMessage, got %v", err)
		}
	})

	t.Run("empty topic", func(t *testing.T) {
		_, err := p.SendMessage(ctx, &Message{Topic: ""})
		if !errors.Is(err, ErrEmptyTopic) {
			t.Errorf("expected ErrEmptyTopic, got %v", err)
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := p.SendMessage(canceledCtx, &Message{Topic: "test"})
		if err == nil {
			t.Error("expected error for canceled context")
		}
	})

	t.Run("producer closed", func(t *testing.T) {
		closedProducer := &KafkaProducer{closed: true}
		_, err := closedProducer.SendMessage(ctx, &Message{Topic: "test"})
		if !errors.Is(err, ErrProducerClosed) {
			t.Errorf("expected ErrProducerClosed, got %v", err)
		}
	})
}

func TestKafkaProducer_Close(t *testing.T) {
	t.Run("close idempotent", func(t *testing.T) {
		p := &KafkaProducer{
			producer: nil,
			closed:   false,
		}

		// 第一次关闭
		err := p.Close()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !p.closed {
			t.Error("expected producer to be closed")
		}

		// 重复关闭应该是安全的
		err = p.Close()
		if err != nil {
			t.Errorf("unexpected error on second close: %v", err)
		}
	})
}

