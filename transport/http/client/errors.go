package client

import "errors"

// 错误定义.
var (
	// ErrRequestFailed 请求创建失败.
	ErrRequestFailed = errors.New("http client: 请求创建失败")

	// ErrDiscoveryFailed 服务发现失败.
	ErrDiscoveryFailed = errors.New("http client: 服务发现失败")

	// ErrServiceNotFound 未找到服务实例.
	ErrServiceNotFound = errors.New("http client: 未找到服务实例")
)
