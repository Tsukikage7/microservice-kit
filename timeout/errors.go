package timeout

import "errors"

// 预定义错误.
var (
	// ErrTimeout 请求超时.
	ErrTimeout = errors.New("timeout: 请求超时")

	// ErrInvalidTimeout 超时时间无效.
	ErrInvalidTimeout = errors.New("timeout: 超时时间必须大于0")

	// ErrNilLogger 日志记录器为空.
	ErrNilLogger = errors.New("timeout: 日志记录器不能为空")
)
