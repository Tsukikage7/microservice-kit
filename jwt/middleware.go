package jwt

import (
	"context"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// HTTPMiddleware 创建 HTTP 认证中间件.
func (j *JWT) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查白名单
		if j.IsWhitelisted(r.Context(), r) {
			next.ServeHTTP(w, r)
			return
		}

		// 提取令牌
		token, err := j.ExtractToken(r.Context(), r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// 验证令牌
		claims, err := j.Validate(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// 将 Claims 存入上下文
		if c, ok := claims.(Claims); ok {
			ctx := ContextWithClaims(r.Context(), c)
			ctx = ContextWithToken(ctx, token)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// HTTPMiddlewareFunc 创建 HTTP 认证中间件（函数形式）.
func (j *JWT) HTTPMiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return j.HTTPMiddleware(next).ServeHTTP
}

// UnaryServerInterceptor 创建 gRPC 一元拦截器.
func (j *JWT) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// 检查白名单
		if j.IsWhitelisted(ctx, req) {
			return handler(ctx, req)
		}

		// 提取令牌
		token, err := j.ExtractToken(ctx, req)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		// 验证令牌
		claims, err := j.Validate(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		// 将 Claims 存入上下文
		if c, ok := claims.(Claims); ok {
			ctx = ContextWithClaims(ctx, c)
			ctx = ContextWithToken(ctx, token)
		}

		return handler(ctx, req)
	}
}

// StreamServerInterceptor 创建 gRPC 流拦截器.
func (j *JWT) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()

		// 检查白名单
		if j.IsWhitelisted(ctx, nil) {
			return handler(srv, ss)
		}

		// 提取令牌
		token, err := j.ExtractToken(ctx, nil)
		if err != nil {
			return status.Error(codes.Unauthenticated, err.Error())
		}

		// 验证令牌
		claims, err := j.Validate(token)
		if err != nil {
			return status.Error(codes.Unauthenticated, err.Error())
		}

		// 创建带有 Claims 的包装流
		if c, ok := claims.(Claims); ok {
			ctx = ContextWithClaims(ctx, c)
			ctx = ContextWithToken(ctx, token)
			ss = &wrappedServerStream{ServerStream: ss, ctx: ctx}
		}

		return handler(srv, ss)
	}
}

// wrappedServerStream 包装的服务端流.
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context 返回包装的上下文.
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// ExtractToken 从请求中提取令牌（独立函数）.
func ExtractToken(ctx context.Context, req any) (string, error) {
	// 从 gRPC metadata 提取
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if authHeaders := md.Get("authorization"); len(authHeaders) > 0 {
			return extractTokenFromHeader(authHeaders[0]), nil
		}
	}

	// 从 HTTP 请求提取
	if httpReq, ok := req.(*http.Request); ok {
		if auth := httpReq.Header.Get("Authorization"); auth != "" {
			return extractTokenFromHeader(auth), nil
		}
	}

	// 从上下文提取
	if token, ok := TokenFromContext(ctx); ok {
		return token, nil
	}

	return "", ErrTokenNotFound
}

// extractTokenFromHeader 从 Authorization Header 提取令牌.
func extractTokenFromHeader(header string) string {
	// 移除 Bearer 前缀
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return strings.TrimSpace(header)
}
