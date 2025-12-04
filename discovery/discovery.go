// Package discovery 提供服务发现功能.
package discovery

import "context"

// Discovery 服务发现接口.
type Discovery interface {
	// Register 注册服务实例，返回服务ID
	Register(ctx context.Context, serviceName, address string) (string, error)

	// RegisterWithProtocol 根据协议注册服务实例，返回服务ID
	RegisterWithProtocol(ctx context.Context, serviceName, address, protocol string) (string, error)

	// Unregister 注销服务实例
	Unregister(ctx context.Context, serviceID string) error

	// Discover 发现服务实例
	Discover(ctx context.Context, serviceName string) ([]string, error)

	// Close 关闭服务发现连接
	Close() error
}
