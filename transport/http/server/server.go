// Package server 提供 HTTP 服务器实现.
package server

import (
	"context"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/Tsukikage7/microservice-kit/auth"
	"github.com/Tsukikage7/microservice-kit/logger"
	"github.com/Tsukikage7/microservice-kit/middleware/recovery"
	"github.com/Tsukikage7/microservice-kit/observability/tracing"
	"github.com/Tsukikage7/microservice-kit/request/clientip"
	"github.com/Tsukikage7/microservice-kit/transport"
	"github.com/Tsukikage7/microservice-kit/transport/health"
)

// Server HTTP 服务器.
type Server struct {
	opts    *options
	handler http.Handler
	server  *http.Server
	health  *health.Health
}

// New 创建 HTTP 服务器.
//
// 示例:
//
//	server := httpserver.New(mux,
//	    httpserver.Logger(log),
//	    httpserver.Addr(":8080"),
//	    httpserver.Feature(fm),
//	    httpserver.Auth(authenticator, "/api/login", "/api/register"),
//	    httpserver.Recovery(),
//	    httpserver.Profiling("/debug/pprof"),
//	)
func New(handler http.Handler, opts ...Option) *Server {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	if o.logger == nil {
		panic("http server: logger is required")
	}

	// 创建健康检查
	healthOpts := []health.Option{health.WithTimeout(o.healthTimeout)}
	healthOpts = append(healthOpts, o.healthOptions...)
	h := health.New(healthOpts...)

	// 中间件包装（由内到外）
	wrapped := health.Middleware(h)(handler)

	if o.clientIP {
		wrapped = clientip.HTTPMiddleware(o.clientIPOpts...)(wrapped)
	}

	if o.authenticator != nil {
		wrapped = auth.HTTPMiddleware(o.authenticator, o.authOpts...)(wrapped)
	}

	if o.traceName != "" {
		wrapped = tracing.HTTPMiddleware(o.traceName)(wrapped)
	}

	if o.recovery {
		wrapped = recovery.HTTPMiddleware(recovery.WithLogger(o.logger))(wrapped)
	}

	if o.profiling != "" {
		wrapped = wrapProfiling(wrapped, o.profiling, o.profilingAuth)
	}

	return &Server{opts: o, handler: wrapped, health: h}
}

func wrapProfiling(next http.Handler, prefix string, authFn func(*http.Request) bool) http.Handler {
	prefix = strings.TrimSuffix(prefix, "/")
	mux := http.NewServeMux()

	// 认证包装器
	wrap := func(h http.HandlerFunc) http.HandlerFunc {
		if authFn == nil {
			return h
		}
		return func(w http.ResponseWriter, r *http.Request) {
			if !authFn(r) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			h(w, r)
		}
	}

	// 注册 pprof 端点
	mux.HandleFunc(prefix+"/", wrap(pprof.Index))
	mux.HandleFunc(prefix+"/cmdline", wrap(pprof.Cmdline))
	mux.HandleFunc(prefix+"/profile", wrap(pprof.Profile))
	mux.HandleFunc(prefix+"/symbol", wrap(pprof.Symbol))
	mux.HandleFunc(prefix+"/trace", wrap(pprof.Trace))
	mux.HandleFunc(prefix+"/heap", wrap(func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("heap").ServeHTTP(w, r)
	}))
	mux.HandleFunc(prefix+"/goroutine", wrap(func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("goroutine").ServeHTTP(w, r)
	}))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, prefix) {
			mux.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Start 启动服务器.
func (s *Server) Start(ctx context.Context) error {
	s.server = &http.Server{
		Addr:         s.opts.addr,
		Handler:      s.handler,
		ReadTimeout:  s.opts.readTimeout,
		WriteTimeout: s.opts.writeTimeout,
		IdleTimeout:  s.opts.idleTimeout,
	}

	s.opts.logger.With(
		logger.String("name", s.opts.name),
		logger.String("addr", s.opts.addr),
	).Info("[HTTP] 服务器启动")

	errCh := make(chan error, 1)
	go func() { errCh <- s.server.ListenAndServe() }()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
	}
	return nil
}

// Stop 停止服务器.
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	s.opts.logger.With(logger.String("name", s.opts.name)).Info("[HTTP] 服务器停止")
	return s.server.Shutdown(ctx)
}

func (s *Server) Name() string          { return s.opts.name }
func (s *Server) Addr() string          { return s.opts.addr }
func (s *Server) Handler() http.Handler { return s.handler }
func (s *Server) Health() *health.Health { return s.health }

func (s *Server) HealthEndpoint() *transport.HealthEndpoint {
	return &transport.HealthEndpoint{
		Type: transport.HealthCheckTypeHTTP,
		Addr: s.opts.addr,
		Path: health.DefaultLivenessPath,
	}
}

// ==================== Options ====================

type Option func(*options)

type options struct {
	name         string
	addr         string
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
	logger       logger.Logger

	// Health
	healthTimeout time.Duration
	healthOptions []health.Option

	// Middleware
	recovery      bool
	traceName     string
	clientIP      bool
	clientIPOpts  []clientip.Option
	authenticator auth.Authenticator
	authOpts      []auth.Option
	profiling     string
	profilingAuth func(*http.Request) bool
}

func defaultOptions() *options {
	return &options{
		name:          "HTTP",
		addr:          ":8080",
		readTimeout:   30 * time.Second,
		writeTimeout:  30 * time.Second,
		idleTimeout:   120 * time.Second,
		healthTimeout: 5 * time.Second,
	}
}

// Logger 设置日志记录器（必需）.
func Logger(l logger.Logger) Option {
	return func(o *options) { o.logger = l }
}

// Name 设置服务器名称.
func Name(name string) Option {
	return func(o *options) { o.name = name }
}

// Addr 设置监听地址.
func Addr(addr string) Option {
	return func(o *options) { o.addr = addr }
}

// Timeout 设置超时时间.
func Timeout(read, write, idle time.Duration) Option {
	return func(o *options) {
		if read > 0 {
			o.readTimeout = read
		}
		if write > 0 {
			o.writeTimeout = write
		}
		if idle > 0 {
			o.idleTimeout = idle
		}
	}
}

// Recovery 启用 panic 恢复.
func Recovery() Option {
	return func(o *options) { o.recovery = true }
}

// Trace 启用链路追踪.
func Trace(serviceName string) Option {
	return func(o *options) { o.traceName = serviceName }
}

// ClientIP 启用客户端 IP 提取.
func ClientIP(opts ...clientip.Option) Option {
	return func(o *options) {
		o.clientIP = true
		o.clientIPOpts = opts
	}
}

// Auth 启用认证，可选指定公开路径.
//
// 示例:
//
//	httpserver.Auth(authenticator)                           // 所有路径都需认证
//	httpserver.Auth(authenticator, "/login", "/register")    // 指定公开路径
//	httpserver.Auth(authenticator, "/api/public/*")          // 前缀匹配
func Auth(authenticator auth.Authenticator, publicPaths ...string) Option {
	return func(o *options) {
		o.authenticator = authenticator
		if o.logger != nil {
			o.authOpts = append(o.authOpts, auth.WithLogger(o.logger))
		}
		if len(publicPaths) > 0 {
			o.authOpts = append(o.authOpts, auth.WithSkipper(buildPathSkipper(publicPaths)))
		}
	}
}

// Profiling 启用 pprof 端点.
//
// 示例:
//
//	httpserver.Profiling("/debug/pprof")
func Profiling(pathPrefix string) Option {
	return func(o *options) { o.profiling = pathPrefix }
}

// ProfilingWithAuth 启用带认证的 pprof 端点.
func ProfilingWithAuth(pathPrefix string, authFn func(*http.Request) bool) Option {
	return func(o *options) {
		o.profiling = pathPrefix
		o.profilingAuth = authFn
	}
}

// HealthTimeout 设置健康检查超时.
func HealthTimeout(d time.Duration) Option {
	return func(o *options) { o.healthTimeout = d }
}

// HealthChecker 添加健康检查器.
func HealthChecker(checkers ...health.Checker) Option {
	return func(o *options) {
		o.healthOptions = append(o.healthOptions, health.WithReadinessChecker(checkers...))
	}
}

// buildPathSkipper 构建路径跳过器.
func buildPathSkipper(paths []string) auth.Skipper {
	exact := make(map[string]bool)
	var prefixes []string

	for _, p := range paths {
		if len(p) > 0 && p[len(p)-1] == '*' {
			prefixes = append(prefixes, p[:len(p)-1])
		} else {
			exact[p] = true
		}
	}

	return func(_ context.Context, req any) bool {
		r, ok := req.(*http.Request)
		if !ok {
			return false
		}
		if exact[r.URL.Path] {
			return true
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				return true
			}
		}
		return false
	}
}

var _ transport.HealthCheckable = (*Server)(nil)
