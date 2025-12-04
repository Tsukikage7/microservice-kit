package messaging

import (
	"context"
	"errors"
	"testing"

	"github.com/Tsukikage7/microservice-kit/metrics"
)

func TestClientOptions(t *testing.T) {
	t.Run("WithBrokers", func(t *testing.T) {
		c := &Client{}
		WithBrokers([]string{"localhost:9092"})(c)
		if len(c.brokers) != 1 || c.brokers[0] != "localhost:9092" {
			t.Error("expected brokers to be set")
		}
	})

	t.Run("WithLogger", func(t *testing.T) {
		c := &Client{}
		log := &mockLogger{}
		WithLogger(log)(c)
		if c.logger == nil {
			t.Error("expected logger to be set")
		}
	})

	t.Run("WithClientID", func(t *testing.T) {
		c := &Client{}
		WithClientID("test-client")(c)
		if c.clientID != "test-client" {
			t.Errorf("expected clientID 'test-client', got '%s'", c.clientID)
		}
	})

	t.Run("WithType", func(t *testing.T) {
		c := &Client{}
		WithType("kafka")(c)
		if c.mqType != "kafka" {
			t.Errorf("expected type 'kafka', got '%s'", c.mqType)
		}
	})

	t.Run("WithMetrics", func(t *testing.T) {
		c := &Client{}
		collector := metrics.MustNew(&metrics.Config{Namespace: "test"})
		WithMetrics(collector)(c)
		if c.metrics == nil {
			t.Error("expected metrics to be set")
		}
	})

	t.Run("WithTracing", func(t *testing.T) {
		c := &Client{}
		WithTracing("test-service")(c)
		if c.tracer == nil {
			t.Error("expected tracer to be set")
		}
	})

}

func TestNewClient_Validation(t *testing.T) {
	t.Run("no brokers", func(t *testing.T) {
		_, err := NewClient()
		if !errors.Is(err, ErrNoBrokers) {
			t.Errorf("expected ErrNoBrokers, got %v", err)
		}
	})
}

func TestClient_Stats(t *testing.T) {
	c := &Client{
		brokers:   []string{"localhost:9092"},
		producers: []*KafkaProducer{{}, {}},
		consumers: []*KafkaConsumer{{}},
		closed:    false,
	}

	stats := c.Stats()

	if stats.ProducerCount != 2 {
		t.Errorf("expected 2 producers, got %d", stats.ProducerCount)
	}
	if stats.ConsumerCount != 1 {
		t.Errorf("expected 1 consumer, got %d", stats.ConsumerCount)
	}
	if stats.Closed {
		t.Error("expected not closed")
	}
}

func TestClient_Brokers(t *testing.T) {
	brokers := []string{"localhost:9092", "localhost:9093"}
	c := &Client{brokers: brokers}

	result := c.Brokers()
	if len(result) != 2 {
		t.Errorf("expected 2 brokers, got %d", len(result))
	}

	// 验证返回的是副本而非原始切片
	result[0] = "modified"
	if c.brokers[0] == "modified" {
		t.Error("Brokers() should return a copy, not the original slice")
	}
}

func TestClient_Producer_Closed(t *testing.T) {
	c := &Client{
		brokers: []string{"localhost:9092"},
		closed:  true,
	}

	_, err := c.Producer()
	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed, got %v", err)
	}
}

func TestClient_Consumer_Closed(t *testing.T) {
	c := &Client{
		brokers: []string{"localhost:9092"},
		closed:  true,
	}

	_, err := c.Consumer("test-group")
	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed, got %v", err)
	}
}

func TestClient_HealthCheck_Closed(t *testing.T) {
	c := &Client{
		brokers: []string{"localhost:9092"},
		closed:  true,
	}

	err := c.HealthCheck(context.Background())
	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed, got %v", err)
	}
}

func TestClient_Shutdown_Idempotent(t *testing.T) {
	c := &Client{
		brokers: []string{"localhost:9092"},
		closed:  false,
	}

	// 第一次关闭
	err := c.Shutdown(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !c.closed {
		t.Error("expected client to be closed")
	}

	// 重复关闭应该是安全的
	err = c.Shutdown(context.Background())
	if err != nil {
		t.Errorf("unexpected error on second shutdown: %v", err)
	}
}
