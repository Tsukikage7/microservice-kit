package locale

import (
	"net/http"
)

// HTTPMiddleware 返回 HTTP 中间件，从请求头解析 Accept-Language 并存入 context.
func HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get("Accept-Language")
			loc := Parse(raw)
			ctx := WithLocale(r.Context(), loc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
