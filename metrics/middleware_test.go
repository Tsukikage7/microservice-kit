package metrics

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPMiddleware(t *testing.T) {
	cfg := &Config{Namespace: "test"}
	collector, err := NewPrometheus(cfg)
	require.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := HTTPMiddleware(collector)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())

	// 验证指标被记录
	metricsHandler := collector.GetHandler()
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	metricsHandler.ServeHTTP(metricsRec, metricsReq)

	body, _ := io.ReadAll(metricsRec.Body)
	bodyStr := string(body)

	assert.Contains(t, bodyStr, "test_http_requests_total")
	assert.Contains(t, bodyStr, `method="GET"`)
	assert.Contains(t, bodyStr, `path="/api/test"`)
	assert.Contains(t, bodyStr, `status_code="200"`)
}

func TestHTTPMiddleware_WithError(t *testing.T) {
	cfg := &Config{Namespace: "test"}
	collector, err := NewPrometheus(cfg)
	require.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error"))
	})

	middleware := HTTPMiddleware(collector)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/error", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// 验证指标被记录
	metricsHandler := collector.GetHandler()
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	metricsHandler.ServeHTTP(metricsRec, metricsReq)

	body, _ := io.ReadAll(metricsRec.Body)
	bodyStr := string(body)

	assert.Contains(t, bodyStr, `status_code="500"`)
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, rw.statusCode)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	n, err := rw.Write([]byte("hello"))

	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, 5, rw.size)
	assert.Equal(t, "hello", rec.Body.String())
}

func TestHTTPMiddleware_DifferentMethods(t *testing.T) {
	cfg := &Config{Namespace: "test"}
	collector, err := NewPrometheus(cfg)
	require.NoError(t, err)

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

	middleware := HTTPMiddleware(collector)
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

func TestEndpointMiddleware(t *testing.T) {
	cfg := &Config{Namespace: "test_endpoint"}
	collector, err := NewPrometheus(cfg)
	require.NoError(t, err)

	t.Run("成功请求", func(t *testing.T) {
		endpoint := func(ctx context.Context, req any) (any, error) {
			return "success", nil
		}

		middleware := EndpointMiddleware(collector, "test-service", "TestMethod")
		wrapped := middleware(endpoint)

		resp, err := wrapped(context.Background(), nil)

		assert.NoError(t, err)
		assert.Equal(t, "success", resp)

		// 验证指标被记录
		metricsHandler := collector.GetHandler()
		metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		metricsRec := httptest.NewRecorder()
		metricsHandler.ServeHTTP(metricsRec, metricsReq)

		body, _ := io.ReadAll(metricsRec.Body)
		bodyStr := string(body)

		assert.Contains(t, bodyStr, "test_endpoint_grpc_requests_total")
		assert.Contains(t, bodyStr, `method="TestMethod"`)
		assert.Contains(t, bodyStr, `service="test-service"`)
		assert.Contains(t, bodyStr, `status_code="OK"`)
	})

	t.Run("失败请求", func(t *testing.T) {
		testErr := errors.New("test error")
		endpoint := func(ctx context.Context, req any) (any, error) {
			return nil, testErr
		}

		middleware := EndpointMiddleware(collector, "test-service", "TestMethodError")
		wrapped := middleware(endpoint)

		resp, err := wrapped(context.Background(), nil)

		assert.Error(t, err)
		assert.Equal(t, testErr, err)
		assert.Nil(t, resp)

		// 验证指标被记录
		metricsHandler := collector.GetHandler()
		metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		metricsRec := httptest.NewRecorder()
		metricsHandler.ServeHTTP(metricsRec, metricsReq)

		body, _ := io.ReadAll(metricsRec.Body)
		bodyStr := string(body)

		assert.Contains(t, bodyStr, `method="TestMethodError"`)
		assert.Contains(t, bodyStr, `status_code="ERROR"`)
	})
}

func TestEndpointInstrumenter(t *testing.T) {
	cfg := &Config{Namespace: "test_instrumenter"}
	collector, err := NewPrometheus(cfg)
	require.NoError(t, err)

	instrumenter := NewEndpointInstrumenter(collector, "user-service")

	t.Run("创建多个方法中间件", func(t *testing.T) {
		endpoint1 := func(ctx context.Context, req any) (any, error) {
			return "user", nil
		}
		endpoint2 := func(ctx context.Context, req any) (any, error) {
			return []string{"user1", "user2"}, nil
		}

		wrapped1 := instrumenter.Middleware("GetUser")(endpoint1)
		wrapped2 := instrumenter.Middleware("ListUsers")(endpoint2)

		resp1, err1 := wrapped1(context.Background(), nil)
		resp2, err2 := wrapped2(context.Background(), nil)

		assert.NoError(t, err1)
		assert.Equal(t, "user", resp1)
		assert.NoError(t, err2)
		assert.Equal(t, []string{"user1", "user2"}, resp2)

		// 验证指标被记录
		metricsHandler := collector.GetHandler()
		metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		metricsRec := httptest.NewRecorder()
		metricsHandler.ServeHTTP(metricsRec, metricsReq)

		body, _ := io.ReadAll(metricsRec.Body)
		bodyStr := string(body)

		assert.Contains(t, bodyStr, `method="GetUser"`)
		assert.Contains(t, bodyStr, `method="ListUsers"`)
		assert.Contains(t, bodyStr, `service="user-service"`)
	})
}

func TestEndpointMiddleware_WithContext(t *testing.T) {
	cfg := &Config{Namespace: "test_ctx"}
	collector, err := NewPrometheus(cfg)
	require.NoError(t, err)

	t.Run("上下文取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 立即取消

		endpoint := func(ctx context.Context, req any) (any, error) {
			return nil, ctx.Err()
		}

		middleware := EndpointMiddleware(collector, "test-service", "CancelledMethod")
		wrapped := middleware(endpoint)

		resp, err := wrapped(ctx, nil)

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Nil(t, resp)
	})
}

func TestEndpointMiddleware_Concurrent(t *testing.T) {
	cfg := &Config{Namespace: "test_concurrent"}
	collector, err := NewPrometheus(cfg)
	require.NoError(t, err)

	endpoint := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	middleware := EndpointMiddleware(collector, "test-service", "ConcurrentMethod")
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

	// 验证指标被记录
	metricsHandler := collector.GetHandler()
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	metricsHandler.ServeHTTP(metricsRec, metricsReq)

	body, _ := io.ReadAll(metricsRec.Body)
	bodyStr := string(body)

	assert.Contains(t, bodyStr, `method="ConcurrentMethod"`)
}
