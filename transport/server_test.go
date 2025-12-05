package transport

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// mockLogger 测试用 mock logger.
type mockLogger struct {
	infos  []string
	warns  []string
	errors []string
}

func newMockLogger() *mockLogger                                    { return &mockLogger{} }
func (m *mockLogger) Debug(args ...any)                             {}
func (m *mockLogger) Debugf(format string, args ...any)             {}
func (m *mockLogger) Info(args ...any)                              {}
func (m *mockLogger) Infof(format string, args ...any)              { m.infos = append(m.infos, format) }
func (m *mockLogger) Warn(args ...any)                              {}
func (m *mockLogger) Warnf(format string, args ...any)              { m.warns = append(m.warns, format) }
func (m *mockLogger) Error(args ...any)                             {}
func (m *mockLogger) Errorf(format string, args ...any)             { m.errors = append(m.errors, format) }
func (m *mockLogger) Fatal(args ...any)                             {}
func (m *mockLogger) Fatalf(format string, args ...any)             {}
func (m *mockLogger) Panic(args ...any)                             {}
func (m *mockLogger) Panicf(format string, args ...any)             {}
func (m *mockLogger) With(fields ...logger.Field) logger.Logger     { return m }
func (m *mockLogger) WithContext(ctx context.Context) logger.Logger { return m }
func (m *mockLogger) Sync() error                                   { return nil }
func (m *mockLogger) Close() error                                  { return nil }

// mockServer 测试用 mock server.
type mockServer struct {
	name       string
	addr       string
	started    atomic.Bool
	stopped    atomic.Bool
	startErr   error
	stopErr    error
	startDelay time.Duration
}

func (m *mockServer) Start(ctx context.Context) error {
	if m.startDelay > 0 {
		time.Sleep(m.startDelay)
	}
	if m.startErr != nil {
		return m.startErr
	}
	m.started.Store(true)
	<-ctx.Done()
	return nil
}

func (m *mockServer) Stop(ctx context.Context) error {
	m.stopped.Store(true)
	return m.stopErr
}

func (m *mockServer) Name() string { return m.name }
func (m *mockServer) Addr() string { return m.addr }

func TestNewApplication(t *testing.T) {
	t.Run("创建成功", func(t *testing.T) {
		log := newMockLogger()
		app := NewApplication(
			WithName("test-app"),
			WithVersion("1.0.0"),
			WithLogger(log),
		)

		if app.Name() != "test-app" {
			t.Errorf("expected name 'test-app', got '%s'", app.Name())
		}
		if app.Version() != "1.0.0" {
			t.Errorf("expected version '1.0.0', got '%s'", app.Version())
		}
	})

	t.Run("未设置logger时panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when logger not set")
			}
		}()
		NewApplication(WithName("test"))
	})
}

func TestApplication_Use(t *testing.T) {
	log := newMockLogger()
	app := NewApplication(WithLogger(log))

	srv1 := &mockServer{name: "srv1", addr: ":8080"}
	srv2 := &mockServer{name: "srv2", addr: ":9090"}

	app.Use(srv1, srv2)

	if len(app.servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(app.servers))
	}
}

func TestApplication_RunAndStop(t *testing.T) {
	log := newMockLogger()
	srv := &mockServer{name: "test", addr: ":8080"}

	app := NewApplication(
		WithName("test-app"),
		WithLogger(log),
		WithGracefulTimeout(1*time.Second),
	)
	app.Use(srv)

	// 启动后立即停止
	go func() {
		time.Sleep(100 * time.Millisecond)
		app.Stop()
	}()

	err := app.Run()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !srv.started.Load() {
		t.Error("server should be started")
	}
	if !srv.stopped.Load() {
		t.Error("server should be stopped")
	}
}

func TestApplication_RunWithNoServers(t *testing.T) {
	log := newMockLogger()
	app := NewApplication(
		WithName("test-app"),
		WithLogger(log),
	)

	go func() {
		time.Sleep(50 * time.Millisecond)
		app.Stop()
	}()

	err := app.Run()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// 应该有警告日志
	if len(log.warns) == 0 {
		t.Error("expected warning log for no servers")
	}
}

func TestApplication_RunAlreadyRunning(t *testing.T) {
	log := newMockLogger()
	app := NewApplication(
		WithName("test-app"),
		WithLogger(log),
	)

	// 模拟已运行状态
	app.running = true

	err := app.Run()
	if !errors.Is(err, ErrServerRunning) {
		t.Errorf("expected ErrServerRunning, got %v", err)
	}
}

func TestApplication_Context(t *testing.T) {
	log := newMockLogger()
	app := NewApplication(WithLogger(log))

	ctx := app.Context()
	if ctx == nil {
		t.Error("context should not be nil")
	}
}

func TestApplicationOptions(t *testing.T) {
	log := newMockLogger()

	t.Run("WithGracefulTimeout", func(t *testing.T) {
		app := NewApplication(
			WithLogger(log),
			WithGracefulTimeout(5*time.Second),
		)
		if app.opts.gracefulTimeout != 5*time.Second {
			t.Error("graceful timeout not set correctly")
		}
	})

	t.Run("WithSignals", func(t *testing.T) {
		app := NewApplication(
			WithLogger(log),
		)
		// 默认应该有信号处理
		if app.opts.signals != nil && len(app.opts.signals) > 0 {
			t.Error("signals should be nil by default (uses defaults in waitForShutdown)")
		}
	})
}
