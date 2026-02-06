// Package app 提供应用程序生命周期管理.
package app

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// ErrRunning 应用正在运行.
var ErrRunning = errors.New("app: 应用正在运行")

// Server 服务器接口.
type Server interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Name() string
	Addr() string
}

// Application 应用程序，管理多个服务器的生命周期.
type Application struct {
	opts    *options
	servers []Server
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	running bool
}

// New 创建应用程序.
func New(opts ...Option) *Application {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	if o.logger == nil {
		panic("app: logger is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Application{
		opts:   o,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Use 注册服务器.
func (a *Application) Use(servers ...Server) *Application {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.servers = append(a.servers, servers...)
	return a
}

// Run 运行应用程序.
func (a *Application) Run() error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return ErrRunning
	}
	a.running = true
	a.mu.Unlock()

	if err := a.opts.hooks.runBeforeStart(a.ctx); err != nil {
		return err
	}

	a.opts.logger.With(
		logger.String("name", a.opts.name),
		logger.String("version", a.opts.version),
	).Info("[App] starting")

	if err := a.start(); err != nil {
		return err
	}

	if err := a.opts.hooks.runAfterStart(a.ctx); err != nil {
		a.opts.logger.With(logger.Err(err)).Error("[App] after start hook failed")
	}

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

func (a *Application) start() error {
	if len(a.servers) == 0 {
		a.opts.logger.Warn("[App] no servers registered")
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(a.servers))

	for _, srv := range a.servers {
		wg.Add(1)
		go func(s Server) {
			defer wg.Done()
			a.opts.logger.With(
				logger.String("server", s.Name()),
				logger.String("addr", s.Addr()),
			).Info("[App] starting server")
			if err := s.Start(a.ctx); err != nil {
				errCh <- err
			}
		}(srv)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	return nil
}

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
		a.opts.logger.With(logger.String("signal", sig.String())).Info("[App] received signal")
	case <-a.ctx.Done():
		a.opts.logger.Info("[App] context cancelled")
	}

	return a.shutdown()
}

func (a *Application) shutdown() error {
	a.opts.logger.With(
		logger.Duration("timeout", a.opts.gracefulTimeout),
	).Info("[App] shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.gracefulTimeout)
	defer cancel()

	if err := a.opts.hooks.runBeforeStop(shutdownCtx); err != nil {
		a.opts.logger.With(logger.Err(err)).Error("[App] before stop hook failed")
	}

	var wg sync.WaitGroup
	for _, srv := range a.servers {
		wg.Add(1)
		go func(s Server) {
			defer wg.Done()
			a.opts.logger.With(logger.String("server", s.Name())).Info("[App] stopping server")
			if err := s.Stop(shutdownCtx); err != nil {
				a.opts.logger.With(
					logger.String("server", s.Name()),
					logger.Err(err),
				).Error("[App] server stop failed")
			}
		}(srv)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.opts.logger.Info("[App] all servers stopped")
	case <-shutdownCtx.Done():
		a.opts.logger.Warn("[App] shutdown timeout")
	}

	a.runCleanups(shutdownCtx)

	if err := a.opts.hooks.runAfterStop(context.Background()); err != nil {
		a.opts.logger.With(logger.Err(err)).Error("[App] after stop hook failed")
	}

	a.mu.Lock()
	a.running = false
	a.mu.Unlock()

	a.opts.logger.Info("[App] stopped")
	return nil
}

func (a *Application) runCleanups(ctx context.Context) {
	if len(a.opts.cleanups) == 0 {
		return
	}

	cleanups := make([]Cleanup, len(a.opts.cleanups))
	copy(cleanups, a.opts.cleanups)
	sort.Slice(cleanups, func(i, j int) bool {
		return cleanups[i].Priority < cleanups[j].Priority
	})

	a.opts.logger.With(logger.Int("count", len(cleanups))).Info("[App] running cleanups")

	for _, c := range cleanups {
		if err := c.Fn(ctx); err != nil {
			a.opts.logger.With(
				logger.String("cleanup", c.Name),
				logger.Err(err),
			).Error("[App] cleanup failed")
		} else {
			a.opts.logger.With(logger.String("cleanup", c.Name)).Info("[App] cleanup done")
		}
	}
}
