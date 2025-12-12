package deviceinfo

import (
	"net/http"
)

// HTTPMiddleware 返回 HTTP 中间件，解析设备信息并存入 context.
func HTTPMiddleware(opts ...Option) func(http.Handler) http.Handler {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	parser := New(opts...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 设置 Accept-CH 响应头
			if o.setAcceptCH {
				w.Header().Set("Accept-CH", AcceptCHHeader())
			}

			// 提取请求头
			headers := Headers{
				SecCHUA:                r.Header.Get("Sec-CH-UA"),
				SecCHUAMobile:          r.Header.Get("Sec-CH-UA-Mobile"),
				SecCHUAPlatform:        r.Header.Get("Sec-CH-UA-Platform"),
				SecCHUAPlatformVersion: r.Header.Get("Sec-CH-UA-Platform-Version"),
				SecCHUAArch:            r.Header.Get("Sec-CH-UA-Arch"),
				SecCHUAModel:           r.Header.Get("Sec-CH-UA-Model"),
				SecCHUABitness:         r.Header.Get("Sec-CH-UA-Bitness"),
				SecCHUAFullVersionList: r.Header.Get("Sec-CH-UA-Full-Version-List"),
				DeviceMemory:           r.Header.Get("Device-Memory"),
				ViewportWidth:          r.Header.Get("Viewport-Width"),
				DPR:                    r.Header.Get("DPR"),
				UserAgent:              r.Header.Get("User-Agent"),
			}

			info := parser.Parse(headers)
			ctx := WithInfo(r.Context(), info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
