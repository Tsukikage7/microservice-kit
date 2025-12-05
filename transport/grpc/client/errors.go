package client

import "errors"

// 错误定义.
var (
	// ErrConnectionFailed 创建连接失败.
	ErrConnectionFailed = errors.New("grpc client: 创建连接失败")

	// ErrDiscoveryFailed 服务发现失败.
	ErrDiscoveryFailed = errors.New("grpc client: 服务发现失败")

	// ErrServiceNotFound 未找到服务实例.
	ErrServiceNotFound = errors.New("grpc client: 未找到服务实例")
)
