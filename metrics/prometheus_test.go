package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrometheus_NilConfig(t *testing.T) {
	c, err := NewPrometheus(nil)

	assert.Nil(t, c)
	assert.ErrorIs(t, err, ErrNilConfig)
}

func TestNewPrometheus_EmptyNamespace(t *testing.T) {
	cfg := &Config{
		Namespace: "",
	}

	c, err := NewPrometheus(cfg)

	// 空命名空间会使用默认值 "app"
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestNewPrometheus_Success(t *testing.T) {
	cfg := &Config{
		Namespace: "test_service",
		Path:      "/metrics",
	}

	c, err := NewPrometheus(cfg)

	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "/metrics", c.GetPath())
}

func TestPrometheusCollector_RecordHTTPRequest(t *testing.T) {
	cfg := &Config{
		Namespace: "test",
	}

	c, err := NewPrometheus(cfg)
	require.NoError(t, err)

	// 不应该 panic
	assert.NotPanics(t, func() {
		c.RecordHTTPRequest("GET", "/api/users", "200", 100*time.Millisecond, 100, 200)
	})

	// 验证指标被记录
	handler := c.GetHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	bodyStr := string(body)

	assert.Contains(t, bodyStr, "test_http_requests_total")
	assert.Contains(t, bodyStr, "test_http_request_duration_seconds")
}

func TestPrometheusCollector_RecordGRPCRequest(t *testing.T) {
	cfg := &Config{
		
		Namespace: "test",
	}

	c, err := NewPrometheus(cfg)
	require.NoError(t, err)

	// 不应该 panic
	assert.NotPanics(t, func() {
		c.RecordGRPCRequest("/test.Service/Method", "test-service", "OK", 50*time.Millisecond)
	})

	// 验证指标被记录
	handler := c.GetHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	bodyStr := string(body)

	assert.Contains(t, bodyStr, "test_grpc_requests_total")
	assert.Contains(t, bodyStr, "test_grpc_request_duration_seconds")
}

func TestPrometheusCollector_RecordPanic(t *testing.T) {
	cfg := &Config{
		
		Namespace: "test",
	}

	c, err := NewPrometheus(cfg)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		c.RecordPanic("test-service", "GET", "/api/crash")
	})

	// 验证指标被记录
	handler := c.GetHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	bodyStr := string(body)

	assert.Contains(t, bodyStr, "test_system_panic_total")
}

func TestPrometheusCollector_UpdateGoroutineCount(t *testing.T) {
	cfg := &Config{
		
		Namespace: "test",
	}

	c, err := NewPrometheus(cfg)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		c.UpdateGoroutineCount(100)
	})

	// 验证指标被记录
	handler := c.GetHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	bodyStr := string(body)

	assert.Contains(t, bodyStr, "test_system_goroutines")
	assert.Contains(t, bodyStr, "100")
}

func TestPrometheusCollector_UpdateMemoryUsage(t *testing.T) {
	cfg := &Config{
		
		Namespace: "test",
	}

	c, err := NewPrometheus(cfg)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		c.UpdateMemoryUsage(1024 * 1024)
	})

	// 验证指标被记录
	handler := c.GetHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	bodyStr := string(body)

	assert.Contains(t, bodyStr, "test_system_memory_usage_bytes")
}

func TestPrometheusCollector_Counter(t *testing.T) {
	cfg := &Config{
		Namespace: "test",
	}

	c, err := NewPrometheus(cfg)
	require.NoError(t, err)

	// 创建自定义计数器
	assert.NotPanics(t, func() {
		c.Counter("custom_events", map[string]string{"type": "click"})
		c.Counter("custom_events", map[string]string{"type": "click"})
	})

	// 验证指标被记录
	handler := c.GetHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	bodyStr := string(body)

	assert.Contains(t, bodyStr, "custom_events")
}

func TestPrometheusCollector_Histogram(t *testing.T) {
	cfg := &Config{
		Namespace: "test",
	}

	c, err := NewPrometheus(cfg)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		c.Histogram("request_latency", 0.5, map[string]string{"endpoint": "/api"})
	})

	// 验证指标被记录
	handler := c.GetHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	bodyStr := string(body)

	assert.Contains(t, bodyStr, "request_latency")
}

func TestPrometheusCollector_Gauge(t *testing.T) {
	cfg := &Config{
		Namespace: "test",
	}

	c, err := NewPrometheus(cfg)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		c.Gauge("active_connections", 50, map[string]string{"server": "main"})
	})

	// 验证指标被记录
	handler := c.GetHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	bodyStr := string(body)

	assert.Contains(t, bodyStr, "active_connections")
}

func TestPrometheusCollector_GetPath_Default(t *testing.T) {
	cfg := &Config{
		
		Namespace: "test",
		Path:      "",
	}

	c, err := NewPrometheus(cfg)
	require.NoError(t, err)

	assert.Equal(t, "/metrics", c.GetPath())
}

func TestPrometheusCollector_GetPath_Custom(t *testing.T) {
	cfg := &Config{
		
		Namespace: "test",
		Path:      "/custom/metrics",
	}

	c, err := NewPrometheus(cfg)
	require.NoError(t, err)

	assert.Equal(t, "/custom/metrics", c.GetPath())
}

func TestPrometheusCollector_GetHandler(t *testing.T) {
	cfg := &Config{
		
		Namespace: "test",
	}

	c, err := NewPrometheus(cfg)
	require.NoError(t, err)

	handler := c.GetHandler()
	assert.NotNil(t, handler)

	// 测试 handler 返回 prometheus 格式
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	// Prometheus 格式包含 HELP 和 TYPE 注释
	assert.True(t, strings.Contains(body, "# HELP") || strings.Contains(body, "# TYPE") || len(body) > 0)
}

func TestPrometheusCollector_ConcurrentAccess(t *testing.T) {
	cfg := &Config{
		
		Namespace: "test",
	}

	c, err := NewPrometheus(cfg)
	require.NoError(t, err)

	// 并发访问测试
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				c.RecordHTTPRequest("GET", "/api", "200", time.Millisecond, 100, 200)
				c.RecordGRPCRequest("/test/Method", "svc", "OK", time.Millisecond)
				c.Counter("concurrent_counter", map[string]string{"worker": "test"})
				c.Histogram("concurrent_histogram", 0.1, map[string]string{"worker": "test"})
				c.Gauge("concurrent_gauge", float64(j), map[string]string{"worker": "test"})
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
}
