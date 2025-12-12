package messaging

import (
	"testing"
	"time"

	"github.com/Tsukikage7/microservice-kit/observability/metrics"
)

func TestMessagingMetrics_RecordSend(t *testing.T) {
	// 创建 Prometheus collector（用于测试）
	collector := metrics.MustNewMetrics(&metrics.Config{
		Namespace: "test",
	})

	m := newMessagingMetrics(collector)

	// 验证不会 panic
	m.RecordSend("test-topic", 100*time.Millisecond)
	m.RecordSend("test-topic", 200*time.Millisecond)
	m.RecordSend("other-topic", 50*time.Millisecond)
}

func TestMessagingMetrics_RecordConsume(t *testing.T) {
	collector := metrics.MustNewMetrics(&metrics.Config{
		Namespace: "test",
	})

	m := newMessagingMetrics(collector)

	// 验证不会 panic
	m.RecordConsume("test-topic", "group1", 100*time.Millisecond)
	m.RecordConsume("test-topic", "group1", 200*time.Millisecond)
	m.RecordConsume("test-topic", "group2", 150*time.Millisecond)
}

func TestMessagingMetrics_RecordErrors(t *testing.T) {
	collector := metrics.MustNewMetrics(&metrics.Config{
		Namespace: "test",
	})

	m := newMessagingMetrics(collector)

	// 验证不会 panic
	m.RecordSendError("test-topic")
	m.RecordSendError("test-topic")
	m.RecordConsumeError("test-topic", "group1")
}

func TestMessagingMetrics_RecordRetryAndDLQ(t *testing.T) {
	collector := metrics.MustNewMetrics(&metrics.Config{
		Namespace: "test",
	})

	m := newMessagingMetrics(collector)

	// 验证不会 panic
	m.RecordRetry("test-topic")
	m.RecordRetry("test-topic")
	m.RecordRetry("test-topic")
	m.RecordDLQ("test-topic")
}
