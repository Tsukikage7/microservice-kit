package ratelimit

import (
	"net/http"
)

// HTTPMiddleware 创建 HTTP 限流中间件.
//
// 当请求被限流时返回 429 Too Many Requests.
func HTTPMiddleware(limiter Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(r.Context()) {
				http.Error(w, "请求过于频繁，请稍后重试", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// HTTPMiddlewareWithWait 创建阻塞式 HTTP 限流中间件.
//
// 当请求被限流时阻塞等待，直到可以通过或请求超时.
func HTTPMiddlewareWithWait(limiter Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := limiter.Wait(r.Context()); err != nil {
				http.Error(w, "请求超时", http.StatusGatewayTimeout)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// HTTPKeyFunc 用于从 HTTP 请求中提取限流键.
type HTTPKeyFunc func(r *http.Request) string

// KeyedHTTPMiddleware 创建基于键的 HTTP 限流中间件.
//
// 可以基于 IP 地址、用户 ID 等进行限流.
func KeyedHTTPMiddleware(keyFunc HTTPKeyFunc, getLimiter KeyedLimiterFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			limiter := getLimiter(key)
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}
			if !limiter.Allow(r.Context()) {
				http.Error(w, "请求过于频繁，请稍后重试", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IPKeyFunc 返回基于客户端 IP 的键提取函数.
func IPKeyFunc() HTTPKeyFunc {
	return func(r *http.Request) string {
		// 优先使用 X-Forwarded-For
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return xff
		}
		// 使用 X-Real-IP
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}
		// 使用 RemoteAddr
		return r.RemoteAddr
	}
}

// PathKeyFunc 返回基于请求路径的键提取函数.
func PathKeyFunc() HTTPKeyFunc {
	return func(r *http.Request) string {
		return r.URL.Path
	}
}

// CompositeKeyFunc 组合多个键提取函数.
func CompositeKeyFunc(funcs ...HTTPKeyFunc) HTTPKeyFunc {
	return func(r *http.Request) string {
		var key string
		for _, f := range funcs {
			if key != "" {
				key += ":"
			}
			key += f(r)
		}
		return key
	}
}
