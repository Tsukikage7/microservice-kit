package cqrs

import "context"

// QueryHandler 查询处理器.
type QueryHandler[Q, R any] func(ctx context.Context, query Q) (R, error)

// QueryBus 查询总线.
type QueryBus[Q, R any] struct {
	handler QueryHandler[Q, R]
}

// NewQueryBus 创建查询总线.
func NewQueryBus[Q, R any](handler QueryHandler[Q, R]) *QueryBus[Q, R] {
	return &QueryBus[Q, R]{handler: handler}
}

// Dispatch 分发查询.
func (b *QueryBus[Q, R]) Dispatch(ctx context.Context, query Q) (R, error) {
	return b.handler(ctx, query)
}
