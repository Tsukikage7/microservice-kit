package idempotency

import (
	"context"
	"sync"
	"time"
)

// MemoryStore 基于内存的幂等性存储.
//
// 适用于单机部署或测试场景.
// 注意: 重启后数据会丢失.
type MemoryStore struct {
	mu      sync.RWMutex
	data    map[string]*memoryEntry
	locks   map[string]time.Time
	closeCh chan struct{}
}

type memoryEntry struct {
	result    *Result
	expiresAt time.Time
}

// NewMemoryStore 创建内存存储.
func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{
		data:    make(map[string]*memoryEntry),
		locks:   make(map[string]time.Time),
		closeCh: make(chan struct{}),
	}

	// 启动清理协程
	go s.cleanup()

	return s
}

// Get 获取幂等键对应的结果.
func (s *MemoryStore) Get(ctx context.Context, key string) (*Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.data[key]
	if !ok {
		return nil, nil
	}

	if time.Now().After(entry.expiresAt) {
		return nil, nil
	}

	return entry.result, nil
}

// Set 设置幂等键和结果.
func (s *MemoryStore) Set(ctx context.Context, key string, result *Result, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = &memoryEntry{
		result:    result,
		expiresAt: time.Now().Add(ttl),
	}

	// 删除锁
	delete(s.locks, key)

	return nil
}

// SetNX 仅在键不存在时设置（用于获取处理锁）.
func (s *MemoryStore) SetNX(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已有结果
	if entry, ok := s.data[key]; ok && time.Now().Before(entry.expiresAt) {
		return false, nil
	}

	// 检查是否有锁
	if lockTime, ok := s.locks[key]; ok && time.Now().Before(lockTime) {
		return false, nil
	}

	// 设置锁
	s.locks[key] = time.Now().Add(ttl)
	return true, nil
}

// Delete 删除幂等键.
func (s *MemoryStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	delete(s.locks, key)
	return nil
}

// Close 关闭存储.
func (s *MemoryStore) Close() error {
	close(s.closeCh)
	return nil
}

// cleanup 定期清理过期数据.
func (s *MemoryStore) cleanup() {
	ticker := time.NewTicker(time.Minute)
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

	// 清理过期数据
	for key, entry := range s.data {
		if now.After(entry.expiresAt) {
			delete(s.data, key)
		}
	}

	// 清理过期锁
	for key, lockTime := range s.locks {
		if now.After(lockTime) {
			delete(s.locks, key)
		}
	}
}
