package domain

import "errors"

// 错误定义.
var (
	ErrNotFound            = errors.New("未找到")
	ErrConcurrencyConflict = errors.New("并发冲突")
)
