package useragent

import (
	"net/http"
)

// HTTPMiddleware 返回 HTTP 中间件，从请求头解析 User-Agent 并存入 context.
func HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get("User-Agent")
			ua := Parse(raw)
			ctx := WithUserAgent(r.Context(), ua)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
