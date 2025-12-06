// Package cqrs 实现CQRS模式的命令和查询处理.
package cqrs

import "context"

// CommandHandler 命令处理器接口.
type CommandHandler[C, R any] interface {
	Handle(ctx context.Context, cmd C) (C, R, error)
}

// ApplyCommand 应用命令处理器.
func ApplyCommand[C, R any](ctx context.Context, cmd C, handler CommandHandler[C, R]) (C, R, error) {
	return handler.Handle(ctx, cmd)
}
