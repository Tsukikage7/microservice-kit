package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPHandler_LivenessHandler(t *testing.T) {
	h := New(WithLivenessChecker(NewAlwaysUpChecker("test")))
	handler := NewHTTPHandler(h)

	t.Run("GET returns 200 when healthy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		handler.LivenessHandler()(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var resp Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, StatusUp, resp.Status)
	})

	t.Run("POST returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
		rec := httptest.NewRecorder()

		handler.LivenessHandler()(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})
}

func TestHTTPHandler_ReadinessHandler(t *testing.T) {
	t.Run("returns 200 when healthy", func(t *testing.T) {
		h := New(WithReadinessChecker(NewAlwaysUpChecker("test")))
		handler := NewHTTPHandler(h)

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()

		handler.ReadinessHandler()(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, StatusUp, resp.Status)
	})

	t.Run("returns 503 when unhealthy", func(t *testing.T) {
		downChecker := NewCheckerFunc("down", func(ctx context.Context) CheckResult {
			return CheckResult{Status: StatusDown, Message: "service down"}
		})
		h := New(WithReadinessChecker(downChecker))
		handler := NewHTTPHandler(h)

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()

		handler.ReadinessHandler()(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

		var resp Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, StatusDown, resp.Status)
	})
}

func TestHTTPHandler_RegisterRoutes(t *testing.T) {
	h := New()
	handler := NewHTTPHandler(h)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	t.Run("healthz endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("readyz endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHTTPHandler_RegisterRoutesWithPrefix(t *testing.T) {
	h := New()
	handler := NewHTTPHandler(h)

	mux := http.NewServeMux()
	handler.RegisterRoutesWithPrefix(mux, "/api/v1")

	t.Run("prefixed healthz endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("prefixed readyz endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/readyz", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestLivenessHandlerFunc(t *testing.T) {
	h := New()
	handler := LivenessHandlerFunc(h)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestReadinessHandlerFunc(t *testing.T) {
	h := New()
	handler := ReadinessHandlerFunc(h)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMiddleware(t *testing.T) {
	h := New()

	// 创建一个简单的 handler 用于测试
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	})

	middleware := Middleware(h)
	wrappedHandler := middleware(nextHandler)

	t.Run("intercepts healthz", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	})

	t.Run("intercepts readyz", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	})

	t.Run("passes through other requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "hello", rec.Body.String())
	})
}

func TestHTTPHandler_CacheHeaders(t *testing.T) {
	h := New()
	handler := NewHTTPHandler(h)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.LivenessHandler()(rec, req)

	assert.Equal(t, "no-cache, no-store, must-revalidate", rec.Header().Get("Cache-Control"))
}
