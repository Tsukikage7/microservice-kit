package discovery

import "errors"

// 预定义错误常量.
var (
	// ErrNilConfig 服务发现配置为空.
	ErrNilConfig = errors.New("服务发现配置为空")

	// ErrNilLogger 日志记录器为空.
	ErrNilLogger = errors.New("日志记录器为空")

	// ErrEmptyAddr 服务发现地址为空.
	ErrEmptyAddr = errors.New("服务发现地址为空")

	// ErrEmptyName 服务名称为空.
	ErrEmptyName = errors.New("服务名称为空")

	// ErrEmptyAddress 服务地址为空.
	ErrEmptyAddress = errors.New("服务地址为空")

	// ErrEmptyServiceID 服务ID为空.
	ErrEmptyServiceID = errors.New("服务ID为空")

	// ErrEmptyType 服务发现类型为空.
	ErrEmptyType = errors.New("服务发现类型为空")

	// ErrUnsupportedType 不支持的服务发现类型.
	ErrUnsupportedType = errors.New("不支持的服务发现类型")

	// ErrUnsupportedProtocol 不支持的协议类型.
	ErrUnsupportedProtocol = errors.New("不支持的协议类型")

	// ErrInvalidAddress 无效的地址格式.
	ErrInvalidAddress = errors.New("无效的地址格式")

	// ErrInvalidPort 无效的端口号.
	ErrInvalidPort = errors.New("无效的端口号")

	// ErrNotFound 未发现任何服务实例.
	ErrNotFound = errors.New("未发现任何服务实例")

	// ErrClientCreate 创建客户端失败.
	ErrClientCreate = errors.New("创建客户端失败")

	// ErrRegister 注册服务失败.
	ErrRegister = errors.New("注册服务失败")

	// ErrUnregister 注销服务失败.
	ErrUnregister = errors.New("注销服务失败")

	// ErrDiscover 发现服务失败.
	ErrDiscover = errors.New("发现服务失败")
)
