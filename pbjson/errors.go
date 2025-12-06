package pbjson

import "errors"

// 错误定义.
var (
	ErrNotProtoMessage = errors.New("不是 proto.Message 类型")
)
