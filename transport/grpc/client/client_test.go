package client

import (
	"context"
	"errors"
	"testing"

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

// mockDiscovery 测试用 mock discovery.
type mockDiscovery struct {
	addrs []string
	err   error
}

func (m *mockDiscovery) Register(ctx context.Context, serviceName, address string) (string, error) {
	return "", nil
}
func (m *mockDiscovery) RegisterWithProtocol(ctx context.Context, serviceName, address, protocol string) (string, error) {
	return "", nil
}
func (m *mockDiscovery) Unregister(ctx context.Context, serviceID string) error { return nil }
func (m *mockDiscovery) Discover(ctx context.Context, serviceName string) ([]string, error) {
	return m.addrs, m.err
}
func (m *mockDiscovery) Close() error { return nil }

func TestNew(t *testing.T) {
	t.Run("创建成功", func(t *testing.T) {
		disc := &mockDiscovery{addrs: []string{"localhost:9090"}}
		client, err := New(
			WithName("test-client"),
			WithServiceName("test-service"),
			WithDiscovery(disc),
			WithLogger(&mockLogger{}),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer client.Close()

		if client.Conn() == nil {
			t.Error("connection should not be nil")
		}
	})

	t.Run("未设置serviceName时panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when serviceName not set")
			}
		}()
		New(
			WithDiscovery(&mockDiscovery{addrs: []string{"localhost:9090"}}),
			WithLogger(&mockLogger{}),
		)
	})

	t.Run("未设置discovery时panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when discovery not set")
			}
		}()
		New(
			WithServiceName("test-service"),
			WithLogger(&mockLogger{}),
		)
	})

	t.Run("未设置logger时panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when logger not set")
			}
		}()
		New(
			WithServiceName("test-service"),
			WithDiscovery(&mockDiscovery{addrs: []string{"localhost:9090"}}),
		)
	})

	t.Run("服务发现失败", func(t *testing.T) {
		disc := &mockDiscovery{err: errors.New("discovery error")}
		_, err := New(
			WithServiceName("test-service"),
			WithDiscovery(disc),
			WithLogger(&mockLogger{}),
		)
		if err == nil {
			t.Error("expected error when discovery fails")
		}
		if !errors.Is(err, ErrDiscoveryFailed) {
			t.Errorf("expected ErrDiscoveryFailed, got %v", err)
		}
	})

	t.Run("未找到服务实例", func(t *testing.T) {
		disc := &mockDiscovery{addrs: []string{}}
		_, err := New(
			WithServiceName("test-service"),
			WithDiscovery(disc),
			WithLogger(&mockLogger{}),
		)
		if err == nil {
			t.Error("expected error when no service found")
		}
		if !errors.Is(err, ErrServiceNotFound) {
			t.Errorf("expected ErrServiceNotFound, got %v", err)
		}
	})
}

func TestClient_Close(t *testing.T) {
	t.Run("关闭连接", func(t *testing.T) {
		disc := &mockDiscovery{addrs: []string{"localhost:9090"}}
		client, err := New(
			WithServiceName("test-service"),
			WithDiscovery(disc),
			WithLogger(&mockLogger{}),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = client.Close()
		if err != nil {
			t.Errorf("unexpected close error: %v", err)
		}
	})

	t.Run("关闭nil连接", func(t *testing.T) {
		client := &Client{opts: &options{logger: &mockLogger{}, name: "test", serviceName: "test"}}
		err := client.Close()
		if err != nil {
			t.Errorf("closing nil connection should not error: %v", err)
		}
	})
}

func TestClientOptions(t *testing.T) {
	t.Run("WithName", func(t *testing.T) {
		opts := defaultOptions()
		WithName("custom-name")(opts)
		if opts.name != "custom-name" {
			t.Errorf("expected name 'custom-name', got '%s'", opts.name)
		}
	})

	t.Run("WithInterceptors", func(t *testing.T) {
		opts := defaultOptions()
		interceptor := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, callOpts ...grpc.CallOption) error {
			return invoker(ctx, method, req, reply, cc, callOpts...)
		}
		WithInterceptors(interceptor)(opts)
		if len(opts.interceptors) != 1 {
			t.Error("interceptor not added")
		}
	})

	t.Run("WithDialOptions", func(t *testing.T) {
		opts := defaultOptions()
		WithDialOptions(grpc.WithAuthority("test"))(opts)
		if len(opts.dialOptions) != 1 {
			t.Error("dial option not added")
		}
	})

	t.Run("默认值", func(t *testing.T) {
		opts := defaultOptions()
		if opts.name != "gRPC-Client" {
			t.Errorf("expected default name 'gRPC-Client', got '%s'", opts.name)
		}
	})
}
