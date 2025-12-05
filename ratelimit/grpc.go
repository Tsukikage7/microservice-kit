package ratelimit

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor 创建 gRPC 一元拦截器.
//
// 当请求被限流时返回 ResourceExhausted 错误.
func UnaryServerInterceptor(limiter Limiter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if !limiter.Allow(ctx) {
			return nil, status.Error(codes.ResourceExhausted, "请求过于频繁，请稍后重试")
		}
		return handler(ctx, req)
	}
}

// UnaryServerInterceptorWithWait 创建阻塞式 gRPC 一元拦截器.
func UnaryServerInterceptorWithWait(limiter Limiter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if err := limiter.Wait(ctx); err != nil {
			return nil, status.Error(codes.DeadlineExceeded, "请求超时")
		}
		return handler(ctx, req)
	}
}

// StreamServerInterceptor 创建 gRPC 流拦截器.
func StreamServerInterceptor(limiter Limiter) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if !limiter.Allow(ss.Context()) {
			return status.Error(codes.ResourceExhausted, "请求过于频繁，请稍后重试")
		}
		return handler(srv, ss)
	}
}

// GRPCKeyFunc 用于从 gRPC 上下文中提取限流键.
type GRPCKeyFunc func(ctx context.Context, info *grpc.UnaryServerInfo) string

// KeyedUnaryServerInterceptor 创建基于键的 gRPC 一元拦截器.
func KeyedUnaryServerInterceptor(keyFunc GRPCKeyFunc, getLimiter KeyedLimiterFunc) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		key := keyFunc(ctx, info)
		limiter := getLimiter(key)
		if limiter == nil {
			return handler(ctx, req)
		}
		if !limiter.Allow(ctx) {
			return nil, status.Error(codes.ResourceExhausted, "请求过于频繁，请稍后重试")
		}
		return handler(ctx, req)
	}
}

// PeerKeyFunc 返回基于客户端地址的键提取函数.
func PeerKeyFunc() GRPCKeyFunc {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) string {
		if p, ok := peer.FromContext(ctx); ok {
			return p.Addr.String()
		}
		return ""
	}
}

// MethodKeyFunc 返回基于方法名的键提取函数.
func MethodKeyFunc() GRPCKeyFunc {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) string {
		return info.FullMethod
	}
}

// MetadataKeyFunc 返回基于 metadata 字段的键提取函数.
func MetadataKeyFunc(key string) GRPCKeyFunc {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) string {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get(key); len(values) > 0 {
				return values[0]
			}
		}
		return ""
	}
}

// CompositeGRPCKeyFunc 组合多个键提取函数.
func CompositeGRPCKeyFunc(funcs ...GRPCKeyFunc) GRPCKeyFunc {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) string {
		var key string
		for _, f := range funcs {
			if key != "" {
				key += ":"
			}
			key += f(ctx, info)
		}
		return key
	}
}
