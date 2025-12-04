package messaging

import (
	"time"

	"github.com/Tsukikage7/microservice-kit/metrics"
)

// messagingMetrics 消息队列指标记录器.
//
// 封装 metrics.PrometheusCollector，提供消息队列特定的指标记录方法.
type messagingMetrics struct {
	collector *metrics.PrometheusCollector
}

// newMessagingMetrics 创建消息队列指标记录器.
func newMessagingMetrics(collector *metrics.PrometheusCollector) *messagingMetrics {
	return &messagingMetrics{collector: collector}
}

// RecordSend 记录消息发送.
func (m *messagingMetrics) RecordSend(topic string, latency time.Duration) {
	labels := map[string]string{"topic": topic}
	m.collector.Counter("messaging_messages_sent_total", labels)
	m.collector.Histogram("messaging_send_duration_seconds", latency.Seconds(), labels)
}

// RecordSendError 记录发送错误.
func (m *messagingMetrics) RecordSendError(topic string) {
	m.collector.Counter("messaging_send_errors_total", map[string]string{"topic": topic})
}

// RecordConsume 记录消息消费.
func (m *messagingMetrics) RecordConsume(topic, groupID string, latency time.Duration) {
	labels := map[string]string{"topic": topic, "group": groupID}
	m.collector.Counter("messaging_messages_consumed_total", labels)
	m.collector.Histogram("messaging_consume_duration_seconds", latency.Seconds(), labels)
}

// RecordConsumeError 记录消费错误.
func (m *messagingMetrics) RecordConsumeError(topic, groupID string) {
	m.collector.Counter("messaging_consume_errors_total", map[string]string{"topic": topic, "group": groupID})
}

// RecordRetry 记录重试.
func (m *messagingMetrics) RecordRetry(topic string) {
	m.collector.Counter("messaging_retries_total", map[string]string{"topic": topic})
}

// RecordDLQ 记录死信队列消息.
func (m *messagingMetrics) RecordDLQ(topic string) {
	m.collector.Counter("messaging_dlq_total", map[string]string{"topic": topic})
}
