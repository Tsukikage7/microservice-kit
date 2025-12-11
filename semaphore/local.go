package semaphore

import (
	"context"
	"sync"
)

// Local 本地信号量.
//
// 基于 channel 实现，适用于单机并发控制.
type Local struct {
	size   int64
	sem    chan struct{}
	closed bool
	mu     sync.RWMutex
}

// NewLocal 创建本地信号量.
//
// size: 最大并发数
func NewLocal(size int64) *Local {
	if size <= 0 {
		panic("semaphore: 信号量大小必须为正数")
	}

	return &Local{
		size: size,
		sem:  make(chan struct{}, size),
	}
}

// Acquire 获取一个许可.
func (s *Local) Acquire(ctx context.Context) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return ErrClosed
	}
	s.mu.RUnlock()

	select {
	case s.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire 尝试获取一个许可.
func (s *Local) TryAcquire(ctx context.Context) bool {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return false
	}
	s.mu.RUnlock()

	select {
	case s.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release 释放一个许可.
func (s *Local) Release(ctx context.Context) error {
	select {
	case <-s.sem:
		return nil
	default:
		// 没有许可可释放，忽略（容错）
		return nil
	}
}

// Available 返回当前可用的许可数量.
func (s *Local) Available(ctx context.Context) (int64, error) {
	return s.size - int64(len(s.sem)), nil
}

// Size 返回信号量的总大小.
func (s *Local) Size() int64 {
	return s.size
}

// Close 关闭信号量.
func (s *Local) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		close(s.sem)
	}
	return nil
}

// WeightedLocal 加权本地信号量.
//
// 允许一次获取/释放多个许可.
type WeightedLocal struct {
	size      int64
	current   int64
	mu        sync.Mutex
	cond      *sync.Cond
	closed    bool
	waiters   int
}

// NewWeightedLocal 创建加权本地信号量.
func NewWeightedLocal(size int64) *WeightedLocal {
	if size <= 0 {
		panic("semaphore: 信号量大小必须为正数")
	}

	s := &WeightedLocal{
		size: size,
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

// Acquire 获取一个许可.
func (s *WeightedLocal) Acquire(ctx context.Context) error {
	return s.AcquireN(ctx, 1)
}

// AcquireN 获取 n 个许可.
func (s *WeightedLocal) AcquireN(ctx context.Context, n int64) error {
	if n <= 0 {
		return nil
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrClosed
	}

	// 如果有足够的许可，直接获取
	if s.current+n <= s.size {
		s.current += n
		s.mu.Unlock()
		return nil
	}

	// 等待许可
	s.waiters++
	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			s.cond.Broadcast()
		case <-done:
		}
	}()

	for s.current+n > s.size && !s.closed && ctx.Err() == nil {
		s.cond.Wait()
	}

	close(done)
	s.waiters--

	if s.closed {
		s.mu.Unlock()
		return ErrClosed
	}

	if ctx.Err() != nil {
		s.mu.Unlock()
		return ctx.Err()
	}

	s.current += n
	s.mu.Unlock()
	return nil
}

// TryAcquire 尝试获取一个许可.
func (s *WeightedLocal) TryAcquire(ctx context.Context) bool {
	return s.TryAcquireN(ctx, 1)
}

// TryAcquireN 尝试获取 n 个许可.
func (s *WeightedLocal) TryAcquireN(ctx context.Context, n int64) bool {
	if n <= 0 {
		return true
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return false
	}

	if s.current+n <= s.size {
		s.current += n
		return true
	}
	return false
}

// Release 释放一个许可.
func (s *WeightedLocal) Release(ctx context.Context) error {
	return s.ReleaseN(ctx, 1)
}

// ReleaseN 释放 n 个许可.
func (s *WeightedLocal) ReleaseN(ctx context.Context, n int64) error {
	if n <= 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.current -= n
	if s.current < 0 {
		s.current = 0
	}

	if s.waiters > 0 {
		s.cond.Broadcast()
	}
	return nil
}

// Available 返回当前可用的许可数量.
func (s *WeightedLocal) Available(ctx context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.size - s.current, nil
}

// Size 返回信号量的总大小.
func (s *WeightedLocal) Size() int64 {
	return s.size
}

// Close 关闭信号量.
func (s *WeightedLocal) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		s.cond.Broadcast()
	}
	return nil
}
