// Package server 提供应用服务器框架.
//
// 特性：
//   - 统一管理多个服务器（HTTP、gRPC 等）
//   - 生命周期钩子（BeforeStart/AfterStart/BeforeStop/AfterStop）
//   - 优雅关闭
//   - 信号处理
//
// 示例：
//
//	// 创建 HTTP 服务器
//	httpSrv := server.NewHTTP(handler,
//	    server.WithHTTPAddr(":8080"),
//	)
//
//	// 创建 gRPC 服务器
//	grpcSrv := server.NewGRPC(
//	    server.WithGRPCAddr(":9090"),
//	)
//	grpcSrv.Register(userService)
//
//	// 创建应用并运行
//	app := server.NewApp(
//	    server.WithName("my-service"),
//	    server.WithLogger(log),
//	)
//	app.Use(httpSrv, grpcSrv)
//	app.Run()
package server

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Server 服务器接口.
type Server interface {
	// Start 启动服务器（阻塞）.
	Start(ctx context.Context) error

	// Stop 停止服务器.
	Stop(ctx context.Context) error

	// Name 服务器名称.
	Name() string

	// Addr 服务器地址.
	Addr() string
}

// App 应用程序，管理多个服务器的生命周期.
type App struct {
	opts    *appOptions
	servers []Server
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	running bool
}

// NewApp 创建应用程序.
//
// 如果未设置 logger，会 panic.
func NewApp(opts ...AppOption) *App {
	o := defaultAppOptions()
	for _, opt := range opts {
		opt(o)
	}

	if o.logger == nil {
		panic("server: 必须设置 logger")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &App{
		opts:   o,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Use 注册服务器.
//
// 支持链式调用.
func (a *App) Use(servers ...Server) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.servers = append(a.servers, servers...)
	return a
}

// Run 运行应用程序.
//
// 阻塞直到收到关闭信号或调用 Stop.
func (a *App) Run() error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return ErrServerRunning
	}
	a.running = true
	a.mu.Unlock()

	// 执行启动前钩子
	if err := a.opts.hooks.runBeforeStart(a.ctx); err != nil {
		return err
	}

	a.opts.logger.Debugf("[%s] 应用启动中 [version:%s]", a.opts.name, a.opts.version)

	// 启动所有服务器
	if err := a.start(); err != nil {
		return err
	}

	// 执行启动后钩子
	if err := a.opts.hooks.runAfterStart(a.ctx); err != nil {
		a.opts.logger.Errorf("[%s] 启动后钩子执行失败 [error:%v]", a.opts.name, err)
	}

	// 等待关闭信号
	return a.waitForShutdown()
}

// Stop 主动停止应用程序.
func (a *App) Stop() {
	a.cancel()
}

// Context 获取应用上下文.
func (a *App) Context() context.Context {
	return a.ctx
}

// Name 获取应用名称.
func (a *App) Name() string {
	return a.opts.name
}

// Version 获取应用版本.
func (a *App) Version() string {
	return a.opts.version
}

// start 启动所有服务器.
func (a *App) start() error {
	if len(a.servers) == 0 {
		a.opts.logger.Warnf("[%s] 没有注册任何服务器", a.opts.name)
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(a.servers))

	for _, srv := range a.servers {
		wg.Add(1)
		go func(s Server) {
			defer wg.Done()
			a.opts.logger.Debugf("[%s] 启动服务器 [name:%s] [addr:%s]", a.opts.name, s.Name(), s.Addr())
			if err := s.Start(a.ctx); err != nil {
				errCh <- err
			}
		}(srv)
	}

	// 非阻塞检查启动错误
	go func() {
		wg.Wait()
		close(errCh)
	}()

	return nil
}

// waitForShutdown 等待关闭信号.
func (a *App) waitForShutdown() error {
	signals := a.opts.signals
	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, signals...)

	select {
	case sig := <-sigCh:
		a.opts.logger.Debugf("[%s] 收到信号 [signal:%s]", a.opts.name, sig.String())
	case <-a.ctx.Done():
		a.opts.logger.Debugf("[%s] 上下文已取消", a.opts.name)
	}

	return a.shutdown()
}

// shutdown 优雅关闭.
func (a *App) shutdown() error {
	a.opts.logger.Debugf("[%s] 开始优雅关闭 [timeout:%v]", a.opts.name, a.opts.gracefulTimeout)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.gracefulTimeout)
	defer cancel()

	// 执行停止前钩子
	if err := a.opts.hooks.runBeforeStop(shutdownCtx); err != nil {
		a.opts.logger.Errorf("[%s] 停止前钩子执行失败 [error:%v]", a.opts.name, err)
	}

	// 停止所有服务器
	var wg sync.WaitGroup
	for _, srv := range a.servers {
		wg.Add(1)
		go func(s Server) {
			defer wg.Done()
			a.opts.logger.Debugf("[%s] 停止服务器 [name:%s]", a.opts.name, s.Name())
			if err := s.Stop(shutdownCtx); err != nil {
				a.opts.logger.Errorf("[%s] 服务器停止失败 [name:%s] [error:%v]", a.opts.name, s.Name(), err)
			}
		}(srv)
	}

	// 等待所有服务器停止或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.opts.logger.Debugf("[%s] 所有服务器已停止", a.opts.name)
	case <-shutdownCtx.Done():
		a.opts.logger.Warnf("[%s] 关闭超时，强制退出", a.opts.name)
	}

	// 执行停止后钩子
	if err := a.opts.hooks.runAfterStop(context.Background()); err != nil {
		a.opts.logger.Errorf("[%s] 停止后钩子执行失败 [error:%v]", a.opts.name, err)
	}

	a.mu.Lock()
	a.running = false
	a.mu.Unlock()

	a.opts.logger.Debugf("[%s] 应用已关闭", a.opts.name)
	return nil
}
