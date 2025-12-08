package health

import (
	"encoding/json"
	"net/http"
)

const (
	// DefaultLivenessPath 默认存活检查路径.
	DefaultLivenessPath = "/healthz"
	// DefaultReadinessPath 默认就绪检查路径.
	DefaultReadinessPath = "/readyz"
)

// HTTPHandler HTTP 健康检查处理器.
type HTTPHandler struct {
	health *Health
}

// NewHTTPHandler 创建 HTTP 健康检查处理器.
func NewHTTPHandler(h *Health) *HTTPHandler {
	return &HTTPHandler{health: h}
}

// LivenessHandler 返回存活检查 HTTP Handler.
func (h *HTTPHandler) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		resp := h.health.Liveness(r.Context())
		h.writeResponse(w, resp)
	}
}

// ReadinessHandler 返回就绪检查 HTTP Handler.
func (h *HTTPHandler) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		resp := h.health.Readiness(r.Context())
		h.writeResponse(w, resp)
	}
}

// writeResponse 写入 HTTP 响应.
func (h *HTTPHandler) writeResponse(w http.ResponseWriter, resp Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	statusCode := http.StatusOK
	if resp.Status != StatusUp {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

// RegisterRoutes 注册健康检查路由到 http.ServeMux.
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc(DefaultLivenessPath, h.LivenessHandler())
	mux.HandleFunc(DefaultReadinessPath, h.ReadinessHandler())
}

// RegisterRoutesWithPrefix 注册带前缀的健康检查路由.
func (h *HTTPHandler) RegisterRoutesWithPrefix(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+DefaultLivenessPath, h.LivenessHandler())
	mux.HandleFunc(prefix+DefaultReadinessPath, h.ReadinessHandler())
}

// LivenessHandlerFunc 便捷函数，直接返回存活检查 Handler.
func LivenessHandlerFunc(h *Health) http.HandlerFunc {
	return NewHTTPHandler(h).LivenessHandler()
}

// ReadinessHandlerFunc 便捷函数，直接返回就绪检查 Handler.
func ReadinessHandlerFunc(h *Health) http.HandlerFunc {
	return NewHTTPHandler(h).ReadinessHandler()
}

// Middleware 健康检查中间件，自动拦截健康检查路径.
//
// 用于在不修改原有路由的情况下添加健康检查支持.
func Middleware(h *Health) func(http.Handler) http.Handler {
	handler := NewHTTPHandler(h)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case DefaultLivenessPath:
				handler.LivenessHandler()(w, r)
				return
			case DefaultReadinessPath:
				handler.ReadinessHandler()(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
