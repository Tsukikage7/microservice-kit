package saga

import (
	"context"
	"sync"
	"time"
)

// Store Saga 状态存储接口.
type Store interface {
	// Save 保存 Saga 状态.
	Save(ctx context.Context, state *State) error

	// Get 获取 Saga 状态.
	Get(ctx context.Context, id string) (*State, error)

	// Delete 删除 Saga 状态.
	Delete(ctx context.Context, id string) error

	// List 列出指定状态的 Saga.
	List(ctx context.Context, status SagaStatus, limit int) ([]*State, error)
}

// MemoryStore 基于内存的状态存储.
//
// 适用于单机部署或测试场景.
type MemoryStore struct {
	mu      sync.RWMutex
	data    map[string]*State
	closeCh chan struct{}
}

// NewMemoryStore 创建内存存储.
func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{
		data:    make(map[string]*State),
		closeCh: make(chan struct{}),
	}

	// 启动清理协程
	go s.cleanup()

	return s
}

// Save 保存 Saga 状态.
func (s *MemoryStore) Save(ctx context.Context, state *State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 深拷贝
	copied := *state
	copied.StepResults = make([]StepResult, len(state.StepResults))
	copy(copied.StepResults, state.StepResults)

	if state.Data != nil {
		copied.Data = make(map[string]any, len(state.Data))
		for k, v := range state.Data {
			copied.Data[k] = v
		}
	}

	s.data[state.ID] = &copied
	return nil
}

// Get 获取 Saga 状态.
func (s *MemoryStore) Get(ctx context.Context, id string) (*State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.data[id]
	if !ok {
		return nil, ErrSagaNotFound
	}

	// 深拷贝返回
	copied := *state
	copied.StepResults = make([]StepResult, len(state.StepResults))
	copy(copied.StepResults, state.StepResults)

	return &copied, nil
}

// Delete 删除 Saga 状态.
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, id)
	return nil
}

// List 列出指定状态的 Saga.
func (s *MemoryStore) List(ctx context.Context, status SagaStatus, limit int) ([]*State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*State
	for _, state := range s.data {
		if state.Status == status {
			copied := *state
			result = append(result, &copied)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// Close 关闭存储.
func (s *MemoryStore) Close() error {
	close(s.closeCh)
	return nil
}

// cleanup 定期清理已完成的 Saga.
func (s *MemoryStore) cleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.closeCh:
			return
		case <-ticker.C:
			s.doCleanup()
		}
	}
}

func (s *MemoryStore) doCleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, state := range s.data {
		// 清理已完成超过24小时的 Saga
		if state.Status.IsTerminal() && state.CompletedAt != nil {
			if now.Sub(*state.CompletedAt) > 24*time.Hour {
				delete(s.data, id)
			}
		}
	}
}

// NopStore 空存储，不保存任何状态.
//
// 适用于不需要持久化状态的场景.
type NopStore struct{}

// NewNopStore 创建空存储.
func NewNopStore() *NopStore {
	return &NopStore{}
}

// Save 保存状态（空实现）.
func (s *NopStore) Save(ctx context.Context, state *State) error {
	return nil
}

// Get 获取状态（始终返回未找到）.
func (s *NopStore) Get(ctx context.Context, id string) (*State, error) {
	return nil, ErrSagaNotFound
}

// Delete 删除状态（空实现）.
func (s *NopStore) Delete(ctx context.Context, id string) error {
	return nil
}

// List 列出状态（返回空列表）.
func (s *NopStore) List(ctx context.Context, status SagaStatus, limit int) ([]*State, error) {
	return nil, nil
}
