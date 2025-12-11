package lock

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Tsukikage7/microservice-kit/cache"
	"github.com/Tsukikage7/microservice-kit/logger"
)

// testLogger 用于测试的模拟日志器.
type testLogger struct{}

func (m *testLogger) Debug(args ...any)                             {}
func (m *testLogger) Debugf(format string, args ...any)             {}
func (m *testLogger) Info(args ...any)                              {}
func (m *testLogger) Infof(format string, args ...any)              {}
func (m *testLogger) Warn(args ...any)                              {}
func (m *testLogger) Warnf(format string, args ...any)              {}
func (m *testLogger) Error(args ...any)                             {}
func (m *testLogger) Errorf(format string, args ...any)             {}
func (m *testLogger) Fatal(args ...any)                             {}
func (m *testLogger) Fatalf(format string, args ...any)             {}
func (m *testLogger) Panic(args ...any)                             {}
func (m *testLogger) Panicf(format string, args ...any)             {}
func (m *testLogger) With(fields ...logger.Field) logger.Logger     { return m }
func (m *testLogger) WithContext(ctx context.Context) logger.Logger { return m }
func (m *testLogger) Sync() error                                   { return nil }
func (m *testLogger) Close() error                                  { return nil }

// newTestLocker 创建测试用的锁.
func newTestLocker(opts ...RedisOption) (*Redis, cache.Cache) {
	memCache, _ := cache.NewMemoryCache(nil, &testLogger{})
	return NewRedis(memCache, opts...), memCache
}

func TestTryLock(t *testing.T) {
	locker, memCache := newTestLocker()
	defer memCache.Close()

	ctx := context.Background()

	t.Run("acquire lock", func(t *testing.T) {
		acquired, err := locker.TryLock(ctx, "test-key", time.Minute)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !acquired {
			t.Error("expected to acquire lock")
		}

		// 验证锁已持有
		if !locker.IsHeld("test-key") {
			t.Error("expected lock to be held")
		}

		// 释放
		_ = locker.Unlock(ctx, "test-key")
	})

	t.Run("lock already held", func(t *testing.T) {
		// 获取锁
		acquired, _ := locker.TryLock(ctx, "held-key", time.Minute)
		if !acquired {
			t.Fatal("failed to acquire lock")
		}

		// 创建另一个 locker 尝试获取同一个锁
		locker2, _ := newTestLocker()
		locker2.cache = locker.cache // 共享缓存

		acquired2, err := locker2.TryLock(ctx, "held-key", time.Minute)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if acquired2 {
			t.Error("expected lock to fail (already held)")
		}

		// 释放
		_ = locker.Unlock(ctx, "held-key")
	})
}

func TestLock(t *testing.T) {
	locker, memCache := newTestLocker(WithRetryWait(10 * time.Millisecond))
	defer memCache.Close()

	ctx := context.Background()

	t.Run("blocking acquire", func(t *testing.T) {
		err := locker.Lock(ctx, "blocking-key", time.Minute)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// 释放
		_ = locker.Unlock(ctx, "blocking-key")
	})

	t.Run("context cancellation", func(t *testing.T) {
		// 先获取锁
		_, _ = locker.TryLock(ctx, "cancel-key", time.Minute)

		// 创建另一个 locker
		locker2, _ := newTestLocker(WithRetryWait(10 * time.Millisecond))
		locker2.cache = locker.cache

		cancelCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		err := locker2.Lock(cancelCtx, "cancel-key", time.Minute)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected DeadlineExceeded, got %v", err)
		}

		// 释放
		_ = locker.Unlock(ctx, "cancel-key")
	})

	t.Run("max retries", func(t *testing.T) {
		// 先获取锁
		_, _ = locker.TryLock(ctx, "retry-key", time.Minute)

		// 创建有最大重试次数的 locker
		locker2, _ := newTestLocker(
			WithRetryWait(10*time.Millisecond),
			WithMaxRetries(3),
		)
		locker2.cache = locker.cache

		err := locker2.Lock(ctx, "retry-key", time.Minute)
		if !errors.Is(err, ErrLockNotAcquired) {
			t.Errorf("expected ErrLockNotAcquired, got %v", err)
		}

		// 释放
		_ = locker.Unlock(ctx, "retry-key")
	})
}

func TestUnlock(t *testing.T) {
	locker, memCache := newTestLocker()
	defer memCache.Close()

	ctx := context.Background()

	t.Run("successful unlock", func(t *testing.T) {
		_, _ = locker.TryLock(ctx, "unlock-key", time.Minute)

		err := locker.Unlock(ctx, "unlock-key")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// 验证锁已释放
		if locker.IsHeld("unlock-key") {
			t.Error("expected lock to be released")
		}
	})

	t.Run("unlock not held", func(t *testing.T) {
		err := locker.Unlock(ctx, "not-held-key")
		if !errors.Is(err, ErrLockNotHeld) {
			t.Errorf("expected ErrLockNotHeld, got %v", err)
		}
	})
}

func TestExtend(t *testing.T) {
	locker, memCache := newTestLocker()
	defer memCache.Close()

	ctx := context.Background()

	t.Run("successful extend", func(t *testing.T) {
		_, _ = locker.TryLock(ctx, "extend-key", time.Minute)

		err := locker.Extend(ctx, "extend-key", 2*time.Minute)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// 释放
		_ = locker.Unlock(ctx, "extend-key")
	})

	t.Run("extend not held", func(t *testing.T) {
		err := locker.Extend(ctx, "not-held-key", time.Minute)
		if !errors.Is(err, ErrLockNotHeld) {
			t.Errorf("expected ErrLockNotHeld, got %v", err)
		}
	})
}

func TestWithLock(t *testing.T) {
	locker, memCache := newTestLocker()
	defer memCache.Close()

	ctx := context.Background()

	t.Run("successful execution", func(t *testing.T) {
		executed := false

		err := WithLock(ctx, locker, "with-lock-key", time.Minute, func() error {
			executed = true
			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !executed {
			t.Error("expected function to be executed")
		}

		// 验证锁已释放
		if locker.IsHeld("with-lock-key") {
			t.Error("expected lock to be released")
		}
	})

	t.Run("function error", func(t *testing.T) {
		expectedErr := errors.New("test error")

		err := WithLock(ctx, locker, "error-key", time.Minute, func() error {
			return expectedErr
		})

		if !errors.Is(err, expectedErr) {
			t.Errorf("expected test error, got %v", err)
		}

		// 验证锁已释放（即使函数返回错误）
		if locker.IsHeld("error-key") {
			t.Error("expected lock to be released after error")
		}
	})
}

func TestTryWithLock(t *testing.T) {
	locker, memCache := newTestLocker()
	defer memCache.Close()

	ctx := context.Background()

	t.Run("successful execution", func(t *testing.T) {
		executed := false

		err := TryWithLock(ctx, locker, "try-with-key", time.Minute, func() error {
			executed = true
			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !executed {
			t.Error("expected function to be executed")
		}
	})

	t.Run("lock not available", func(t *testing.T) {
		// 先获取锁
		_, _ = locker.TryLock(ctx, "occupied-key", time.Minute)

		// 创建另一个 locker
		locker2, _ := newTestLocker()
		locker2.cache = locker.cache

		executed := false
		err := TryWithLock(ctx, locker2, "occupied-key", time.Minute, func() error {
			executed = true
			return nil
		})

		if !errors.Is(err, ErrLockNotAcquired) {
			t.Errorf("expected ErrLockNotAcquired, got %v", err)
		}
		if executed {
			t.Error("expected function to not be executed")
		}

		// 释放
		_ = locker.Unlock(ctx, "occupied-key")
	})
}

func TestConcurrency(t *testing.T) {
	locker, memCache := newTestLocker(WithRetryWait(5 * time.Millisecond))
	defer memCache.Close()

	ctx := context.Background()
	var counter int32
	var maxConcurrent int32
	var current int32
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := WithLock(ctx, locker, "concurrent-key", time.Minute, func() error {
				c := atomic.AddInt32(&current, 1)
				for {
					max := atomic.LoadInt32(&maxConcurrent)
					if c <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, c) {
						break
					}
				}

				atomic.AddInt32(&counter, 1)
				time.Sleep(5 * time.Millisecond)

				atomic.AddInt32(&current, -1)
				return nil
			})

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()

	// 验证所有操作都执行了
	if counter != 50 {
		t.Errorf("expected 50 executions, got %d", counter)
	}

	// 验证同一时间只有一个在执行（互斥）
	if maxConcurrent > 1 {
		t.Errorf("expected max concurrent 1, got %d", maxConcurrent)
	}
}

func TestOptions(t *testing.T) {
	t.Run("custom owner ID", func(t *testing.T) {
		locker, memCache := newTestLocker(WithOwnerID("custom-owner"))
		defer memCache.Close()

		if locker.OwnerID() != "custom-owner" {
			t.Errorf("expected owner ID 'custom-owner', got %s", locker.OwnerID())
		}
	})

	t.Run("custom key prefix", func(t *testing.T) {
		locker, memCache := newTestLocker(WithKeyPrefix("myapp:lock:"))
		defer memCache.Close()

		ctx := context.Background()
		_, _ = locker.TryLock(ctx, "test", time.Minute)

		// 验证键前缀
		exists, _ := locker.cache.Exists(ctx, "myapp:lock:test")
		if !exists {
			t.Error("expected key with custom prefix to exist")
		}

		_ = locker.Unlock(ctx, "test")
	})
}

func TestPanicOnNilCache(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()

	NewRedis(nil)
}
