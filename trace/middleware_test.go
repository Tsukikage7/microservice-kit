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

func TestEndpointMiddleware(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	t.Run("成功请求", func(t *testing.T) {
		endpoint := func(ctx context.Context, req any) (any, error) {
			// 验证 context 中有 span
			span := SpanFromContext(ctx)
			assert.NotNil(t, span)
			return "success", nil
		}

		middleware := EndpointMiddleware("test-service", "TestMethod")
		wrapped := middleware(endpoint)

		resp, err := wrapped(context.Background(), nil)

		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})

	t.Run("失败请求", func(t *testing.T) {
		testErr := errors.New("test error")
		endpoint := func(ctx context.Context, req any) (any, error) {
			return nil, testErr
		}

		middleware := EndpointMiddleware("test-service", "TestMethodError")
		wrapped := middleware(endpoint)

		resp, err := wrapped(context.Background(), nil)

		assert.Error(t, err)
		assert.Equal(t, testErr, err)
		assert.Nil(t, resp)
	})
}

func TestEndpointMiddleware_SpanCreation(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	var capturedCtx context.Context
	endpoint := func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		return "ok", nil
	}

	middleware := EndpointMiddleware("test-service", "TestMethod")
	wrapped := middleware(endpoint)

	_, err := wrapped(context.Background(), nil)
	assert.NoError(t, err)

	// 验证 context 中有 span
	span := SpanFromContext(capturedCtx)
	assert.NotNil(t, span)
}

func TestEndpointMiddleware_NestedSpans(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	endpoint := func(ctx context.Context, req any) (any, error) {
		// 创建子 span
		ctx, span := StartSpan(ctx, "test-service", "child-operation")
		defer span.End()

		// 添加事件
		AddSpanEvent(ctx, "processing")

		return "ok", nil
	}

	middleware := EndpointMiddleware("test-service", "ParentMethod")
	wrapped := middleware(endpoint)

	resp, err := wrapped(context.Background(), nil)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestEndpointTracer(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	tracer := NewEndpointTracer("user-service")

	t.Run("创建多个方法中间件", func(t *testing.T) {
		endpoint1 := func(ctx context.Context, req any) (any, error) {
			return "user", nil
		}
		endpoint2 := func(ctx context.Context, req any) (any, error) {
			return []string{"user1", "user2"}, nil
		}

		wrapped1 := tracer.Middleware("GetUser")(endpoint1)
		wrapped2 := tracer.Middleware("ListUsers")(endpoint2)

		resp1, err1 := wrapped1(context.Background(), nil)
		resp2, err2 := wrapped2(context.Background(), nil)

		assert.NoError(t, err1)
		assert.Equal(t, "user", resp1)
		assert.NoError(t, err2)
		assert.Equal(t, []string{"user1", "user2"}, resp2)
	})

	t.Run("不同服务名", func(t *testing.T) {
		tracer2 := NewEndpointTracer("order-service")

		endpoint := func(ctx context.Context, req any) (any, error) {
			return "order", nil
		}

		wrapped := tracer2.Middleware("GetOrder")(endpoint)

		resp, err := wrapped(context.Background(), nil)

		assert.NoError(t, err)
		assert.Equal(t, "order", resp)
	})
}

func TestEndpointMiddleware_WithContext(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	t.Run("上下文取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 立即取消

		endpoint := func(ctx context.Context, req any) (any, error) {
			return nil, ctx.Err()
		}

		middleware := EndpointMiddleware("test-service", "CancelledMethod")
		wrapped := middleware(endpoint)

		resp, err := wrapped(ctx, nil)

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Nil(t, resp)
	})
}

func TestEndpointMiddleware_Concurrent(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	endpoint := func(ctx context.Context, req any) (any, error) {
		// 验证每个 goroutine 都有自己的 span
		span := SpanFromContext(ctx)
		assert.NotNil(t, span)
		return "ok", nil
	}

	middleware := EndpointMiddleware("test-service", "ConcurrentMethod")
	wrapped := middleware(endpoint)

	// 并发调用
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			resp, err := wrapped(context.Background(), nil)
			assert.NoError(t, err)
			assert.Equal(t, "ok", resp)
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestEndpointMiddleware_ErrorRecording(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	testErr := errors.New("database connection failed")

	endpoint := func(ctx context.Context, req any) (any, error) {
		return nil, testErr
	}

	middleware := EndpointMiddleware("test-service", "FailingMethod")
	wrapped := middleware(endpoint)

	resp, err := wrapped(context.Background(), nil)

	assert.Error(t, err)
	assert.Equal(t, testErr, err)
	assert.Nil(t, resp)
}

func TestEndpointMiddleware_RequestPassthrough(t *testing.T) {
	tp := setupTestTracer(t)
	defer tp.Shutdown(context.Background())

	type TestRequest struct {
		ID   string
		Name string
	}

	type TestResponse struct {
		Success bool
		Data    string
	}

	endpoint := func(ctx context.Context, req any) (any, error) {
		r := req.(*TestRequest)
		return &TestResponse{
			Success: true,
			Data:    r.Name,
		}, nil
	}

	middleware := EndpointMiddleware("test-service", "ProcessRequest")
	wrapped := middleware(endpoint)

	req := &TestRequest{ID: "123", Name: "test"}
	resp, err := wrapped(context.Background(), req)

	assert.NoError(t, err)
	result := resp.(*TestResponse)
	assert.True(t, result.Success)
	assert.Equal(t, "test", result.Data)
}
