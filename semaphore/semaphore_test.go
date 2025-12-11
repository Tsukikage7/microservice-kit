package semaphore

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestLocalSemaphore(t *testing.T) {
	sem := NewLocal(3)
	defer sem.Close()

	ctx := context.Background()

	t.Run("acquire and release", func(t *testing.T) {
		if err := sem.Acquire(ctx); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		available, _ := sem.Available(ctx)
		if available != 2 {
			t.Errorf("expected 2 available, got %d", available)
		}

		if err := sem.Release(ctx); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		available, _ = sem.Available(ctx)
		if available != 3 {
			t.Errorf("expected 3 available, got %d", available)
		}
	})

	t.Run("try acquire", func(t *testing.T) {
		// 获取所有许可
		for i := 0; i < 3; i++ {
			if !sem.TryAcquire(ctx) {
				t.Errorf("expected to acquire permit %d", i)
			}
		}

		// 第4个应该失败
		if sem.TryAcquire(ctx) {
			t.Error("expected to fail acquiring 4th permit")
		}

		// 释放所有
		for i := 0; i < 3; i++ {
			_ = sem.Release(ctx)
		}
	})

	t.Run("size", func(t *testing.T) {
		if sem.Size() != 3 {
			t.Errorf("expected size 3, got %d", sem.Size())
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		// 先占满
		for i := 0; i < 3; i++ {
			_ = sem.Acquire(ctx)
		}

		cancelCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		err := sem.Acquire(cancelCtx)
		if err != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", err)
		}

		// 释放
		for i := 0; i < 3; i++ {
			_ = sem.Release(ctx)
		}
	})
}

func TestWeightedLocalSemaphore(t *testing.T) {
	sem := NewWeightedLocal(10)
	defer sem.Close()

	ctx := context.Background()

	t.Run("acquire N", func(t *testing.T) {
		if err := sem.AcquireN(ctx, 5); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		available, _ := sem.Available(ctx)
		if available != 5 {
			t.Errorf("expected 5 available, got %d", available)
		}

		if err := sem.ReleaseN(ctx, 5); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("try acquire N", func(t *testing.T) {
		if !sem.TryAcquireN(ctx, 7) {
			t.Error("expected to acquire 7")
		}

		// 剩余3个，尝试获取5个应该失败
		if sem.TryAcquireN(ctx, 5) {
			t.Error("expected to fail acquiring 5")
		}

		// 但3个应该可以
		if !sem.TryAcquireN(ctx, 3) {
			t.Error("expected to acquire 3")
		}

		// 释放
		_ = sem.ReleaseN(ctx, 10)
	})
}

func TestLocalConcurrency(t *testing.T) {
	sem := NewLocal(5)
	defer sem.Close()

	ctx := context.Background()
	var maxConcurrent int32
	var current int32
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := sem.Acquire(ctx); err != nil {
				return
			}
			defer sem.Release(ctx)

			c := atomic.AddInt32(&current, 1)
			for {
				max := atomic.LoadInt32(&maxConcurrent)
				if c <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, c) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&current, -1)
		}()
	}

	wg.Wait()

	if maxConcurrent > 5 {
		t.Errorf("max concurrent exceeded limit: %d > 5", maxConcurrent)
	}
}

func TestEndpointMiddleware(t *testing.T) {
	sem := NewLocal(2)
	defer sem.Close()

	var callCount int32
	endpoint := func(ctx context.Context, request any) (any, error) {
		atomic.AddInt32(&callCount, 1)
		time.Sleep(50 * time.Millisecond)
		return "ok", nil
	}

	wrapped := EndpointMiddleware(sem)(endpoint)

	ctx := context.Background()
	var wg sync.WaitGroup
	var errors int32

	// 启动5个并发请求
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := wrapped(ctx, nil)
			if err != nil {
				atomic.AddInt32(&errors, 1)
			}
		}()
	}

	wg.Wait()

	// 应该有3个失败（因为只允许2个并发）
	if errors != 3 {
		t.Errorf("expected 3 errors, got %d", errors)
	}
}

func TestEndpointMiddlewareWithBlock(t *testing.T) {
	sem := NewLocal(2)
	defer sem.Close()

	var callCount int32
	endpoint := func(ctx context.Context, request any) (any, error) {
		atomic.AddInt32(&callCount, 1)
		time.Sleep(50 * time.Millisecond)
		return "ok", nil
	}

	wrapped := EndpointMiddleware(sem, WithBlock(true))(endpoint)

	ctx := context.Background()
	var wg sync.WaitGroup

	// 启动5个并发请求
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = wrapped(ctx, nil)
		}()
	}

	wg.Wait()

	// 所有请求应该都成功（因为会阻塞等待）
	if callCount != 5 {
		t.Errorf("expected 5 calls, got %d", callCount)
	}
}

func TestHTTPMiddleware(t *testing.T) {
	sem := NewLocal(1)
	defer sem.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	wrapped := HTTPMiddleware(sem)(handler)

	// 第一个请求
	req1 := httptest.NewRequest("GET", "/", nil)
	rec1 := httptest.NewRecorder()

	// 第二个请求
	req2 := httptest.NewRequest("GET", "/", nil)
	rec2 := httptest.NewRecorder()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		wrapped.ServeHTTP(rec1, req1)
	}()

	time.Sleep(10 * time.Millisecond) // 确保第一个请求先开始

	go func() {
		defer wg.Done()
		wrapped.ServeHTTP(rec2, req2)
	}()

	wg.Wait()

	// 第一个应该成功
	if rec1.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec1.Code)
	}

	// 第二个应该被拒绝
	if rec2.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec2.Code)
	}
}

func TestUnaryServerInterceptor(t *testing.T) {
	sem := NewLocal(1)
	defer sem.Close()

	handler := func(ctx context.Context, req any) (any, error) {
		time.Sleep(100 * time.Millisecond)
		return "ok", nil
	}

	interceptor := UnaryServerInterceptor(sem)
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}

	var wg sync.WaitGroup
	var results [2]error

	wg.Add(2)

	go func() {
		defer wg.Done()
		_, results[0] = interceptor(context.Background(), nil, info, handler)
	}()

	time.Sleep(10 * time.Millisecond)

	go func() {
		defer wg.Done()
		_, results[1] = interceptor(context.Background(), nil, info, handler)
	}()

	wg.Wait()

	// 一个成功，一个失败
	successCount := 0
	for _, err := range results {
		if err == nil {
			successCount++
		}
	}
	if successCount != 1 {
		t.Errorf("expected 1 success, got %d", successCount)
	}
}

func TestPanicOnInvalidSize(t *testing.T) {
	t.Run("local", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		NewLocal(0)
	})

	t.Run("weighted local", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		NewWeightedLocal(-1)
	})
}

func TestClosedSemaphore(t *testing.T) {
	sem := NewLocal(1)
	sem.Close()

	ctx := context.Background()

	if err := sem.Acquire(ctx); err != ErrClosed {
		t.Errorf("expected ErrClosed, got %v", err)
	}

	if sem.TryAcquire(ctx) {
		t.Error("expected TryAcquire to return false on closed semaphore")
	}
}
