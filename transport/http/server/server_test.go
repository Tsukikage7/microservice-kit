package server

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/Tsukikage7/microservice-kit/logger"
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

func getAvailablePort(t *testing.T) string {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

func TestNew(t *testing.T) {
	mux := http.NewServeMux()

	t.Run("创建成功", func(t *testing.T) {
		srv := New(mux,
			Name("test-http"),
			Addr(":8080"),
			Logger(&mockLogger{}),
		)

		if srv.Name() != "test-http" {
			t.Errorf("expected name 'test-http', got '%s'", srv.Name())
		}
		if srv.Addr() != ":8080" {
			t.Errorf("expected addr ':8080', got '%s'", srv.Addr())
		}
		// Handler() 返回包装后的 handler（包含健康检查中间件）
		if srv.Handler() == nil {
			t.Error("handler should not be nil")
		}
		// 验证健康检查管理器已创建
		if srv.Health() == nil {
			t.Error("health manager should not be nil")
		}
	})

	t.Run("未设置logger时panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when logger not set")
			}
		}()
		New(mux, Addr(":8080"))
	})

	t.Run("默认值", func(t *testing.T) {
		srv := New(mux, Logger(&mockLogger{}))

		if srv.Name() != "HTTP" {
			t.Errorf("expected default name 'HTTP', got '%s'", srv.Name())
		}
		if srv.Addr() != ":8080" {
			t.Errorf("expected default addr ':8080', got '%s'", srv.Addr())
		}
	})
}

func TestServer_StartAndStop(t *testing.T) {
	addr := getAvailablePort(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := New(mux,
		Addr(addr),
		Logger(&mockLogger{}),
	)

	ctx, cancel := context.WithCancel(context.Background())

	// 启动服务器
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 验证服务器可访问
	resp, err := http.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("failed to reach server: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
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
	mux := http.NewServeMux()
	srv := New(mux, Logger(&mockLogger{}))

	err := srv.Stop(context.Background())
	if err != nil {
		t.Errorf("stopping non-started server should not error: %v", err)
	}
}

func TestServerOptions(t *testing.T) {
	mux := http.NewServeMux()

	t.Run("ReadTimeout", func(t *testing.T) {
		srv := New(mux,
			Logger(&mockLogger{}),
			Timeout(10*time.Second, 0, 0),
		)
		if srv.opts.readTimeout != 10*time.Second {
			t.Error("read timeout not set correctly")
		}
	})

	t.Run("WriteTimeout", func(t *testing.T) {
		srv := New(mux,
			Logger(&mockLogger{}),
			Timeout(0, 15*time.Second, 0),
		)
		if srv.opts.writeTimeout != 15*time.Second {
			t.Error("write timeout not set correctly")
		}
	})

	t.Run("IdleTimeout", func(t *testing.T) {
		srv := New(mux,
			Logger(&mockLogger{}),
			Timeout(0, 0, 60*time.Second),
		)
		if srv.opts.idleTimeout != 60*time.Second {
			t.Error("idle timeout not set correctly")
		}
	})

	t.Run("默认超时值", func(t *testing.T) {
		srv := New(mux, Logger(&mockLogger{}))

		if srv.opts.readTimeout != 30*time.Second {
			t.Errorf("expected default read timeout 30s, got %v", srv.opts.readTimeout)
		}
		if srv.opts.writeTimeout != 30*time.Second {
			t.Errorf("expected default write timeout 30s, got %v", srv.opts.writeTimeout)
		}
		if srv.opts.idleTimeout != 120*time.Second {
			t.Errorf("expected default idle timeout 120s, got %v", srv.opts.idleTimeout)
		}
	})
}
