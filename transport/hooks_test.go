package transport

import (
	"context"
	"errors"
	"testing"
)

func TestHooksBuilder(t *testing.T) {
	t.Run("构建空钩子", func(t *testing.T) {
		hooks := NewHooks().Build()

		if hooks == nil {
			t.Fatal("hooks should not be nil")
		}
		if len(hooks.BeforeStart) != 0 {
			t.Error("BeforeStart should be empty")
		}
		if len(hooks.AfterStart) != 0 {
			t.Error("AfterStart should be empty")
		}
		if len(hooks.BeforeStop) != 0 {
			t.Error("BeforeStop should be empty")
		}
		if len(hooks.AfterStop) != 0 {
			t.Error("AfterStop should be empty")
		}
	})

	t.Run("链式添加钩子", func(t *testing.T) {
		hook := func(ctx context.Context) error { return nil }

		hooks := NewHooks().
			BeforeStart(hook).
			BeforeStart(hook).
			AfterStart(hook).
			BeforeStop(hook).
			AfterStop(hook).
			Build()

		if len(hooks.BeforeStart) != 2 {
			t.Errorf("expected 2 BeforeStart hooks, got %d", len(hooks.BeforeStart))
		}
		if len(hooks.AfterStart) != 1 {
			t.Errorf("expected 1 AfterStart hook, got %d", len(hooks.AfterStart))
		}
		if len(hooks.BeforeStop) != 1 {
			t.Errorf("expected 1 BeforeStop hook, got %d", len(hooks.BeforeStop))
		}
		if len(hooks.AfterStop) != 1 {
			t.Errorf("expected 1 AfterStop hook, got %d", len(hooks.AfterStop))
		}
	})
}

func TestHooks_Run(t *testing.T) {
	t.Run("按顺序执行钩子", func(t *testing.T) {
		var order []int

		hooks := NewHooks().
			BeforeStart(func(ctx context.Context) error {
				order = append(order, 1)
				return nil
			}).
			BeforeStart(func(ctx context.Context) error {
				order = append(order, 2)
				return nil
			}).
			BeforeStart(func(ctx context.Context) error {
				order = append(order, 3)
				return nil
			}).
			Build()

		err := hooks.runBeforeStart(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(order) != 3 {
			t.Fatalf("expected 3 hooks executed, got %d", len(order))
		}
		for i, v := range order {
			if v != i+1 {
				t.Errorf("expected order[%d]=%d, got %d", i, i+1, v)
			}
		}
	})

	t.Run("钩子返回错误时停止执行", func(t *testing.T) {
		var count int
		testErr := errors.New("test error")

		hooks := NewHooks().
			BeforeStart(func(ctx context.Context) error {
				count++
				return nil
			}).
			BeforeStart(func(ctx context.Context) error {
				count++
				return testErr
			}).
			BeforeStart(func(ctx context.Context) error {
				count++
				return nil
			}).
			Build()

		err := hooks.runBeforeStart(context.Background())
		if !errors.Is(err, testErr) {
			t.Errorf("expected testErr, got %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 hooks executed, got %d", count)
		}
	})

	t.Run("nil钩子安全执行", func(t *testing.T) {
		var hooks *Hooks

		err := hooks.runBeforeStart(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestHooks_AllMethods(t *testing.T) {
	var beforeStartCalled, afterStartCalled, beforeStopCalled, afterStopCalled bool

	hooks := NewHooks().
		BeforeStart(func(ctx context.Context) error {
			beforeStartCalled = true
			return nil
		}).
		AfterStart(func(ctx context.Context) error {
			afterStartCalled = true
			return nil
		}).
		BeforeStop(func(ctx context.Context) error {
			beforeStopCalled = true
			return nil
		}).
		AfterStop(func(ctx context.Context) error {
			afterStopCalled = true
			return nil
		}).
		Build()

	ctx := context.Background()

	hooks.runBeforeStart(ctx)
	hooks.runAfterStart(ctx)
	hooks.runBeforeStop(ctx)
	hooks.runAfterStop(ctx)

	if !beforeStartCalled {
		t.Error("BeforeStart hook not called")
	}
	if !afterStartCalled {
		t.Error("AfterStart hook not called")
	}
	if !beforeStopCalled {
		t.Error("BeforeStop hook not called")
	}
	if !afterStopCalled {
		t.Error("AfterStop hook not called")
	}
}
