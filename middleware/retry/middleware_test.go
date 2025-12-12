package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEndpointMiddleware(t *testing.T) {
	t.Run("成功不重试", func(t *testing.T) {
		callCount := 0
		endpoint := func(ctx context.Context, req any) (any, error) {
			callCount++
			return "success", nil
		}

		cfg := DefaultConfig()
		wrapped := EndpointMiddleware(cfg)(endpoint)

		resp, err := wrapped(context.Background(), nil)
		if err != nil {
			t.Errorf("不期望错误: %v", err)
		}
		if resp != "success" {
			t.Error("期望返回 success")
		}
		if callCount != 1 {
			t.Errorf("期望调用 1 次，实际 %d 次", callCount)
		}
	})

	t.Run("失败后重试成功", func(t *testing.T) {
		callCount := 0
		endpoint := func(ctx context.Context, req any) (any, error) {
			callCount++
			if callCount < 3 {
				return nil, errors.New("temporary error")
			}
			return "success", nil
		}

		cfg := &Config{
			MaxAttempts: 5,
			Delay:       1 * time.Millisecond,
			Backoff:     FixedBackoff,
			Retryable:   AlwaysRetry,
		}
		wrapped := EndpointMiddleware(cfg)(endpoint)

		resp, err := wrapped(context.Background(), nil)
		if err != nil {
			t.Errorf("不期望错误: %v", err)
		}
		if resp != "success" {
			t.Error("期望返回 success")
		}
		if callCount != 3 {
			t.Errorf("期望调用 3 次，实际 %d 次", callCount)
		}
	})

	t.Run("达到最大重试次数", func(t *testing.T) {
		callCount := 0
		endpoint := func(ctx context.Context, req any) (any, error) {
			callCount++
			return nil, errors.New("persistent error")
		}

		cfg := &Config{
			MaxAttempts: 3,
			Delay:       1 * time.Millisecond,
			Backoff:     FixedBackoff,
			Retryable:   AlwaysRetry,
		}
		wrapped := EndpointMiddleware(cfg)(endpoint)

		_, err := wrapped(context.Background(), nil)
		if err == nil {
			t.Error("期望错误")
		}
		if callCount != 3 {
			t.Errorf("期望调用 3 次，实际 %d 次", callCount)
		}
	})

	t.Run("不可重试错误", func(t *testing.T) {
		callCount := 0
		notRetryableErr := errors.New("not retryable")
		endpoint := func(ctx context.Context, req any) (any, error) {
			callCount++
			return nil, notRetryableErr
		}

		cfg := &Config{
			MaxAttempts: 5,
			Delay:       1 * time.Millisecond,
			Backoff:     FixedBackoff,
			Retryable: func(err error) bool {
				return !errors.Is(err, notRetryableErr)
			},
		}
		wrapped := EndpointMiddleware(cfg)(endpoint)

		_, err := wrapped(context.Background(), nil)
		if !errors.Is(err, notRetryableErr) {
			t.Errorf("期望 notRetryableErr，得到 %v", err)
		}
		if callCount != 1 {
			t.Errorf("期望调用 1 次，实际 %d 次", callCount)
		}
	})

	t.Run("上下文取消", func(t *testing.T) {
		callCount := 0
		endpoint := func(ctx context.Context, req any) (any, error) {
			callCount++
			return nil, errors.New("error")
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 立即取消

		cfg := &Config{
			MaxAttempts: 5,
			Delay:       time.Second,
			Backoff:     FixedBackoff,
			Retryable:   AlwaysRetry,
		}
		wrapped := EndpointMiddleware(cfg)(endpoint)

		_, err := wrapped(ctx, nil)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("期望 context.Canceled，得到 %v", err)
		}
	})
}

func TestBackoffFunctions(t *testing.T) {
	delay := 100 * time.Millisecond

	t.Run("FixedBackoff", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			result := FixedBackoff(i, delay)
			if result != delay {
				t.Errorf("attempt %d: 期望 %v，得到 %v", i, delay, result)
			}
		}
	})

	t.Run("ExponentialBackoff", func(t *testing.T) {
		expected := []time.Duration{
			100 * time.Millisecond,  // 2^0 * 100ms
			200 * time.Millisecond,  // 2^1 * 100ms
			400 * time.Millisecond,  // 2^2 * 100ms
			800 * time.Millisecond,  // 2^3 * 100ms
			1600 * time.Millisecond, // 2^4 * 100ms
		}
		for i, exp := range expected {
			result := ExponentialBackoff(i, delay)
			if result != exp {
				t.Errorf("attempt %d: 期望 %v，得到 %v", i, exp, result)
			}
		}
	})

	t.Run("LinearBackoff", func(t *testing.T) {
		expected := []time.Duration{
			100 * time.Millisecond, // 1 * 100ms
			200 * time.Millisecond, // 2 * 100ms
			300 * time.Millisecond, // 3 * 100ms
			400 * time.Millisecond, // 4 * 100ms
			500 * time.Millisecond, // 5 * 100ms
		}
		for i, exp := range expected {
			result := LinearBackoff(i, delay)
			if result != exp {
				t.Errorf("attempt %d: 期望 %v，得到 %v", i, exp, result)
			}
		}
	})
}

func TestRetryableFunctions(t *testing.T) {
	t.Run("AlwaysRetry", func(t *testing.T) {
		if !AlwaysRetry(errors.New("any error")) {
			t.Error("AlwaysRetry 应该返回 true")
		}
	})

	t.Run("NeverRetry", func(t *testing.T) {
		if NeverRetry(errors.New("any error")) {
			t.Error("NeverRetry 应该返回 false")
		}
	})
}

func TestEndpointRetrier(t *testing.T) {
	t.Run("链式配置", func(t *testing.T) {
		callCount := 0
		endpoint := func(ctx context.Context, req any) (any, error) {
			callCount++
			if callCount < 2 {
				return nil, errors.New("error")
			}
			return "success", nil
		}

		retrier := NewEndpointRetrier(nil).
			WithMaxAttempts(5).
			WithDelay(1 * time.Millisecond).
			WithBackoff(FixedBackoff).
			WithRetryable(AlwaysRetry)

		wrapped := retrier.Middleware()(endpoint)
		resp, err := wrapped(context.Background(), nil)

		if err != nil {
			t.Errorf("不期望错误: %v", err)
		}
		if resp != "success" {
			t.Error("期望返回 success")
		}
		if callCount != 2 {
			t.Errorf("期望调用 2 次，实际 %d 次", callCount)
		}
	})
}
