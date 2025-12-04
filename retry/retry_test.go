package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDo_Success(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	err := Do(ctx, func() error {
		callCount++
		return nil
	}).Run()

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	err := Do(ctx, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("临时错误")
		}
		return nil
	}).Run()

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestDo_MaxAttemptsExceeded(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	err := Do(ctx, func() error {
		callCount++
		return errors.New("持续错误")
	}).WithMaxAttempts(3).WithDelay(10 * time.Millisecond).Run()

	assert.ErrorIs(t, err, ErrMaxAttempts)
	assert.Equal(t, 3, callCount)
}

func TestDo_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	// 立即取消
	cancel()

	err := Do(ctx, func() error {
		callCount++
		return errors.New("错误")
	}).Run()

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, callCount)
}

func TestDo_ContextCanceledDuringRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	err := Do(ctx, func() error {
		callCount++
		if callCount == 1 {
			cancel() // 第一次调用后取消
		}
		return errors.New("错误")
	}).WithMaxAttempts(5).WithDelay(100 * time.Millisecond).Run()

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, callCount)
}

func TestDo_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	callCount := 0

	err := Do(ctx, func() error {
		callCount++
		return errors.New("错误")
	}).WithMaxAttempts(10).WithDelay(100 * time.Millisecond).Run()

	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.GreaterOrEqual(t, callCount, 1)
}

func TestDo_ZeroAttempts(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	err := Do(ctx, func() error {
		callCount++
		return nil
	}).WithMaxAttempts(0).Run()

	// 0 次尝试意味着不会执行函数
	assert.ErrorIs(t, err, ErrMaxAttempts)
	assert.Equal(t, 0, callCount)
}

func TestDo_SingleAttempt(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	err := Do(ctx, func() error {
		callCount++
		return errors.New("错误")
	}).WithMaxAttempts(1).Run()

	assert.ErrorIs(t, err, ErrMaxAttempts)
	assert.Equal(t, 1, callCount)
}

func TestDo_ChainedOptions(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	start := time.Now()
	err := Do(ctx, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("错误")
		}
		return nil
	}).WithMaxAttempts(5).WithDelay(50 * time.Millisecond).Run()

	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
	// 2 次重试间隔，每次 50ms
	assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond)
}

func TestDo_DefaultConfig(t *testing.T) {
	// 验证默认配置值
	assert.Equal(t, 3, DefaultMaxAttempts)
	assert.Equal(t, 100*time.Millisecond, DefaultDelay)
}

func TestRetry_WithMaxAttempts(t *testing.T) {
	ctx := context.Background()
	r := Do(ctx, func() error { return nil })

	result := r.WithMaxAttempts(10)

	assert.Same(t, r, result) // 返回同一实例
	assert.Equal(t, 10, r.maxAttempts)
}

func TestRetry_WithDelay(t *testing.T) {
	ctx := context.Background()
	r := Do(ctx, func() error { return nil })

	result := r.WithDelay(500 * time.Millisecond)

	assert.Same(t, r, result) // 返回同一实例
	assert.Equal(t, 500*time.Millisecond, r.delay)
}
