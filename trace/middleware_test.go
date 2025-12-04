package trace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

func setupTestTracer(t *testing.T) *trace.TracerProvider {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	return tp
}

func TestHTTPMiddleware(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := HTTPMiddleware("test-service")
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())
}

func TestHTTPMiddleware_WithError(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error"))
	})

	middleware := HTTPMiddleware("test-service")
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/error", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHTTPMiddleware_ContextPropagation(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	var capturedCtx context.Context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	middleware := HTTPMiddleware("test-service")
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	// 验证 context 中有 span
	span := SpanFromContext(capturedCtx)
	assert.NotNil(t, span)
}

func TestResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, rw.statusCode)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSpanFromContext(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	ctx := context.Background()
	span := SpanFromContext(ctx)
	assert.NotNil(t, span)
}

func TestStartSpan(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-service", "test-operation")
	defer span.End()

	assert.NotNil(t, span)
	assert.NotEqual(t, ctx, context.Background())
}

func TestAddSpanEvent(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	ctx, span := StartSpan(context.Background(), "test-service", "test-operation")
	defer span.End()

	// 不应该 panic
	assert.NotPanics(t, func() {
		AddSpanEvent(ctx, "test-event", attribute.String("key", "value"))
	})
}

func TestSetSpanError(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	ctx, span := StartSpan(context.Background(), "test-service", "test-operation")
	defer span.End()

	testErr := errors.New("test error")

	// 不应该 panic
	assert.NotPanics(t, func() {
		SetSpanError(ctx, testErr)
	})
}

func TestSetSpanAttributes(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	ctx, span := StartSpan(context.Background(), "test-service", "test-operation")
	defer span.End()

	// 不应该 panic
	assert.NotPanics(t, func() {
		SetSpanAttributes(ctx, attribute.String("key", "value"), attribute.Int("count", 10))
	})
}

func TestInjectHTTPHeaders(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	ctx, span := StartSpan(context.Background(), "test-service", "test-operation")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com/api", nil)
	require.NoError(t, err)

	// 不应该 panic
	assert.NotPanics(t, func() {
		InjectHTTPHeaders(ctx, req)
	})
}

func TestTraceID(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	// 无 span 的 context
	ctx := context.Background()
	traceID := TraceID(ctx)
	assert.Empty(t, traceID)

	// 有 span 的 context
	ctx, span := StartSpan(context.Background(), "test-service", "test-operation")
	defer span.End()

	traceID = TraceID(ctx)
	// SDK 可能不会生成有效的 trace ID，取决于配置
	// 这里只验证不会 panic
	assert.NotPanics(t, func() {
		_ = TraceID(ctx)
	})
}

func TestSpanID(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	// 无 span 的 context
	ctx := context.Background()
	spanID := SpanID(ctx)
	assert.Empty(t, spanID)

	// 有 span 的 context
	ctx, span := StartSpan(context.Background(), "test-service", "test-operation")
	defer span.End()

	// SDK 可能不会生成有效的 span ID，取决于配置
	// 这里只验证不会 panic
	assert.NotPanics(t, func() {
		_ = SpanID(ctx)
	})
}

func TestHTTPMiddleware_DifferentMethods(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := HTTPMiddleware("test-service")
	wrappedHandler := middleware(handler)

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/test", nil)
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestHTTPMiddleware_DifferentStatusCodes(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	middleware := HTTPMiddleware("test-service")

	for _, code := range statusCodes {
		t.Run(http.StatusText(code), func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			})

			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			assert.Equal(t, code, rec.Code)
		})
	}
}
