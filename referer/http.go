package referer

import (
	"net/http"
)

// Option 配置选项函数.
type Option func(*options)

type options struct {
	currentHost string
}

// WithCurrentHost 设置当前站点域名，用于判断站内/站外跳转.
func WithCurrentHost(host string) Option {
	return func(o *options) {
		o.currentHost = host
	}
}

// HTTPMiddleware 返回 HTTP 中间件，从请求头解析 Referer 并存入 context.
func HTTPMiddleware(opts ...Option) func(http.Handler) http.Handler {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get("Referer")

			var ref *Referer
			if o.currentHost != "" {
				ref = ParseWithHost(raw, o.currentHost)
			} else {
				// 尝试从请求中获取当前主机
				currentHost := r.Host
				if currentHost != "" {
					ref = ParseWithHost(raw, currentHost)
				} else {
					ref = Parse(raw)
				}
			}

			ctx := WithReferer(r.Context(), ref)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
