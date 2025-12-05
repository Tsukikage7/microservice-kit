package transport

import (
	"context"
)

// Endpoint 表示单个 RPC 方法.
type Endpoint func(ctx context.Context, request any) (response any, err error)

// Middleware 是 Endpoint 中间件.
//
// 用于添加横切关注点（日志、认证、限流等）.
type Middleware func(Endpoint) Endpoint

// Chain 将多个中间件链接在一起.
//
// 执行顺序：第一个中间件最先执行.
func Chain(outer Middleware, others ...Middleware) Middleware {
	return func(next Endpoint) Endpoint {
		for i := len(others) - 1; i >= 0; i-- {
			next = others[i](next)
		}
		return outer(next)
	}
}

// Nop 是一个空的 Endpoint，什么都不做.
func Nop(context.Context, any) (any, error) { return struct{}{}, nil }

// NopMiddleware 是一个空的中间件，直接调用下一个 Endpoint.
func NopMiddleware(next Endpoint) Endpoint { return next }
