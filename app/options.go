package app

import (
	"context"
	"os"
	"time"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// CleanupFunc 清理函数.
type CleanupFunc func(ctx context.Context) error

// Cleanup 清理任务.
type Cleanup struct {
	Name     string
	Fn       CleanupFunc
	Priority int // 优先级，数字越小越先执行
}

// options 内部配置.
type options struct {
	name            string
	version         string
	logger          logger.Logger
	hooks           *Hooks
	gracefulTimeout time.Duration
	signals         []os.Signal
	cleanups        []Cleanup
}

func defaultOptions() *options {
	return &options{
		name:            "app",
		version:         "1.0.0",
		gracefulTimeout: 30 * time.Second,
	}
}

// Option 配置选项.
type Option func(*options)

// Name 设置应用名称.
func Name(name string) Option {
	return func(o *options) { o.name = name }
}

// Version 设置应用版本.
func Version(version string) Option {
	return func(o *options) { o.version = version }
}

// Logger 设置日志记录器（必需）.
func Logger(log logger.Logger) Option {
	return func(o *options) { o.logger = log }
}

// Hooks 设置生命周期钩子.
func SetHooks(hooks *Hooks) Option {
	return func(o *options) { o.hooks = hooks }
}

// GracefulTimeout 设置优雅关闭超时时间.
func GracefulTimeout(d time.Duration) Option {
	return func(o *options) { o.gracefulTimeout = d }
}

// Signals 设置监听的系统信号.
func Signals(signals ...os.Signal) Option {
	return func(o *options) { o.signals = signals }
}

// Cleanup 注册清理任务.
func RegisterCleanup(name string, fn CleanupFunc, priority int) Option {
	return func(o *options) {
		o.cleanups = append(o.cleanups, Cleanup{
			Name:     name,
			Fn:       fn,
			Priority: priority,
		})
	}
}

// Closer 注册 io.Closer 作为清理任务.
func RegisterCloser(name string, closer interface{ Close() error }, priority int) Option {
	return RegisterCleanup(name, func(_ context.Context) error {
		return closer.Close()
	}, priority)
}
