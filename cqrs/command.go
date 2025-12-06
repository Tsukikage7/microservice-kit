package cqrs

import "context"

// CommandHandler 命令处理器.
type CommandHandler[C any] func(ctx context.Context, cmd C) error

// CommandBus 命令总线.
type CommandBus[C any] struct {
	handler CommandHandler[C]
}

// NewCommandBus 创建命令总线.
func NewCommandBus[C any](handler CommandHandler[C]) *CommandBus[C] {
	return &CommandBus[C]{handler: handler}
}

// Dispatch 分发命令.
func (b *CommandBus[C]) Dispatch(ctx context.Context, cmd C) error {
	return b.handler(ctx, cmd)
}
