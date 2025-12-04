package messaging

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// messagingTracer 消息队列追踪器.
//
// 使用全局 OpenTelemetry TracerProvider，与 HTTP/gRPC 追踪统一管理.
// 需要先通过 trace.NewTracer 初始化全局 TracerProvider.
type messagingTracer struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

// newMessagingTracer 创建消息队列追踪器.
func newMessagingTracer(serviceName string) *messagingTracer {
	return &messagingTracer{
		tracer:     otel.Tracer(serviceName),
		propagator: otel.GetTextMapPropagator(),
	}
}

// startProducerSpan 开始生产者 span.
func (t *messagingTracer) startProducerSpan(ctx context.Context, topic string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "kafka.produce",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination.name", topic),
			attribute.String("messaging.operation", "publish"),
		),
	)
}

// startConsumerSpan 开始消费者 span.
func (t *messagingTracer) startConsumerSpan(ctx context.Context, topic string, partition int32, offset int64) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "kafka.consume",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination.name", topic),
			attribute.String("messaging.operation", "receive"),
			attribute.Int64("messaging.kafka.partition", int64(partition)),
			attribute.Int64("messaging.kafka.offset", offset),
		),
	)
}

// injectHeaders 将追踪上下文注入到消息 Headers.
func (t *messagingTracer) injectHeaders(ctx context.Context, headers map[string]string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}
	carrier := &mapCarrier{headers: headers}
	t.propagator.Inject(ctx, carrier)
	return headers
}

// extractContext 从消息 Headers 提取追踪上下文.
func (t *messagingTracer) extractContext(ctx context.Context, headers map[string]string) context.Context {
	if headers == nil {
		return ctx
	}
	carrier := &mapCarrier{headers: headers}
	return t.propagator.Extract(ctx, carrier)
}

// setError 设置 span 错误.
func (t *messagingTracer) setError(span trace.Span, err error) {
	if span != nil && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// mapCarrier 实现 propagation.TextMapCarrier 接口.
type mapCarrier struct {
	headers map[string]string
}

func (c *mapCarrier) Get(key string) string {
	return c.headers[key]
}

func (c *mapCarrier) Set(key, value string) {
	c.headers[key] = value
}

func (c *mapCarrier) Keys() []string {
	keys := make([]string, 0, len(c.headers))
	for k := range c.headers {
		keys = append(keys, k)
	}
	return keys
}
