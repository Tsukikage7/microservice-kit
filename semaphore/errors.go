package semaphore

import "errors"

// 预定义错误.
var (
	// ErrNoPermit 无法获取许可.
	ErrNoPermit = errors.New("semaphore: 无法获取许可")

	// ErrTimeout 获取许可超时.
	ErrTimeout = errors.New("semaphore: 获取许可超时")

	// ErrClosed 信号量已关闭.
	ErrClosed = errors.New("semaphore: 信号量已关闭")

	// ErrInvalidSize 信号量大小无效.
	ErrInvalidSize = errors.New("semaphore: 大小必须大于0")

	// ErrNilCache 缓存为空.
	ErrNilCache = errors.New("semaphore: 缓存不能为空")
)
