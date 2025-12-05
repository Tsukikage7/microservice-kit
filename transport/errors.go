package transport

import "errors"

// 预定义错误.
var (
	// ErrServerRunning 服务器正在运行.
	ErrServerRunning = errors.New("transport: 服务器正在运行")
)
