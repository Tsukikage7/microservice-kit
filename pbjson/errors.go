package pbjson

import "errors"

// 预定义错误.
var (
	// ErrNotProtoMessage 不是 proto.Message 类型.
	ErrNotProtoMessage = errors.New("不是 proto.Message 类型")
)
