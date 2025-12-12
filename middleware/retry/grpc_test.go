package retry

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryClientInterceptor(t *testing.T) {
	t.Run("成功不重试", func(t *testing.T) {
		callCount := 0
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			callCount++
			return nil
		}

		cfg := DefaultConfig()
		interceptor := UnaryClientInterceptor(cfg)

		err := interceptor(context.Background(), "/test/Method", nil, nil, nil, invoker)
		if err != nil {
			t.Errorf("不期望错误: %v", err)
		}
		if callCount != 1 {
			t.Errorf("期望调用 1 次，实际 %d 次", callCount)
		}
	})

	t.Run("重试后成功", func(t *testing.T) {
		callCount := 0
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			callCount++
			if callCount < 3 {
				return status.Error(codes.Unavailable, "service unavailable")
			}
			return nil
		}

		cfg := &Config{
			MaxAttempts: 5,
			Delay:       1 * time.Millisecond,
			Backoff:     FixedBackoff,
		}
		interceptor := UnaryClientInterceptor(cfg)

		err := interceptor(context.Background(), "/test/Method", nil, nil, nil, invoker)
		if err != nil {
			t.Errorf("不期望错误: %v", err)
		}
		if callCount != 3 {
			t.Errorf("期望调用 3 次，实际 %d 次", callCount)
		}
	})

	t.Run("不可重试错误", func(t *testing.T) {
		callCount := 0
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			callCount++
			return status.Error(codes.InvalidArgument, "invalid argument")
		}

		cfg := &Config{
			MaxAttempts: 5,
			Delay:       1 * time.Millisecond,
			Backoff:     FixedBackoff,
		}
		interceptor := UnaryClientInterceptor(cfg)

		err := interceptor(context.Background(), "/test/Method", nil, nil, nil, invoker)
		if err == nil {
			t.Error("期望错误")
		}
		if callCount != 1 {
			t.Errorf("期望调用 1 次，实际 %d 次", callCount)
		}
	})
}

func TestStreamClientInterceptor(t *testing.T) {
	t.Run("成功不重试", func(t *testing.T) {
		callCount := 0
		streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			callCount++
			return nil, nil
		}

		cfg := DefaultConfig()
		interceptor := StreamClientInterceptor(cfg)

		_, err := interceptor(context.Background(), nil, nil, "/test/Method", streamer)
		if err != nil {
			t.Errorf("不期望错误: %v", err)
		}
		if callCount != 1 {
			t.Errorf("期望调用 1 次，实际 %d 次", callCount)
		}
	})

	t.Run("重试后成功", func(t *testing.T) {
		callCount := 0
		streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			callCount++
			if callCount < 2 {
				return nil, status.Error(codes.Unavailable, "service unavailable")
			}
			return nil, nil
		}

		cfg := &Config{
			MaxAttempts: 5,
			Delay:       1 * time.Millisecond,
			Backoff:     FixedBackoff,
		}
		interceptor := StreamClientInterceptor(cfg)

		_, err := interceptor(context.Background(), nil, nil, "/test/Method", streamer)
		if err != nil {
			t.Errorf("不期望错误: %v", err)
		}
		if callCount != 2 {
			t.Errorf("期望调用 2 次，实际 %d 次", callCount)
		}
	})
}

func TestDefaultGRPCRetryable(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"Unavailable", status.Error(codes.Unavailable, ""), true},
		{"ResourceExhausted", status.Error(codes.ResourceExhausted, ""), true},
		{"Aborted", status.Error(codes.Aborted, ""), true},
		{"DeadlineExceeded", status.Error(codes.DeadlineExceeded, ""), true},
		{"InvalidArgument", status.Error(codes.InvalidArgument, ""), false},
		{"NotFound", status.Error(codes.NotFound, ""), false},
		{"PermissionDenied", status.Error(codes.PermissionDenied, ""), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := DefaultGRPCRetryable(tc.err)
			if result != tc.expected {
				t.Errorf("期望 %v，得到 %v", tc.expected, result)
			}
		})
	}
}

func TestRetryableCodesFunc(t *testing.T) {
	retryable := RetryableCodesFunc(codes.Unavailable, codes.ResourceExhausted)

	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{"Unavailable", status.Error(codes.Unavailable, ""), true},
		{"ResourceExhausted", status.Error(codes.ResourceExhausted, ""), true},
		{"InvalidArgument", status.Error(codes.InvalidArgument, ""), false},
		{"nil error", nil, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := retryable(tc.err)
			if result != tc.expected {
				t.Errorf("期望 %v，得到 %v", tc.expected, result)
			}
		})
	}
}

func TestGRPCRetrier(t *testing.T) {
	t.Run("链式配置", func(t *testing.T) {
		retrier := NewGRPCRetrier(nil).
			WithMaxAttempts(5).
			WithDelay(1 * time.Millisecond).
			WithBackoff(ExponentialBackoff).
			WithRetryableCodes(codes.Unavailable)

		if retrier.cfg.MaxAttempts != 5 {
			t.Errorf("期望 MaxAttempts=5，得到 %d", retrier.cfg.MaxAttempts)
		}
		if retrier.cfg.Delay != 1*time.Millisecond {
			t.Errorf("期望 Delay=1ms，得到 %v", retrier.cfg.Delay)
		}
	})
}
