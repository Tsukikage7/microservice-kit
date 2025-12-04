package metrics

import (
	"net/http"
	"strconv"
	"time"
)

// HTTPMiddleware 返回 HTTP 指标采集中间件.
//
// 使用示例:
//
//	collector, _ := metrics.New(cfg)
//	handler := metrics.HTTPMiddleware(collector)(mux)
//	http.ListenAndServe(":8080", handler)
func HTTPMiddleware(collector *PrometheusCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// 包装 ResponseWriter 捕获状态码和响应大小
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// 执行下一个处理器
			next.ServeHTTP(rw, r)

			// 记录指标
			collector.RecordHTTPRequest(
				r.Method,
				r.URL.Path,
				strconv.Itoa(rw.statusCode),
				time.Since(start),
				float64(r.ContentLength),
				float64(rw.size),
			)
		})
	}
}

// responseWriter 包装 http.ResponseWriter 以捕获状态码和响应大小.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}
