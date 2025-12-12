package cache

import "errors"

// 预定义错误常量.
var (
	// ErrNotFound 缓存键不存在.
	ErrNotFound = errors.New("缓存键不存在")

	// ErrLockNotHeld 锁未持有或已过期.
	ErrLockNotHeld = errors.New("锁未持有或已过期")

	// ErrNilConfig 缓存配置为空.
	ErrNilConfig = errors.New("缓存配置为空")

	// ErrEmptyAddr 缓存地址为空.
	ErrEmptyAddr = errors.New("缓存地址为空")

	// ErrUnsupported 不支持的缓存类型.
	ErrUnsupported = errors.New("不支持的缓存类型")

	// ErrNilLogger 日志记录器为空.
	ErrNilLogger = errors.New("日志记录器为空")

	// ErrNotInteger 值不是整数.
	ErrNotInteger = errors.New("值不是整数")

	// ErrSerialize 序列化值失败.
	ErrSerialize = errors.New("序列化值失败")

	// ErrConnect 连接失败.
	ErrConnect = errors.New("连接失败")
)
