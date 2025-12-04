// Package discovery 提供服务发现功能.
package discovery

import (
	"context"
	"errors"
)

// 常见错误.
var (
	ErrNilConfig           = errors.New("服务发现配置为空")
	ErrNilLogger           = errors.New("日志记录器为空")
	ErrEmptyAddr           = errors.New("服务发现地址为空")
	ErrEmptyName           = errors.New("服务名称为空")
	ErrEmptyAddress        = errors.New("服务地址为空")
	ErrEmptyServiceID      = errors.New("服务ID为空")
	ErrEmptyType           = errors.New("服务发现类型为空")
	ErrUnsupportedType     = errors.New("不支持的服务发现类型")
	ErrUnsupportedProtocol = errors.New("不支持的协议类型")
	ErrInvalidAddress      = errors.New("无效的地址格式")
	ErrInvalidPort         = errors.New("无效的端口号")
	ErrNotFound            = errors.New("未发现任何服务实例")
	ErrClientCreate        = errors.New("创建客户端失败")
	ErrRegister            = errors.New("注册服务失败")
	ErrUnregister          = errors.New("注销服务失败")
	ErrDiscover            = errors.New("发现服务失败")
)

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
