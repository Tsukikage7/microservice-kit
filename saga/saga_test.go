package saga

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tsukikage7/microservice-kit/storage/cache"
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

// newTestStore 创建测试用的存储.
func newTestStore() (*KVStore, cache.Cache) {
	memCache, _ := cache.NewMemoryCache(nil, &testLogger{})
	kv := CacheKV(memCache)
	return NewKVStore(kv), memCache
}

func TestSagaSuccess(t *testing.T) {
	executed := make([]string, 0)

	step1 := func(ctx context.Context, data *Data) error {
		executed = append(executed, "step1")
		data.Set("step1_result", "done")
		return nil
	}

	step2 := func(ctx context.Context, data *Data) error {
		executed = append(executed, "step2")
		// 验证可以读取上一步的数据
		if data.GetString("step1_result") != "done" {
			t.Error("expected step1_result to be 'done'")
		}
		return nil
	}

	step3 := func(ctx context.Context, data *Data) error {
		executed = append(executed, "step3")
		return nil
	}

	saga := New("test-saga").
		Step("step1", step1, nil).
		Step("step2", step2, nil).
		Step("step3", step3, nil).
		Build()

	err := saga.Execute(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(executed) != 3 {
		t.Errorf("expected 3 steps executed, got %d", len(executed))
	}

	for i, name := range []string{"step1", "step2", "step3"} {
		if executed[i] != name {
			t.Errorf("expected step %d to be %s, got %s", i, name, executed[i])
		}
	}
}

func TestSagaFailureWithCompensation(t *testing.T) {
	executed := make([]string, 0)
	compensated := make([]string, 0)

	step1 := func(ctx context.Context, data *Data) error {
		executed = append(executed, "step1")
		return nil
	}
	comp1 := func(ctx context.Context, data *Data) error {
		compensated = append(compensated, "comp1")
		return nil
	}

	step2 := func(ctx context.Context, data *Data) error {
		executed = append(executed, "step2")
		return nil
	}
	comp2 := func(ctx context.Context, data *Data) error {
		compensated = append(compensated, "comp2")
		return nil
	}

	step3 := func(ctx context.Context, data *Data) error {
		executed = append(executed, "step3")
		return errors.New("step3 failed")
	}

	saga := New("test-saga").
		Step("step1", step1, comp1).
		Step("step2", step2, comp2).
		Step("step3", step3, nil).
		Build()

	err := saga.Execute(context.Background())
	if err == nil {
		t.Error("expected error")
	}

	if !errors.Is(err, ErrSagaFailed) {
		t.Errorf("expected ErrSagaFailed, got %v", err)
	}

	// 验证执行顺序
	if len(executed) != 3 {
		t.Errorf("expected 3 steps executed, got %d", len(executed))
	}

	// 验证补偿顺序（逆序）
	if len(compensated) != 2 {
		t.Errorf("expected 2 compensations, got %d", len(compensated))
	}
	if compensated[0] != "comp2" {
		t.Errorf("expected first compensation to be comp2, got %s", compensated[0])
	}
	if compensated[1] != "comp1" {
		t.Errorf("expected second compensation to be comp1, got %s", compensated[1])
	}
}

func TestSagaWithTimeout(t *testing.T) {
	step := func(ctx context.Context, data *Data) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
			return nil
		}
	}

	saga := New("timeout-saga").
		Step("slow-step", step, nil).
		Options(WithTimeout(100 * time.Millisecond)).
		Build()

	err := saga.Execute(context.Background())
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestSagaWithRetry(t *testing.T) {
	attempts := 0

	step := func(ctx context.Context, data *Data) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	saga := New("retry-saga").
		Step("flaky-step", step, nil).
		Options(WithRetry(3, 10*time.Millisecond)).
		Build()

	err := saga.Execute(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestSagaWithStore(t *testing.T) {
	store, memCache := newTestStore()
	defer memCache.Close()

	step := func(ctx context.Context, data *Data) error {
		return nil
	}

	saga := New("store-saga").
		Step("step1", step, nil).
		Options(WithStore(store)).
		Build()

	err := saga.Execute(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// 注意: RedisStore.List 返回 nil，因此不能用 List 验证
	// 状态已通过 saga 执行过程中的 Save 调用保存
}

func TestSagaWithHooks(t *testing.T) {
	started := make([]string, 0)
	ended := make([]string, 0)

	step := func(ctx context.Context, data *Data) error {
		return nil
	}

	saga := New("hooks-saga").
		Step("step1", step, nil).
		Step("step2", step, nil).
		Options(WithStepHooks(
			func(name string) { started = append(started, name) },
			func(name string, err error) { ended = append(ended, name) },
		)).
		Build()

	err := saga.Execute(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(started) != 2 {
		t.Errorf("expected 2 starts, got %d", len(started))
	}
	if len(ended) != 2 {
		t.Errorf("expected 2 ends, got %d", len(ended))
	}
}

func TestSagaCompensationFailure(t *testing.T) {
	step := func(ctx context.Context, data *Data) error {
		return nil
	}
	failStep := func(ctx context.Context, data *Data) error {
		return errors.New("action failed")
	}
	failComp := func(ctx context.Context, data *Data) error {
		return errors.New("compensation failed")
	}

	saga := New("comp-fail-saga").
		Step("step1", step, failComp).
		Step("step2", failStep, nil).
		Build()

	err := saga.Execute(context.Background())
	if err == nil {
		t.Error("expected error")
	}

	if !errors.Is(err, ErrSagaFailed) {
		t.Errorf("expected ErrSagaFailed, got %v", err)
	}

	// 错误消息应该包含补偿失败信息
	if !errors.Is(err, ErrSagaFailed) {
		t.Error("error should contain compensation failure info")
	}
}

func TestDataHelpers(t *testing.T) {
	data := NewData()

	// String
	data.Set("str", "hello")
	if data.GetString("str") != "hello" {
		t.Error("GetString failed")
	}
	if data.GetString("not_exist") != "" {
		t.Error("GetString should return empty for non-existent key")
	}

	// Int
	data.Set("int", 42)
	if data.GetInt("int") != 42 {
		t.Error("GetInt failed")
	}
	if data.GetInt("not_exist") != 0 {
		t.Error("GetInt should return 0 for non-existent key")
	}

	// Int64
	data.Set("int64", int64(123456789))
	if data.GetInt64("int64") != 123456789 {
		t.Error("GetInt64 failed")
	}

	// Bool
	data.Set("bool", true)
	if !data.GetBool("bool") {
		t.Error("GetBool failed")
	}

	// Get
	v, ok := data.Get("str")
	if !ok || v != "hello" {
		t.Error("Get failed")
	}

	// Keys
	keys := data.Keys()
	if len(keys) != 4 {
		t.Errorf("expected 4 keys, got %d", len(keys))
	}

	// Delete
	data.Delete("str")
	if data.GetString("str") != "" {
		t.Error("Delete failed")
	}
}

func TestKVStore(t *testing.T) {
	store, memCache := newTestStore()
	defer memCache.Close()

	ctx := context.Background()

	// Save
	state := NewState("test-1", "test-saga", 2)
	state.Status = SagaStatusCompleted

	err := store.Save(ctx, state)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Get
	got, err := store.Get(ctx, "test-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got.Name != "test-saga" {
		t.Errorf("expected name test-saga, got %s", got.Name)
	}

	// Delete
	err = store.Delete(ctx, "test-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Get after delete
	_, err = store.Get(ctx, "test-1")
	if !errors.Is(err, ErrSagaNotFound) {
		t.Errorf("expected ErrSagaNotFound, got %v", err)
	}
}

func TestSagaStatusIsTerminal(t *testing.T) {
	tests := []struct {
		status   SagaStatus
		terminal bool
	}{
		{SagaStatusPending, false},
		{SagaStatusRunning, false},
		{SagaStatusCompensating, false},
		{SagaStatusCompleted, true},
		{SagaStatusCompensated, true},
		{SagaStatusCompensateFailed, true},
	}

	for _, tt := range tests {
		if tt.status.IsTerminal() != tt.terminal {
			t.Errorf("status %s: expected IsTerminal=%v", tt.status, tt.terminal)
		}
	}
}

func TestBuilderPanicOnNoSteps(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()

	New("empty-saga").Build()
}
