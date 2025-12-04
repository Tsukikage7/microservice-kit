package logger

import "errors"

// 预定义错误常量.
var (
	// ErrCreateDir 创建日志目录失败.
	ErrCreateDir = errors.New("创建日志目录失败")

	// ErrOpenFile 打开日志文件失败.
	ErrOpenFile = errors.New("打开日志文件失败")
)
