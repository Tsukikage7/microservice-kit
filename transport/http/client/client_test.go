package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Tsukikage7/microservice-kit/logger"
	"github.com/Tsukikage7/microservice-kit/transport"
)

// mockLogger 测试用 mock logger.
type mockLogger struct{}

func (m *mockLogger) Debug(args ...any)                 {}
func (m *mockLogger) Debugf(format string, args ...any) {}
func (m *mockLogger) Info(args ...any)                  {}
func (m *mockLogger) Infof(format string, args ...any)  {}
func (m *mockLogger) Warn(args ...any)                  {}
func (m *mockLogger) Warnf(format string, args ...any)  {}
func (m *mockLogger) Error(args ...any)                 {}
func (m *mockLogger) Errorf(format string, args ...any) {}
func (m *mockLogger) Fatal(args ...any)                 {}
func (m *mockLogger) Fatalf(format string, args ...any) {}
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
func (m *mockDiscovery) RegisterWithHealthEndpoint(ctx context.Context, serviceName, address, protocol string, healthEndpoint *transport.HealthEndpoint) (string, error) {
	return "", nil
}
func (m *mockDiscovery) Unregister(ctx context.Context, serviceID string) error { return nil }
func (m *mockDiscovery) Discover(ctx context.Context, serviceName string) ([]string, error) {
	return m.addrs, m.err
}
func (m *mockDiscovery) Close() error { return nil }

func TestNew(t *testing.T) {
	t.Run("创建成功", func(t *testing.T) {
		disc := &mockDiscovery{addrs: []string{"localhost:8080"}}
		client, err := New(
			WithName("test-client"),
			WithServiceName("test-service"),
			WithDiscovery(disc),
			WithLogger(&mockLogger{}),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if client.HTTPClient() == nil {
			t.Error("http client should not be nil")
		}
		if client.BaseURL() != "http://localhost:8080" {
			t.Errorf("expected baseURL 'http://localhost:8080', got '%s'", client.BaseURL())
		}
	})

	t.Run("未设置serviceName时panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when serviceName not set")
			}
		}()
		New(
			WithDiscovery(&mockDiscovery{addrs: []string{"localhost:8080"}}),
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
			WithDiscovery(&mockDiscovery{addrs: []string{"localhost:8080"}}),
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

func TestClient_HTTPMethods(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte("GET OK"))
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			w.Write([]byte("POST: " + string(body)))
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			w.Write([]byte("PUT: " + string(body)))
		case http.MethodDelete:
			w.Write([]byte("DELETE OK"))
		}
	}))
	defer server.Close()

	// 提取地址
	addr := strings.TrimPrefix(server.URL, "http://")
	disc := &mockDiscovery{addrs: []string{addr}}

	client, err := New(
		WithServiceName("test-service"),
		WithDiscovery(disc),
		WithLogger(&mockLogger{}),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		resp, err := client.Get(ctx, "/test")
		if err != nil {
			t.Fatalf("Get error: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "GET OK" {
			t.Errorf("expected 'GET OK', got '%s'", body)
		}
	})

	t.Run("Post", func(t *testing.T) {
		resp, err := client.Post(ctx, "/test", strings.NewReader("hello"))
		if err != nil {
			t.Fatalf("Post error: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "POST: hello" {
			t.Errorf("expected 'POST: hello', got '%s'", body)
		}
	})

	t.Run("Put", func(t *testing.T) {
		resp, err := client.Put(ctx, "/test", strings.NewReader("update"))
		if err != nil {
			t.Fatalf("Put error: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "PUT: update" {
			t.Errorf("expected 'PUT: update', got '%s'", body)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		resp, err := client.Delete(ctx, "/test")
		if err != nil {
			t.Fatalf("Delete error: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "DELETE OK" {
			t.Errorf("expected 'DELETE OK', got '%s'", body)
		}
	})
}

func TestClient_Headers(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	disc := &mockDiscovery{addrs: []string{addr}}

	client, err := New(
		WithServiceName("test-service"),
		WithDiscovery(disc),
		WithLogger(&mockLogger{}),
		WithHeader("X-Custom-Header", "custom-value"),
		WithHeaders(map[string]string{
			"X-Another-Header": "another-value",
		}),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.Get(context.Background(), "/test")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	resp.Body.Close()

	if receivedHeaders.Get("X-Custom-Header") != "custom-value" {
		t.Error("custom header not sent")
	}
	if receivedHeaders.Get("X-Another-Header") != "another-value" {
		t.Error("another header not sent")
	}
}

func TestClientOptions(t *testing.T) {
	t.Run("WithName", func(t *testing.T) {
		opts := defaultOptions()
		WithName("custom-name")(opts)
		if opts.name != "custom-name" {
			t.Errorf("expected name 'custom-name', got '%s'", opts.name)
		}
	})

	t.Run("WithScheme", func(t *testing.T) {
		opts := defaultOptions()
		WithScheme("https")(opts)
		if opts.scheme != "https" {
			t.Errorf("expected scheme 'https', got '%s'", opts.scheme)
		}
	})

	t.Run("WithTimeout", func(t *testing.T) {
		opts := defaultOptions()
		WithTimeout(10 * time.Second)(opts)
		if opts.timeout != 10*time.Second {
			t.Error("timeout not set correctly")
		}
	})

	t.Run("WithTransport", func(t *testing.T) {
		opts := defaultOptions()
		transport := &http.Transport{}
		WithTransport(transport)(opts)
		if opts.transport != transport {
			t.Error("transport not set correctly")
		}
	})

	t.Run("默认值", func(t *testing.T) {
		opts := defaultOptions()
		if opts.name != "HTTP-Client" {
			t.Errorf("expected default name 'HTTP-Client', got '%s'", opts.name)
		}
		if opts.scheme != "http" {
			t.Errorf("expected default scheme 'http', got '%s'", opts.scheme)
		}
		if opts.timeout != 30*time.Second {
			t.Errorf("expected default timeout 30s, got %v", opts.timeout)
		}
	})
}
