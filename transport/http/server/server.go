// Package server 提供 HTTP 服务器实现.
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/Tsukikage7/microservice-kit/logger"
	"github.com/Tsukikage7/microservice-kit/transport"
)

// Server HTTP 服务器.
type Server struct {
	opts    *options
	handler http.Handler
	server  *http.Server
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

	return &Server{
		opts:    o,
		handler: handler,
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

// Option 配置选项函数.
type Option func(*options)

// options 服务器配置.
type options struct {
	name         string
	addr         string
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
	logger       logger.Logger
}

// defaultOptions 返回默认配置.
func defaultOptions() *options {
	return &options{
		name:         "HTTP",
		addr:         ":8080",
		readTimeout:  30 * time.Second,
		writeTimeout: 30 * time.Second,
		idleTimeout:  120 * time.Second,
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
