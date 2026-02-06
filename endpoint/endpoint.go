// Package endpoint 提供端点抽象和中间件链.
package endpoint

import "context"

// Endpoint 表示单个 RPC 方法.
type Endpoint func(ctx context.Context, request any) (response any, err error)

// Middleware 是 Endpoint 中间件.
type Middleware func(Endpoint) Endpoint

// Chain 将多个中间件链接在一起.
func Chain(outer Middleware, others ...Middleware) Middleware {
	return func(next Endpoint) Endpoint {
		for i := len(others) - 1; i >= 0; i-- {
			next = others[i](next)
		}
		return outer(next)
	}
}

// Nop 是一个空的 Endpoint.
func Nop(context.Context, any) (any, error) { return struct{}{}, nil }

// NopMiddleware 是一个空的中间件.
func NopMiddleware(next Endpoint) Endpoint { return next }
