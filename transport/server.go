package transport

import (
	"context"
	"os"
	"os/signal"
	"sort"
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

// ApplicationConfig 应用程序配置，支持从配置文件加载.
type ApplicationConfig struct {
	Name            string        `json:"name" toml:"name" yaml:"name" mapstructure:"name"`
	Version         string        `json:"version" toml:"version" yaml:"version" mapstructure:"version"`
	GracefulTimeout time.Duration `json:"graceful_timeout" toml:"graceful_timeout" yaml:"graceful_timeout" mapstructure:"graceful_timeout"`
	HTTP            HTTPConfig    `json:"http" toml:"http" yaml:"http" mapstructure:"http"`
	GRPC            GRPCConfig    `json:"grpc" toml:"grpc" yaml:"grpc" mapstructure:"grpc"`
}

// HTTPConfig HTTP 服务器配置.
type HTTPConfig struct {
	Name         string        `json:"name" toml:"name" yaml:"name" mapstructure:"name"`
	Addr         string        `json:"addr" toml:"addr" yaml:"addr" mapstructure:"addr"`
	ReadTimeout  time.Duration `json:"read_timeout" toml:"read_timeout" yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" toml:"write_timeout" yaml:"write_timeout" mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout" toml:"idle_timeout" yaml:"idle_timeout" mapstructure:"idle_timeout"`
	PublicPaths  []string      `json:"public_paths" toml:"public_paths" yaml:"public_paths" mapstructure:"public_paths"`
}

// GRPCConfig gRPC 服务器配置.
type GRPCConfig struct {
	Name             string        `json:"name" toml:"name" yaml:"name" mapstructure:"name"`
	Addr             string        `json:"addr" toml:"addr" yaml:"addr" mapstructure:"addr"`
	EnableReflection bool          `json:"enable_reflection" toml:"enable_reflection" yaml:"enable_reflection" mapstructure:"enable_reflection"`
	KeepaliveTime    time.Duration `json:"keepalive_time" toml:"keepalive_time" yaml:"keepalive_time" mapstructure:"keepalive_time"`
	KeepaliveTimeout time.Duration `json:"keepalive_timeout" toml:"keepalive_timeout" yaml:"keepalive_timeout" mapstructure:"keepalive_timeout"`
	PublicMethods    []string      `json:"public_methods" toml:"public_methods" yaml:"public_methods" mapstructure:"public_methods"`
}

// GatewayConfig Gateway 服务器配置.
type GatewayConfig struct {
	Name          string        `json:"name" toml:"name" yaml:"name" mapstructure:"name"`
	GRPCAddr      string        `json:"grpc_addr" toml:"grpc_addr" yaml:"grpc_addr" mapstructure:"grpc_addr"`
	HTTPAddr      string        `json:"http_addr" toml:"http_addr" yaml:"http_addr" mapstructure:"http_addr"`
	PublicMethods []string      `json:"public_methods" toml:"public_methods" yaml:"public_methods" mapstructure:"public_methods"`
	KeepaliveTime time.Duration `json:"keepalive_time" toml:"keepalive_time" yaml:"keepalive_time" mapstructure:"keepalive_time"`
}

// AppOption App 配置选项.
type AppOption func(*appOptions)

// CleanupFunc 清理函数.
type CleanupFunc func(ctx context.Context) error

// Cleanup 清理任务.
type Cleanup struct {
	Name     string
	Fn       CleanupFunc
	Priority int // 优先级，数字越小越先执行
}

// appOptions App 内部配置.
type appOptions struct {
	name            string
	version         string
	logger          logger.Logger
	hooks           *Hooks
	gracefulTimeout time.Duration
	signals         []os.Signal
	cleanups        []Cleanup
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

// WithConfig 从配置结构体设置应用选项.
// 仅设置非零值字段，零值字段将保持默认值.
func WithConfig(cfg ApplicationConfig) AppOption {
	return func(o *appOptions) {
		if cfg.Name != "" {
			o.name = cfg.Name
		}
		if cfg.Version != "" {
			o.version = cfg.Version
		}
		if cfg.GracefulTimeout > 0 {
			o.gracefulTimeout = cfg.GracefulTimeout
		}
	}
}

// WithCleanup 注册清理任务.
//
// 清理任务在所有服务器停止后按优先级执行（数字越小越先执行）.
// 典型用途: 关闭数据库连接、Redis 连接、刷新日志缓冲等.
//
// 示例:
//
//	app := transport.NewApplication(
//	    transport.WithCleanup("database", db.Close, 10),
//	    transport.WithCleanup("redis", redis.Close, 10),
//	    transport.WithCleanup("logger", logger.Sync, 100),
//	)
func WithCleanup(name string, fn CleanupFunc, priority int) AppOption {
	return func(o *appOptions) {
		o.cleanups = append(o.cleanups, Cleanup{
			Name:     name,
			Fn:       fn,
			Priority: priority,
		})
	}
}

// WithCloser 注册 io.Closer 作为清理任务（便捷方法）.
func WithCloser(name string, closer interface{ Close() error }, priority int) AppOption {
	return WithCleanup(name, func(_ context.Context) error {
		return closer.Close()
	}, priority)
}

// Application 应用程序，管理多个服务器的生命周期.
type Application struct {
	opts    *appOptions
	servers []Server
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	running bool
}

// NewApplication 创建应用程序.
//
// 如果未设置 logger，会 panic.
func NewApplication(opts ...AppOption) *Application {
	o := defaultAppOptions()
	for _, opt := range opts {
		opt(o)
	}

	if o.logger == nil {
		panic("transport: 必须设置 logger")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Application{
		opts:   o,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Use 注册服务器.
//
// 支持链式调用.
func (a *Application) Use(servers ...Server) *Application {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.servers = append(a.servers, servers...)
	return a
}

// Run 运行应用程序.
//
// 阻塞直到收到关闭信号或调用 Stop.
func (a *Application) Run() error {
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

	a.opts.logger.With(
		logger.String("name", a.opts.name),
		logger.String("version", a.opts.version),
	).Info("[App] 应用启动中")

	// 启动所有服务器
	if err := a.start(); err != nil {
		return err
	}

	// 执行启动后钩子
	if err := a.opts.hooks.runAfterStart(a.ctx); err != nil {
		a.opts.logger.With(
			logger.String("name", a.opts.name),
			logger.Err(err),
		).Error("[App] 启动后钩子执行失败")
	}

	// 等待关闭信号
	return a.waitForShutdown()
}

// Stop 主动停止应用程序.
func (a *Application) Stop() {
	a.cancel()
}

// Context 获取应用上下文.
func (a *Application) Context() context.Context {
	return a.ctx
}

// Name 获取应用名称.
func (a *Application) Name() string {
	return a.opts.name
}

// Version 获取应用版本.
func (a *Application) Version() string {
	return a.opts.version
}

// start 启动所有服务器.
func (a *Application) start() error {
	if len(a.servers) == 0 {
		a.opts.logger.With(logger.String("name", a.opts.name)).Warn("[App] 没有注册任何服务器")
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(a.servers))

	for _, srv := range a.servers {
		wg.Add(1)
		go func(s Server) {
			defer wg.Done()
			a.opts.logger.With(
				logger.String("app", a.opts.name),
				logger.String("server", s.Name()),
				logger.String("addr", s.Addr()),
			).Info("[App] 启动服务器")
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
func (a *Application) waitForShutdown() error {
	signals := a.opts.signals
	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, signals...)
	defer signal.Stop(sigCh)

	select {
	case sig := <-sigCh:
		a.opts.logger.With(
			logger.String("name", a.opts.name),
			logger.String("signal", sig.String()),
		).Info("[App] 收到信号")
	case <-a.ctx.Done():
		a.opts.logger.With(logger.String("name", a.opts.name)).Info("[App] 上下文已取消")
	}

	return a.shutdown()
}

// shutdown 优雅关闭.
func (a *Application) shutdown() error {
	a.opts.logger.With(
		logger.String("name", a.opts.name),
		logger.Duration("timeout", a.opts.gracefulTimeout),
	).Info("[App] 开始优雅关闭")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.gracefulTimeout)
	defer cancel()

	// 执行停止前钩子
	if err := a.opts.hooks.runBeforeStop(shutdownCtx); err != nil {
		a.opts.logger.With(
			logger.String("name", a.opts.name),
			logger.Err(err),
		).Error("[App] 停止前钩子执行失败")
	}

	// 停止所有服务器
	var wg sync.WaitGroup
	for _, srv := range a.servers {
		wg.Add(1)
		go func(s Server) {
			defer wg.Done()
			a.opts.logger.With(
				logger.String("app", a.opts.name),
				logger.String("server", s.Name()),
			).Info("[App] 停止服务器")
			if err := s.Stop(shutdownCtx); err != nil {
				a.opts.logger.With(
					logger.String("app", a.opts.name),
					logger.String("server", s.Name()),
					logger.Err(err),
				).Error("[App] 服务器停止失败")
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
		a.opts.logger.With(logger.String("name", a.opts.name)).Info("[App] 所有服务器已停止")
	case <-shutdownCtx.Done():
		a.opts.logger.With(logger.String("name", a.opts.name)).Warn("[App] 服务器关闭超时")
	}

	// 执行清理任务（按优先级）
	a.runCleanups(shutdownCtx)

	// 执行停止后钩子
	if err := a.opts.hooks.runAfterStop(context.Background()); err != nil {
		a.opts.logger.With(
			logger.String("name", a.opts.name),
			logger.Err(err),
		).Error("[App] 停止后钩子执行失败")
	}

	a.mu.Lock()
	a.running = false
	a.mu.Unlock()

	a.opts.logger.With(logger.String("name", a.opts.name)).Info("[App] 应用已关闭")
	return nil
}

// runCleanups 按优先级执行清理任务.
func (a *Application) runCleanups(ctx context.Context) {
	if len(a.opts.cleanups) == 0 {
		return
	}

	// 复制并按优先级排序
	cleanups := make([]Cleanup, len(a.opts.cleanups))
	copy(cleanups, a.opts.cleanups)
	sort.Slice(cleanups, func(i, j int) bool {
		return cleanups[i].Priority < cleanups[j].Priority
	})

	a.opts.logger.With(
		logger.String("name", a.opts.name),
		logger.Int("count", len(cleanups)),
	).Info("[App] 执行清理任务")

	for _, c := range cleanups {
		if err := c.Fn(ctx); err != nil {
			a.opts.logger.With(
				logger.String("name", a.opts.name),
				logger.String("cleanup", c.Name),
				logger.Err(err),
			).Error("[App] 清理任务失败")
		} else {
			a.opts.logger.With(
				logger.String("name", a.opts.name),
				logger.String("cleanup", c.Name),
			).Info("[App] 清理任务完成")
		}
	}
}
