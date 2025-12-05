package server

import (
	"context"
	"net/http"
)

// HTTP HTTP 服务器.
type HTTP struct {
	opts    *httpOptions
	handler http.Handler
	server  *http.Server
}

// NewHTTP 创建 HTTP 服务器.
//
// 如果未设置 logger，会 panic.
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

	if o.logger == nil {
		panic("server: 必须设置 logger")
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

	s.opts.logger.Debugf("[%s] 服务器启动 [addr:%s]", s.opts.name, s.opts.addr)

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

	s.opts.logger.Debugf("[%s] 服务器停止中", s.opts.name)
	return s.server.Shutdown(ctx)
}

// Name 返回服务器名称.
func (s *HTTP) Name() string {
	return s.opts.name
}

// Addr 返回服务器地址.
func (s *HTTP) Addr() string {
	return s.opts.addr
}

// Handler 返回 HTTP Handler.
func (s *HTTP) Handler() http.Handler {
	return s.handler
}

