package server

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/Tsukikage7/microservice-kit/logger"
	"google.golang.org/grpc"
)

// mockLogger 测试用 mock logger.
type mockLogger struct{}

func (m *mockLogger) Debug(args ...any)                             {}
func (m *mockLogger) Debugf(format string, args ...any)             {}
func (m *mockLogger) Info(args ...any)                              {}
func (m *mockLogger) Infof(format string, args ...any)              {}
func (m *mockLogger) Warn(args ...any)                              {}
func (m *mockLogger) Warnf(format string, args ...any)              {}
func (m *mockLogger) Error(args ...any)                             {}
func (m *mockLogger) Errorf(format string, args ...any)             {}
func (m *mockLogger) Fatal(args ...any)                             {}
func (m *mockLogger) Fatalf(format string, args ...any)             {}
func (m *mockLogger) Panic(args ...any)                             {}
func (m *mockLogger) Panicf(format string, args ...any)             {}
func (m *mockLogger) With(fields ...logger.Field) logger.Logger     { return m }
func (m *mockLogger) WithContext(ctx context.Context) logger.Logger { return m }
func (m *mockLogger) Sync() error                                   { return nil }
func (m *mockLogger) Close() error                                  { return nil }

// mockRegistrar 测试用 mock registrar.
type mockRegistrar struct {
	registered bool
}

func (m *mockRegistrar) RegisterGRPC(server *grpc.Server) {
	m.registered = true
}

func getAvailablePort(t *testing.T) string {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

func TestNew(t *testing.T) {
	t.Run("创建成功", func(t *testing.T) {
		srv := New(
			WithName("test-grpc"),
			WithAddr(":9090"),
			WithLogger(&mockLogger{}),
		)

		if srv.Name() != "test-grpc" {
			t.Errorf("expected name 'test-grpc', got '%s'", srv.Name())
		}
		if srv.Addr() != ":9090" {
			t.Errorf("expected addr ':9090', got '%s'", srv.Addr())
		}
	})

	t.Run("未设置logger时panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when logger not set")
			}
		}()
		New(WithAddr(":9090"))
	})

	t.Run("默认值", func(t *testing.T) {
		srv := New(WithLogger(&mockLogger{}))

		if srv.Name() != "gRPC" {
			t.Errorf("expected default name 'gRPC', got '%s'", srv.Name())
		}
		if srv.Addr() != ":9090" {
			t.Errorf("expected default addr ':9090', got '%s'", srv.Addr())
		}
	})
}

func TestServer_Register(t *testing.T) {
	srv := New(WithLogger(&mockLogger{}))

	reg1 := &mockRegistrar{}
	reg2 := &mockRegistrar{}

	result := srv.Register(reg1, reg2)

	// 验证链式调用返回自身
	if result != srv {
		t.Error("Register should return server for chaining")
	}

	if len(srv.opts.services) != 2 {
		t.Errorf("expected 2 services, got %d", len(srv.opts.services))
	}
}

func TestServer_StartAndStop(t *testing.T) {
	addr := getAvailablePort(t)
	srv := New(
		WithAddr(addr),
		WithLogger(&mockLogger{}),
		WithReflection(true),
	)

	reg := &mockRegistrar{}
	srv.Register(reg)

	ctx, cancel := context.WithCancel(context.Background())

	// 启动服务器
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 验证服务已注册
	if !reg.registered {
		t.Error("service should be registered")
	}

	// 验证 GRPCServer 不为 nil
	if srv.GRPCServer() == nil {
		t.Error("GRPCServer should not be nil after start")
	}

	// 停止服务器
	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	if err := srv.Stop(stopCtx); err != nil {
		t.Errorf("unexpected stop error: %v", err)
	}

	// 等待 Start 返回
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("unexpected start error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for server to stop")
	}
}

func TestServer_StopNotStarted(t *testing.T) {
	srv := New(WithLogger(&mockLogger{}))

	err := srv.Stop(context.Background())
	if err != nil {
		t.Errorf("stopping non-started server should not error: %v", err)
	}
}

func TestServer_GRPCServerBeforeStart(t *testing.T) {
	srv := New(WithLogger(&mockLogger{}))

	if srv.GRPCServer() != nil {
		t.Error("GRPCServer should be nil before start")
	}
}

func TestServerOptions(t *testing.T) {
	t.Run("WithReflection", func(t *testing.T) {
		srv := New(
			WithLogger(&mockLogger{}),
			WithReflection(false),
		)
		if srv.opts.enableReflection {
			t.Error("reflection should be disabled")
		}
	})

	t.Run("WithKeepalive", func(t *testing.T) {
		srv := New(
			WithLogger(&mockLogger{}),
			WithKeepalive(30*time.Second, 10*time.Second),
		)
		if srv.opts.keepaliveTime != 30*time.Second {
			t.Error("keepalive time not set correctly")
		}
		if srv.opts.keepaliveTimeout != 10*time.Second {
			t.Error("keepalive timeout not set correctly")
		}
	})

	t.Run("WithUnaryInterceptor", func(t *testing.T) {
		interceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
		srv := New(
			WithLogger(&mockLogger{}),
			WithUnaryInterceptor(interceptor),
		)
		if len(srv.opts.unaryInterceptors) != 1 {
			t.Error("unary interceptor not added")
		}
	})

	t.Run("WithStreamInterceptor", func(t *testing.T) {
		interceptor := func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
		srv := New(
			WithLogger(&mockLogger{}),
			WithStreamInterceptor(interceptor),
		)
		if len(srv.opts.streamInterceptors) != 1 {
			t.Error("stream interceptor not added")
		}
	})

	t.Run("WithServerOption", func(t *testing.T) {
		srv := New(
			WithLogger(&mockLogger{}),
			WithServerOption(grpc.MaxRecvMsgSize(1024)),
		)
		if len(srv.opts.serverOptions) != 1 {
			t.Error("server option not added")
		}
	})
}
