package domain

import "errors"

// 预定义错误.
var (
	// ErrNotFound 未找到错误.
	ErrNotFound = errors.New("未找到")
	// ErrConcurrencyConflict 并发冲突错误.
	ErrConcurrencyConflict = errors.New("并发冲突")
)
