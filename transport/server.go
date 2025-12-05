package transport

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Tsukikage7/microservice-kit/logger"
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

// AppOption App 配置选项.
type AppOption func(*appOptions)

// appOptions App 内部配置.
type appOptions struct {
	name            string
	version         string
	logger          logger.Logger
	hooks           *Hooks
	gracefulTimeout time.Duration
	signals         []os.Signal
}

// defaultAppOptions 返回默认配置.
func defaultAppOptions() *appOptions {
	return &appOptions{
		name:            "app",
		version:         "1.0.0",
		gracefulTimeout: 30 * time.Second,
	}
}

// WithName 设置应用名称.
func WithName(name string) AppOption {
	return func(o *appOptions) {
		o.name = name
	}
}

// WithVersion 设置应用版本.
func WithVersion(version string) AppOption {
	return func(o *appOptions) {
		o.version = version
	}
}

// WithLogger 设置日志记录器（必需）.
func WithLogger(log logger.Logger) AppOption {
	return func(o *appOptions) {
		o.logger = log
	}
}

// WithHooks 设置生命周期钩子.
func WithHooks(hooks *Hooks) AppOption {
	return func(o *appOptions) {
		o.hooks = hooks
	}
}

// WithGracefulTimeout 设置优雅关闭超时时间.
//
// 默认: 30 秒.
func WithGracefulTimeout(d time.Duration) AppOption {
	return func(o *appOptions) {
		o.gracefulTimeout = d
	}
}

// WithSignals 设置监听的系统信号.
//
// 默认: SIGINT, SIGTERM.
func WithSignals(signals ...os.Signal) AppOption {
	return func(o *appOptions) {
		o.signals = signals
	}
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
		panic("transport: 必须设置 logger")
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

	a.opts.logger.Infof("[%s] 应用启动中 [version:%s]", a.opts.name, a.opts.version)

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
			a.opts.logger.Infof("[%s] 启动服务器 [name:%s] [addr:%s]", a.opts.name, s.Name(), s.Addr())
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
	defer signal.Stop(sigCh)

	select {
	case sig := <-sigCh:
		a.opts.logger.Infof("[%s] 收到信号 [signal:%s]", a.opts.name, sig.String())
	case <-a.ctx.Done():
		a.opts.logger.Infof("[%s] 上下文已取消", a.opts.name)
	}

	return a.shutdown()
}

// shutdown 优雅关闭.
func (a *App) shutdown() error {
	a.opts.logger.Infof("[%s] 开始优雅关闭 [timeout:%v]", a.opts.name, a.opts.gracefulTimeout)

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
			a.opts.logger.Infof("[%s] 停止服务器 [name:%s]", a.opts.name, s.Name())
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
		a.opts.logger.Infof("[%s] 所有服务器已停止", a.opts.name)
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

	a.opts.logger.Infof("[%s] 应用已关闭", a.opts.name)
	return nil
}
