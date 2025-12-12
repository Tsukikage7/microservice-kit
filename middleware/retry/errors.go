package retry

import "errors"

// 预定义错误常量.
var (
	// ErrMaxAttempts 已达到最大重试次数.
	ErrMaxAttempts = errors.New("已达到最大重试次数")
)
