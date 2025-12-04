package messaging

import (
	"context"
	"testing"
)

func TestMessagingTracer_Creation(t *testing.T) {
	tracer := newMessagingTracer("test-service")

	if tracer == nil {
		t.Fatal("expected non-nil tracer")
	}
	if tracer.tracer == nil {
		t.Error("expected non-nil otel tracer")
	}
	if tracer.propagator == nil {
		t.Error("expected non-nil propagator")
	}
}

func TestMessagingTracer_StartProducerSpan(t *testing.T) {
	tracer := newMessagingTracer("test-service")

	ctx, span := tracer.startProducerSpan(context.Background(), "test-topic")

	if ctx == nil {
		t.Error("expected non-nil context")
	}
	if span == nil {
		t.Error("expected non-nil span")
	}

	// 清理
	span.End()
}

func TestMessagingTracer_StartConsumerSpan(t *testing.T) {
	tracer := newMessagingTracer("test-service")

	ctx, span := tracer.startConsumerSpan(context.Background(), "test-topic", 0, 100)

	if ctx == nil {
		t.Error("expected non-nil context")
	}
	if span == nil {
		t.Error("expected non-nil span")
	}

	// 清理
	span.End()
}

func TestMessagingTracer_InjectHeaders(t *testing.T) {
	tracer := newMessagingTracer("test-service")

	t.Run("nil headers", func(t *testing.T) {
		headers := tracer.injectHeaders(context.Background(), nil)
		if headers == nil {
			t.Error("expected non-nil headers map")
		}
	})

	t.Run("existing headers preserved", func(t *testing.T) {
		existing := map[string]string{"custom": "value"}
		headers := tracer.injectHeaders(context.Background(), existing)
		if headers["custom"] != "value" {
			t.Error("expected existing headers to be preserved")
		}
	})
}

func TestMessagingTracer_ExtractContext(t *testing.T) {
	tracer := newMessagingTracer("test-service")

	t.Run("nil headers", func(t *testing.T) {
		ctx := tracer.extractContext(context.Background(), nil)
		if ctx == nil {
			t.Error("expected non-nil context")
		}
	})

	t.Run("empty headers", func(t *testing.T) {
		ctx := tracer.extractContext(context.Background(), map[string]string{})
		if ctx == nil {
			t.Error("expected non-nil context")
		}
	})
}

func TestMessagingTracer_SetError(t *testing.T) {
	tracer := newMessagingTracer("test-service")

	t.Run("nil span", func(t *testing.T) {
		// 不应该 panic
		tracer.setError(nil, nil)
	})

	t.Run("nil error", func(t *testing.T) {
		_, span := tracer.startProducerSpan(context.Background(), "test-topic")
		defer span.End()

		// 不应该 panic
		tracer.setError(span, nil)
	})
}

func TestMapCarrier(t *testing.T) {
	headers := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	carrier := &mapCarrier{headers: headers}

	t.Run("Get", func(t *testing.T) {
		if carrier.Get("key1") != "value1" {
			t.Error("expected 'value1'")
		}
		if carrier.Get("nonexistent") != "" {
			t.Error("expected empty string for nonexistent key")
		}
	})

	t.Run("Set", func(t *testing.T) {
		carrier.Set("key3", "value3")
		if carrier.Get("key3") != "value3" {
			t.Error("expected 'value3'")
		}
	})

	t.Run("Keys", func(t *testing.T) {
		keys := carrier.Keys()
		if len(keys) != 3 {
			t.Errorf("expected 3 keys, got %d", len(keys))
		}
	})
}
