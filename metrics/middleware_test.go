package metrics

import (
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
