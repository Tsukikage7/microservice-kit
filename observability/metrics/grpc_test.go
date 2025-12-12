package metrics

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryServerInterceptor(t *testing.T) {
	cfg := &Config{Namespace: "test"}
	collector, err := NewPrometheus(cfg)
	require.NoError(t, err)

	interceptor := UnaryServerInterceptor(collector)

	tests := []struct {
		name        string
		handlerErr  error
		expectError bool
	}{
		{"success", nil, false},
		{"error", errors.New("handler error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/Method",
			}

			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return "response", tt.handlerErr
			}

			resp, err := interceptor(context.Background(), "request", info, handler)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "response", resp)
			}
		})
	}

	// 验证指标被记录
	metricsHandler := collector.GetHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	metricsHandler.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	bodyStr := string(body)

	assert.Contains(t, bodyStr, "test_grpc_requests_total")
}

func TestStreamServerInterceptor(t *testing.T) {
	cfg := &Config{Namespace: "test"}
	collector, err := NewPrometheus(cfg)
	require.NoError(t, err)

	interceptor := StreamServerInterceptor(collector)

	tests := []struct {
		name        string
		handlerErr  error
		expectError bool
	}{
		{"success", nil, false},
		{"error", errors.New("handler error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &grpc.StreamServerInfo{
				FullMethod: "/test.Service/StreamMethod",
			}

			handler := func(srv interface{}, stream grpc.ServerStream) error {
				return tt.handlerErr
			}

			mockStream := &mockServerStream{ctx: context.Background()}
			err := interceptor(nil, mockStream, info, handler)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnaryClientInterceptor(t *testing.T) {
	cfg := &Config{Namespace: "test"}
	collector, err := NewPrometheus(cfg)
	require.NoError(t, err)

	interceptor := UnaryClientInterceptor(collector)

	tests := []struct {
		name        string
		invokerErr  error
		expectError bool
	}{
		{"success", nil, false},
		{"error", errors.New("invoker error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				return tt.invokerErr
			}

			var reply string
			err := interceptor(context.Background(), "/test.Service/Method", "request", &reply, nil, invoker)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStreamClientInterceptor(t *testing.T) {
	cfg := &Config{Namespace: "test"}
	collector, err := NewPrometheus(cfg)
	require.NoError(t, err)

	interceptor := StreamClientInterceptor(collector)

	t.Run("success", func(t *testing.T) {
		desc := &grpc.StreamDesc{}

		streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			return &mockClientStream{}, nil
		}

		stream, err := interceptor(context.Background(), desc, nil, "/test.Service/StreamMethod", streamer)

		assert.NoError(t, err)
		assert.NotNil(t, stream)
	})

	t.Run("error", func(t *testing.T) {
		desc := &grpc.StreamDesc{}

		streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			return nil, errors.New("streamer error")
		}

		stream, err := interceptor(context.Background(), desc, nil, "/test.Service/StreamMethod", streamer)

		assert.Error(t, err)
		assert.Nil(t, stream)
	})
}

// Mock implementations

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

type mockClientStream struct {
	grpc.ClientStream
}

func (m *mockClientStream) Context() context.Context {
	return context.Background()
}

func (m *mockClientStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockClientStream) RecvMsg(msg interface{}) error {
	return nil
}

func (m *mockClientStream) CloseSend() error {
	return nil
}

func (m *mockClientStream) Header() (metadata.MD, error) {
	return nil, nil
}

func (m *mockClientStream) Trailer() metadata.MD {
	return nil
}
