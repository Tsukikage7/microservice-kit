// Package server 提供 HTTP 服务器实现.
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/Tsukikage7/microservice-kit/logger"
	"github.com/Tsukikage7/microservice-kit/trace"
	"github.com/Tsukikage7/microservice-kit/transport"
	"github.com/Tsukikage7/microservice-kit/transport/health"
)

// Server HTTP 服务器.
type Server struct {
	opts    *options
	handler http.Handler
	server  *http.Server

	// 内置健康检查
	health *health.Health
}

// New 创建 HTTP 服务器，如果未设置 logger 会 panic.
func New(handler http.Handler, opts ...Option) *Server {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	if o.logger == nil {
		panic("http server: 必须设置 logger")
	}

	// 创建内置健康检查管理器
	healthOpts := []health.Option{health.WithTimeout(o.healthTimeout)}
	healthOpts = append(healthOpts, o.healthOptions...)
	h := health.New(healthOpts...)

	// 使用健康检查中间件包装 handler
	wrappedHandler := health.Middleware(h)(handler)

	// 如果启用链路追踪，使用 trace 中间件包装（在最外层）
	if o.tracerName != "" {
		wrappedHandler = trace.HTTPMiddleware(o.tracerName)(wrappedHandler)
	}

	return &Server{
		opts:    o,
		handler: wrappedHandler,
		health:  h,
	}
}

// Start 启动 HTTP 服务器.
func (s *Server) Start(ctx context.Context) error {
	s.server = &http.Server{
		Addr:         s.opts.addr,
		Handler:      s.handler,
		ReadTimeout:  s.opts.readTimeout,
		WriteTimeout: s.opts.writeTimeout,
		IdleTimeout:  s.opts.idleTimeout,
	}

	s.opts.logger.Infof("[%s] 服务器启动 [addr:%s]", s.opts.name, s.opts.addr)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
		// 上下文取消，正常退出
	}

	return nil
}

// Stop 停止 HTTP 服务器.
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	s.opts.logger.Infof("[%s] 服务器停止中", s.opts.name)
	return s.server.Shutdown(ctx)
}

// Name 返回服务器名称.
func (s *Server) Name() string {
	return s.opts.name
}

// Addr 返回服务器地址.
func (s *Server) Addr() string {
	return s.opts.addr
}

// Handler 返回 HTTP Handler.
func (s *Server) Handler() http.Handler {
	return s.handler
}

// Health 返回健康检查管理器.
func (s *Server) Health() *health.Health {
	return s.health
}

// HealthEndpoint 返回健康检查端点信息.
func (s *Server) HealthEndpoint() *transport.HealthEndpoint {
	return &transport.HealthEndpoint{
		Type: transport.HealthCheckTypeHTTP,
		Addr: s.opts.addr,
		Path: health.DefaultLivenessPath,
	}
}

// Option 配置选项函数.
type Option func(*options)

// options 服务器配置.
type options struct {
	name          string
	addr          string
	readTimeout   time.Duration
	writeTimeout  time.Duration
	idleTimeout   time.Duration
	logger        logger.Logger
	healthTimeout time.Duration
	healthOptions []health.Option
	tracerName    string // 链路追踪服务名，为空则不启用
}

// defaultOptions 返回默认配置.
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

// WithName 设置服务器名称.
func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// WithAddr 设置监听地址.
func WithAddr(addr string) Option {
	return func(o *options) {
		o.addr = addr
	}
}

// WithReadTimeout 设置读取超时.
func WithReadTimeout(d time.Duration) Option {
	return func(o *options) {
		o.readTimeout = d
	}
}

// WithWriteTimeout 设置写入超时.
func WithWriteTimeout(d time.Duration) Option {
	return func(o *options) {
		o.writeTimeout = d
	}
}

// WithIdleTimeout 设置空闲超时.
func WithIdleTimeout(d time.Duration) Option {
	return func(o *options) {
		o.idleTimeout = d
	}
}

// WithConfig 从配置结构体设置服务器选项.
// 仅设置非零值字段，零值字段将保持默认值.
func WithConfig(cfg transport.HTTPConfig) Option {
	return func(o *options) {
		if cfg.Name != "" {
			o.name = cfg.Name
		}
		if cfg.Addr != "" {
			o.addr = cfg.Addr
		}
		if cfg.ReadTimeout > 0 {
			o.readTimeout = cfg.ReadTimeout
		}
		if cfg.WriteTimeout > 0 {
			o.writeTimeout = cfg.WriteTimeout
		}
		if cfg.IdleTimeout > 0 {
			o.idleTimeout = cfg.IdleTimeout
		}
	}
}

// WithLogger 设置日志记录器（必需）.
func WithLogger(log logger.Logger) Option {
	return func(o *options) {
		o.logger = log
	}
}

// WithHealthTimeout 设置健康检查超时时间.
func WithHealthTimeout(d time.Duration) Option {
	return func(o *options) {
		o.healthTimeout = d
	}
}

// WithHealthOptions 添加健康检查选项.
func WithHealthOptions(opts ...health.Option) Option {
	return func(o *options) {
		o.healthOptions = append(o.healthOptions, opts...)
	}
}

// WithReadinessChecker 添加就绪检查器（便捷方法）.
func WithReadinessChecker(checkers ...health.Checker) Option {
	return func(o *options) {
		o.healthOptions = append(o.healthOptions, health.WithReadinessChecker(checkers...))
	}
}

// WithLivenessChecker 添加存活检查器（便捷方法）.
func WithLivenessChecker(checkers ...health.Checker) Option {
	return func(o *options) {
		o.healthOptions = append(o.healthOptions, health.WithLivenessChecker(checkers...))
	}
}

// WithTrace 启用链路追踪.
//
// 启用后，会自动添加 trace 中间件，从请求中提取或生成 traceId/spanId，
// 并注入到 context 中，业务代码可通过 log.WithContext(ctx) 自动获取这些字段.
//
// 注意: 需要先调用 trace.NewTracer() 初始化全局 TracerProvider.
//
// 使用示例:
//
//	// 初始化 TracerProvider
//	tp := trace.MustNewTracer(cfg, "my-service", "1.0.0")
//	defer tp.Shutdown(context.Background())
//
//	// 创建 HTTP 服务器
//	srv := server.New(handler,
//	    server.WithLogger(log),
//	    server.WithTrace("my-service"),
//	)
//
//	// 业务代码中
//	func handleRequest(w http.ResponseWriter, r *http.Request) {
//	    log.WithContext(r.Context()).Info("处理请求")
//	    // 输出: {"msg":"处理请求","traceId":"abc...","spanId":"def..."}
//	}
func WithTrace(serviceName string) Option {
	return func(o *options) {
		o.tracerName = serviceName
	}
}

// 确保 Server 实现了 transport.HealthCheckable 接口.
var _ transport.HealthCheckable = (*Server)(nil)
