package config

import "errors"

// 预定义错误常量.
var (
	// ErrNilConfig 配置为空.
	ErrNilConfig = errors.New("配置为空")

	// ErrFileNotFound 配置文件不存在.
	ErrFileNotFound = errors.New("配置文件不存在")

	// ErrInvalidType 不支持的配置文件类型.
	ErrInvalidType = errors.New("不支持的配置文件类型")

	// ErrReadConfig 读取配置失败.
	ErrReadConfig = errors.New("读取配置失败")

	// ErrUnmarshal 解析配置失败.
	ErrUnmarshal = errors.New("解析配置失败")

	// ErrValidation 配置验证失败.
	ErrValidation = errors.New("配置验证失败")
)
