package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func TestGRPCServer_Check(t *testing.T) {
	t.Run("empty service name - uses readiness", func(t *testing.T) {
		h := New(WithReadinessChecker(NewAlwaysUpChecker("test")))
		server := NewGRPCServer(h)

		resp, err := server.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{
			Service: "",
		})

		require.NoError(t, err)
		assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
	})

	t.Run("readiness service", func(t *testing.T) {
		h := New(WithReadinessChecker(NewAlwaysUpChecker("test")))
		server := NewGRPCServer(h)

		resp, err := server.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{
			Service: ServiceReadiness,
		})

		require.NoError(t, err)
		assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
	})

	t.Run("liveness service", func(t *testing.T) {
		h := New(WithLivenessChecker(NewAlwaysUpChecker("test")))
		server := NewGRPCServer(h)

		resp, err := server.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{
			Service: ServiceLiveness,
		})

		require.NoError(t, err)
		assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
	})

	t.Run("unhealthy returns NOT_SERVING", func(t *testing.T) {
		downChecker := NewCheckerFunc("down", func(ctx context.Context) CheckResult {
			return CheckResult{Status: StatusDown}
		})
		h := New(WithReadinessChecker(downChecker))
		server := NewGRPCServer(h)

		resp, err := server.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{
			Service: "",
		})

		require.NoError(t, err)
		assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, resp.Status)
	})

	t.Run("unknown status returns UNKNOWN", func(t *testing.T) {
		unknownChecker := NewCheckerFunc("unknown", func(ctx context.Context) CheckResult {
			return CheckResult{Status: StatusUnknown}
		})
		h := New(WithReadinessChecker(unknownChecker))
		server := NewGRPCServer(h)

		resp, err := server.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{
			Service: "",
		})

		require.NoError(t, err)
		assert.Equal(t, grpc_health_v1.HealthCheckResponse_UNKNOWN, resp.Status)
	})

	t.Run("unknown service returns NOT_FOUND", func(t *testing.T) {
		h := New()
		server := NewGRPCServer(h)

		_, err := server.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{
			Service: "unknown-service",
		})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})
}

func TestGRPCServer_SetServingStatus(t *testing.T) {
	h := New(WithReadinessChecker(NewAlwaysUpChecker("test")))
	server := NewGRPCServer(h)

	// 手动覆盖状态
	server.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	resp, err := server.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{
		Service: "",
	})

	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, resp.Status)
}

func TestGRPCServer_ClearServingStatus(t *testing.T) {
	h := New(WithReadinessChecker(NewAlwaysUpChecker("test")))
	server := NewGRPCServer(h)

	// 手动覆盖状态
	server.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// 清除覆盖
	server.ClearServingStatus("")

	resp, err := server.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{
		Service: "",
	})

	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
}

func TestGRPCServer_convertStatus(t *testing.T) {
	server := NewGRPCServer(New())

	tests := []struct {
		input    Status
		expected grpc_health_v1.HealthCheckResponse_ServingStatus
	}{
		{StatusUp, grpc_health_v1.HealthCheckResponse_SERVING},
		{StatusDown, grpc_health_v1.HealthCheckResponse_NOT_SERVING},
		{StatusUnknown, grpc_health_v1.HealthCheckResponse_UNKNOWN},
		{Status("invalid"), grpc_health_v1.HealthCheckResponse_UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := server.convertStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// mockWatchStream 模拟 Watch 流.
type mockWatchStream struct {
	grpc_health_v1.Health_WatchServer
	ctx      context.Context
	sent     []*grpc_health_v1.HealthCheckResponse
	sendErr  error
	cancelFn context.CancelFunc
}

func newMockWatchStream() *mockWatchStream {
	ctx, cancel := context.WithCancel(context.Background())
	return &mockWatchStream{
		ctx:      ctx,
		cancelFn: cancel,
	}
}

func (m *mockWatchStream) Send(resp *grpc_health_v1.HealthCheckResponse) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, resp)
	// 发送后立即取消，模拟客户端断开
	m.cancelFn()
	return nil
}

func (m *mockWatchStream) Context() context.Context {
	return m.ctx
}

func TestGRPCServer_Watch(t *testing.T) {
	h := New(WithReadinessChecker(NewAlwaysUpChecker("test")))
	server := NewGRPCServer(h)

	stream := newMockWatchStream()

	err := server.Watch(&grpc_health_v1.HealthCheckRequest{Service: ""}, stream)

	// Watch 返回 context.Canceled 是预期行为
	assert.ErrorIs(t, err, context.Canceled)
	require.Len(t, stream.sent, 1)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, stream.sent[0].Status)
}
