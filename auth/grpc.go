package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// gRPC 相关常量.
const (
	// GRPCAuthorizationMetadata gRPC Authorization 元数据键.
	GRPCAuthorizationMetadata = "authorization"

	// GRPCAPIKeyMetadata gRPC API Key 元数据键.
	GRPCAPIKeyMetadata = "x-api-key"
)

// UnaryServerInterceptor 返回 gRPC 一元服务器认证拦截器.
//
// 示例:
//
//	authenticator := authjwt.NewAuthenticator(jwtService)
//	srv := grpc.NewServer(
//	    grpc.ChainUnaryInterceptor(
//	        auth.UnaryServerInterceptor(authenticator),
//	    ),
//	)
func UnaryServerInterceptor(authenticator Authenticator, opts ...Option) grpc.UnaryServerInterceptor {
	if authenticator == nil {
		panic("auth: 认证器不能为空")
	}

	o := defaultOptions(authenticator)
	for _, opt := range opts {
		opt(o)
	}

	// 设置默认的 gRPC 凭据提取器
	if o.credentialsExtractor == nil {
		o.credentialsExtractor = DefaultGRPCCredentialsExtractor
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// 检查是否跳过
		if o.skipper != nil && o.skipper(ctx, req) {
			return handler(ctx, req)
		}

		// 提取凭据
		creds, err := o.credentialsExtractor(ctx, req)
		if err != nil {
			if o.logger != nil {
				o.logger.WithContext(ctx).Debug(
					"[Auth] gRPC凭据提取失败",
					logger.String("method", info.FullMethod),
					logger.Err(err),
				)
			}
			return nil, status.Error(codes.Unauthenticated, "credentials not found")
		}

		// 认证
		principal, err := authenticator.Authenticate(ctx, *creds)
		if err != nil {
			if o.logger != nil {
				o.logger.WithContext(ctx).Warn(
					"[Auth] gRPC认证失败",
					logger.String("method", info.FullMethod),
					logger.Err(err),
				)
			}
			return nil, status.Error(codes.Unauthenticated, "authentication failed")
		}

		// 将主体存入 context
		ctx = WithPrincipal(ctx, principal)

		// 授权
		if o.authorizer != nil {
			action := o.action
			resource := o.resource

			// 如果没有指定，使用方法名
			if action == "" {
				action = extractGRPCAction(info.FullMethod)
			}
			if resource == "" {
				resource = extractGRPCResource(info.FullMethod)
			}

			if err := o.authorizer.Authorize(ctx, principal, action, resource); err != nil {
				if o.logger != nil {
					o.logger.WithContext(ctx).Warn(
						"[Auth] gRPC授权失败",
						logger.String("principal_id", principal.ID),
						logger.String("method", info.FullMethod),
						logger.Err(err),
					)
				}
				return nil, status.Error(codes.PermissionDenied, "permission denied")
			}
		}

		if o.logger != nil {
			o.logger.WithContext(ctx).Debug(
				"[Auth] gRPC认证成功",
				logger.String("principal_id", principal.ID),
				logger.String("method", info.FullMethod),
			)
		}

		return handler(ctx, req)
	}
}

// StreamServerInterceptor 返回 gRPC 流服务器认证拦截器.
func StreamServerInterceptor(authenticator Authenticator, opts ...Option) grpc.StreamServerInterceptor {
	if authenticator == nil {
		panic("auth: 认证器不能为空")
	}

	o := defaultOptions(authenticator)
	for _, opt := range opts {
		opt(o)
	}

	// 设置默认的 gRPC 凭据提取器
	if o.credentialsExtractor == nil {
		o.credentialsExtractor = DefaultGRPCCredentialsExtractor
	}

	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()

		// 检查是否跳过
		if o.skipper != nil && o.skipper(ctx, nil) {
			return handler(srv, ss)
		}

		// 提取凭据
		creds, err := o.credentialsExtractor(ctx, nil)
		if err != nil {
			if o.logger != nil {
				o.logger.WithContext(ctx).Debug(
					"[Auth] gRPC流凭据提取失败",
					logger.String("method", info.FullMethod),
					logger.Err(err),
				)
			}
			return status.Error(codes.Unauthenticated, "credentials not found")
		}

		// 认证
		principal, err := authenticator.Authenticate(ctx, *creds)
		if err != nil {
			if o.logger != nil {
				o.logger.WithContext(ctx).Warn(
					"[Auth] gRPC流认证失败",
					logger.String("method", info.FullMethod),
					logger.Err(err),
				)
			}
			return status.Error(codes.Unauthenticated, "authentication failed")
		}

		// 将主体存入 context
		ctx = WithPrincipal(ctx, principal)

		// 授权
		if o.authorizer != nil {
			action := o.action
			resource := o.resource

			if action == "" {
				action = extractGRPCAction(info.FullMethod)
			}
			if resource == "" {
				resource = extractGRPCResource(info.FullMethod)
			}

			if err := o.authorizer.Authorize(ctx, principal, action, resource); err != nil {
				if o.logger != nil {
					o.logger.WithContext(ctx).Warn(
						"[Auth] gRPC流授权失败",
						logger.String("principal_id", principal.ID),
						logger.String("method", info.FullMethod),
						logger.Err(err),
					)
				}
				return status.Error(codes.PermissionDenied, "permission denied")
			}
		}

		if o.logger != nil {
			o.logger.WithContext(ctx).Debug(
				"[Auth] gRPC流认证成功",
				logger.String("principal_id", principal.ID),
				logger.String("method", info.FullMethod),
			)
		}

		// 包装 ServerStream 以使用新的 context
		wrapped := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrapped)
	}
}

// wrappedServerStream 包装 grpc.ServerStream 以替换 context.
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// DefaultGRPCCredentialsExtractor 默认的 gRPC 凭据提取器.
//
// 按以下顺序尝试提取:
//  1. authorization 元数据 (Bearer Token)
//  2. x-api-key 元数据
func DefaultGRPCCredentialsExtractor(ctx context.Context, _ any) (*Credentials, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, ErrCredentialsNotFound
	}

	// 1. 尝试 authorization (Bearer)
	if vals := md.Get(GRPCAuthorizationMetadata); len(vals) > 0 {
		auth := vals[0]
		if strings.HasPrefix(auth, BearerPrefix) {
			return &Credentials{
				Type:  CredentialTypeBearer,
				Token: strings.TrimPrefix(auth, BearerPrefix),
			}, nil
		}
		// 也支持小写 bearer
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			return &Credentials{
				Type:  CredentialTypeBearer,
				Token: auth[7:],
			}, nil
		}
	}

	// 2. 尝试 x-api-key
	if vals := md.Get(GRPCAPIKeyMetadata); len(vals) > 0 {
		return &Credentials{
			Type:  CredentialTypeAPIKey,
			Token: vals[0],
		}, nil
	}

	return nil, ErrCredentialsNotFound
}

// GRPCBearerExtractor 仅提取 Bearer Token.
func GRPCBearerExtractor(ctx context.Context, _ any) (*Credentials, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, ErrCredentialsNotFound
	}

	vals := md.Get(GRPCAuthorizationMetadata)
	if len(vals) == 0 {
		return nil, ErrCredentialsNotFound
	}

	auth := vals[0]
	if !strings.HasPrefix(auth, BearerPrefix) && !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return nil, ErrCredentialsNotFound
	}

	token := auth
	if strings.HasPrefix(auth, BearerPrefix) {
		token = strings.TrimPrefix(auth, BearerPrefix)
	} else {
		token = auth[7:]
	}

	return &Credentials{
		Type:  CredentialTypeBearer,
		Token: token,
	}, nil
}

// GRPCAPIKeyExtractor 仅提取 API Key.
func GRPCAPIKeyExtractor(metadataKey string) CredentialsExtractor {
	if metadataKey == "" {
		metadataKey = GRPCAPIKeyMetadata
	}
	return func(ctx context.Context, _ any) (*Credentials, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, ErrCredentialsNotFound
		}

		vals := md.Get(metadataKey)
		if len(vals) == 0 {
			return nil, ErrCredentialsNotFound
		}

		return &Credentials{
			Type:  CredentialTypeAPIKey,
			Token: vals[0],
		}, nil
	}
}

// GRPCSkipMethods 返回跳过指定 gRPC 方法的 Skipper.
func GRPCSkipMethods(methods ...string) Skipper {
	methodSet := make(map[string]bool)
	for _, m := range methods {
		methodSet[m] = true
	}
	return func(ctx context.Context, _ any) bool {
		method, ok := grpc.Method(ctx)
		if !ok {
			return false
		}
		return methodSet[method]
	}
}

// extractGRPCAction 从 gRPC 方法名提取操作.
// 格式: /package.service/Method -> Method (转小写)
func extractGRPCAction(fullMethod string) string {
	parts := strings.Split(fullMethod, "/")
	if len(parts) >= 3 {
		method := parts[2]
		// 常见前缀映射
		switch {
		case strings.HasPrefix(method, "Get"), strings.HasPrefix(method, "List"), strings.HasPrefix(method, "Query"):
			return "read"
		case strings.HasPrefix(method, "Create"), strings.HasPrefix(method, "Add"):
			return "create"
		case strings.HasPrefix(method, "Update"), strings.HasPrefix(method, "Set"):
			return "update"
		case strings.HasPrefix(method, "Delete"), strings.HasPrefix(method, "Remove"):
			return "delete"
		default:
			return strings.ToLower(method)
		}
	}
	return strings.ToLower(fullMethod)
}

// extractGRPCResource 从 gRPC 方法名提取资源.
// 格式: /package.service/Method -> package.service
func extractGRPCResource(fullMethod string) string {
	parts := strings.Split(fullMethod, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return fullMethod
}
