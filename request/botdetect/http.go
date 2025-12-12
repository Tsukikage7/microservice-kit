package botdetect

import (
	"net/http"
)

// HTTPMiddleware 返回 HTTP 中间件，检测机器人并存入 context.
func HTTPMiddleware(opts ...Option) func(http.Handler) http.Handler {
	detector := New(opts...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent := r.Header.Get("User-Agent")
			result := detector.Detect(userAgent)
			ctx := WithResult(r.Context(), result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// BlockBotsMiddleware 返回 HTTP 中间件，阻止机器人访问.
func BlockBotsMiddleware(opts ...Option) func(http.Handler) http.Handler {
	detector := New(opts...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent := r.Header.Get("User-Agent")
			result := detector.Detect(userAgent)

			// 只阻止恶意机器人，允许良性机器人
			if result.IsBadBot() {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			ctx := WithResult(r.Context(), result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AllowOnlyGoodBotsMiddleware 返回 HTTP 中间件，只允许良性机器人和人类.
func AllowOnlyGoodBotsMiddleware(opts ...Option) func(http.Handler) http.Handler {
	detector := New(opts...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent := r.Header.Get("User-Agent")
			result := detector.Detect(userAgent)

			// 阻止未知意图的机器人
			if result.IsBot && result.Intent != IntentGood {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			ctx := WithResult(r.Context(), result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
