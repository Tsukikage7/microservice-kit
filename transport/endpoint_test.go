package transport

import (
	"context"
	"errors"
	"testing"
)

func TestEndpoint(t *testing.T) {
	t.Run("基本调用", func(t *testing.T) {
		endpoint := func(ctx context.Context, req any) (any, error) {
			return req.(int) * 2, nil
		}

		resp, err := endpoint(context.Background(), 21)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.(int) != 42 {
			t.Errorf("expected 42, got %v", resp)
		}
	})

	t.Run("返回错误", func(t *testing.T) {
		testErr := errors.New("test error")
		endpoint := func(ctx context.Context, req any) (any, error) {
			return nil, testErr
		}

		_, err := endpoint(context.Background(), nil)
		if !errors.Is(err, testErr) {
			t.Errorf("expected testErr, got %v", err)
		}
	})
}

func TestMiddleware(t *testing.T) {
	t.Run("单个中间件", func(t *testing.T) {
		var order []string

		middleware := func(next Endpoint) Endpoint {
			return func(ctx context.Context, req any) (any, error) {
				order = append(order, "before")
				resp, err := next(ctx, req)
				order = append(order, "after")
				return resp, err
			}
		}

		endpoint := func(ctx context.Context, req any) (any, error) {
			order = append(order, "endpoint")
			return "ok", nil
		}

		wrapped := middleware(endpoint)
		_, err := wrapped(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{"before", "endpoint", "after"}
		if len(order) != len(expected) {
			t.Fatalf("expected %d calls, got %d", len(expected), len(order))
		}
		for i, v := range expected {
			if order[i] != v {
				t.Errorf("expected order[%d]=%s, got %s", i, v, order[i])
			}
		}
	})
}

func TestChain(t *testing.T) {
	t.Run("链式中间件执行顺序", func(t *testing.T) {
		var order []int

		makeMiddleware := func(id int) Middleware {
			return func(next Endpoint) Endpoint {
				return func(ctx context.Context, req any) (any, error) {
					order = append(order, id)
					return next(ctx, req)
				}
			}
		}

		endpoint := func(ctx context.Context, req any) (any, error) {
			order = append(order, 0)
			return nil, nil
		}

		// Chain(1, 2, 3) 应该按 1 -> 2 -> 3 -> endpoint 顺序执行
		chained := Chain(makeMiddleware(1), makeMiddleware(2), makeMiddleware(3))
		wrapped := chained(endpoint)

		_, err := wrapped(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []int{1, 2, 3, 0}
		if len(order) != len(expected) {
			t.Fatalf("expected %d calls, got %d", len(expected), len(order))
		}
		for i, v := range expected {
			if order[i] != v {
				t.Errorf("expected order[%d]=%d, got %d", i, v, order[i])
			}
		}
	})

	t.Run("单个中间件链", func(t *testing.T) {
		called := false
		middleware := func(next Endpoint) Endpoint {
			return func(ctx context.Context, req any) (any, error) {
				called = true
				return next(ctx, req)
			}
		}

		endpoint := func(ctx context.Context, req any) (any, error) {
			return "ok", nil
		}

		chained := Chain(middleware)
		wrapped := chained(endpoint)

		resp, err := wrapped(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("middleware not called")
		}
		if resp.(string) != "ok" {
			t.Errorf("expected 'ok', got %v", resp)
		}
	})
}

func TestNop(t *testing.T) {
	resp, err := Nop(context.Background(), nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("response should not be nil")
	}
}

func TestNopMiddleware(t *testing.T) {
	called := false
	endpoint := func(ctx context.Context, req any) (any, error) {
		called = true
		return "ok", nil
	}

	wrapped := NopMiddleware(endpoint)
	resp, err := wrapped(context.Background(), nil)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("endpoint not called")
	}
	if resp.(string) != "ok" {
		t.Errorf("expected 'ok', got %v", resp)
	}
}
