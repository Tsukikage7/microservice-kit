package cqrs

import "context"

// QueryHandler 查询处理器接口.
type QueryHandler[Q, R any] interface {
	Handle(ctx context.Context, query Q) (R, error)
}

// ApplyQueryHandler 应用查询处理器.
func ApplyQueryHandler[Q, R any](ctx context.Context, query Q, handler QueryHandler[Q, R]) (R, error) {
	return handler.Handle(ctx, query)
}
