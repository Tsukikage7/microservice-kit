package server

import "errors"

// 预定义错误.
var (
	// ErrServerClosed 服务器已关闭.
	ErrServerClosed = errors.New("server: server is closed")

	// ErrServerRunning 服务器正在运行.
	ErrServerRunning = errors.New("server: server is already running")

	// ErrNoServers 没有注册任何服务器.
	ErrNoServers = errors.New("server: no servers registered")

	// ErrAddrEmpty 地址为空.
	ErrAddrEmpty = errors.New("server: address is empty")

	// ErrNilHandler 处理器为空.
	ErrNilHandler = errors.New("server: handler is nil")
)
