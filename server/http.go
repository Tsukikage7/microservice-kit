package server

import (
	"context"
	"net/http"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// HTTP HTTP 服务器.
type HTTP struct {
	opts    *httpOptions
	handler http.Handler
	server  *http.Server
}

// NewHTTP 创建 HTTP 服务器.
//
// 示例:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/health", healthHandler)
//
//	srv := server.NewHTTP(mux,
//	    server.WithHTTPAddr(":8080"),
//	    server.WithHTTPLogger(log),
//	)
func NewHTTP(handler http.Handler, opts ...HTTPOption) *HTTP {
	o := defaultHTTPOptions()
	for _, opt := range opts {
		opt(o)
	}

	return &HTTP{
		opts:    o,
		handler: handler,
	}
}

// Start 启动 HTTP 服务器.
func (s *HTTP) Start(ctx context.Context) error {
	s.server = &http.Server{
		Addr:         s.opts.addr,
		Handler:      s.handler,
		ReadTimeout:  s.opts.readTimeout,
		WriteTimeout: s.opts.writeTimeout,
		IdleTimeout:  s.opts.idleTimeout,
	}

	s.logDebugf("HTTP 服务器启动 [addr:%s]", s.opts.addr)

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
func (s *HTTP) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	s.logDebug("HTTP 服务器停止中...")
	return s.server.Shutdown(ctx)
}

// Name 返回服务器名称.
func (s *HTTP) Name() string {
	return "http"
}

// Addr 返回服务器地址.
func (s *HTTP) Addr() string {
	return s.opts.addr
}

// Handler 返回 HTTP Handler.
func (s *HTTP) Handler() http.Handler {
	return s.handler
}

// 日志辅助方法.

func (s *HTTP) logger() logger.Logger {
	return s.opts.logger
}

func (s *HTTP) logDebug(msg string) {
	if log := s.logger(); log != nil {
		log.Debug("[HTTP] " + msg)
	}
}

func (s *HTTP) logDebugf(format string, args ...any) {
	if log := s.logger(); log != nil {
		log.Debugf("[HTTP] "+format, args...)
	}
}
