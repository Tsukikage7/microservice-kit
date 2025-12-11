// Package semaphore 提供信号量并发控制.
//
// 信号量用于限制对共享资源的并发访问数量，
// 支持本地和分布式两种模式。
//
// 本地信号量:
//
//	sem := semaphore.NewLocal(10) // 最多10个并发
//	if err := sem.Acquire(ctx); err != nil {
//	    return err
//	}
//	defer sem.Release()
//
// 分布式信号量:
//
//	sem := semaphore.NewRedis(redisClient, "api-limit", 100)
//	if err := sem.Acquire(ctx); err != nil {
//	    return err
//	}
//	defer sem.Release(ctx)
//
// 中间件:
//
//	endpoint = semaphore.EndpointMiddleware(sem)(endpoint)
package semaphore

import (
	"context"
)

// Semaphore 信号量接口.
type Semaphore interface {
	// Acquire 获取一个许可.
	// 如果没有可用许可，会阻塞等待直到获取成功或 context 取消.
	Acquire(ctx context.Context) error

	// TryAcquire 尝试获取一个许可.
	// 如果没有可用许可，立即返回 false.
	TryAcquire(ctx context.Context) bool

	// Release 释放一个许可.
	Release(ctx context.Context) error

	// Available 返回当前可用的许可数量.
	Available(ctx context.Context) (int64, error)

	// Size 返回信号量的总大小.
	Size() int64
}

// Weighted 加权信号量接口.
//
// 允许一次获取/释放多个许可.
type Weighted interface {
	Semaphore

	// AcquireN 获取 n 个许可.
	AcquireN(ctx context.Context, n int64) error

	// TryAcquireN 尝试获取 n 个许可.
	TryAcquireN(ctx context.Context, n int64) bool

	// ReleaseN 释放 n 个许可.
	ReleaseN(ctx context.Context, n int64) error
}
