package auth

import (
	"context"

	"github.com/Tsukikage7/microservice-kit/logger"
	"github.com/Tsukikage7/microservice-kit/transport"
)

// Middleware 返回 Endpoint 认证授权中间件.
//
// 示例:
//
//	authenticator := authjwt.NewAuthenticator(jwtService)
//	endpoint = auth.Middleware(authenticator)(endpoint)
//
//	// 带授权
//	endpoint = auth.Middleware(authenticator,
//	    auth.WithAuthorizer(rbacAuth),
//	    auth.WithActionResource("read", "orders"),
//	)(endpoint)
func Middleware(authenticator Authenticator, opts ...Option) transport.Middleware {
	if authenticator == nil {
		panic("auth: 认证器不能为空")
	}

	o := defaultOptions(authenticator)
	for _, opt := range opts {
		opt(o)
	}

	return func(next transport.Endpoint) transport.Endpoint {
		return func(ctx context.Context, request any) (any, error) {
			// 检查是否跳过
			if o.skipper != nil && o.skipper(ctx, request) {
				return next(ctx, request)
			}

			// 提取凭据
			creds, err := extractCredentials(ctx, request, o)
			if err != nil {
				if o.logger != nil {
					o.logger.WithContext(ctx).Debug(
						"[Auth] 凭据提取失败",
						logger.Err(err),
					)
				}
				return nil, handleError(ctx, ErrCredentialsNotFound, o)
			}

			// 认证
			principal, err := authenticator.Authenticate(ctx, *creds)
			if err != nil {
				if o.logger != nil {
					o.logger.WithContext(ctx).Warn(
						"[Auth] 认证失败",
						logger.Err(err),
					)
				}
				return nil, handleError(ctx, err, o)
			}

			// 将主体存入 context
			ctx = WithPrincipal(ctx, principal)

			// 授权
			if o.authorizer != nil {
				if err := o.authorizer.Authorize(ctx, principal, o.action, o.resource); err != nil {
					if o.logger != nil {
						o.logger.WithContext(ctx).Warn(
							"[Auth] 授权失败",
							logger.String("principal_id", principal.ID),
							logger.String("action", o.action),
							logger.String("resource", o.resource),
							logger.Err(err),
						)
					}
					return nil, handleError(ctx, err, o)
				}
			}

			if o.logger != nil {
				o.logger.WithContext(ctx).Debug(
					"[Auth] 认证成功",
					logger.String("principal_id", principal.ID),
					logger.String("principal_type", principal.Type),
				)
			}

			return next(ctx, request)
		}
	}
}

// extractCredentials 提取凭据.
func extractCredentials(ctx context.Context, request any, o *options) (*Credentials, error) {
	// 使用自定义提取器
	if o.credentialsExtractor != nil {
		return o.credentialsExtractor(ctx, request)
	}

	// 尝试从 context 获取
	if creds, ok := CredentialsFromContext(ctx); ok {
		return creds, nil
	}

	// 尝试从请求中提取（如果请求实现了凭据接口）
	if credsProvider, ok := request.(interface{ Credentials() *Credentials }); ok {
		return credsProvider.Credentials(), nil
	}

	return nil, ErrCredentialsNotFound
}

// handleError 处理错误.
func handleError(ctx context.Context, err error, o *options) error {
	if o.errorHandler != nil {
		return o.errorHandler(ctx, err)
	}
	return err
}

// RequireAuth 便捷函数，创建仅认证的中间件.
func RequireAuth(authenticator Authenticator, opts ...Option) transport.Middleware {
	return Middleware(authenticator, opts...)
}

// RequireRoles 便捷函数，创建需要指定角色的中间件.
func RequireRoles(authenticator Authenticator, roles []string, opts ...Option) transport.Middleware {
	opts = append(opts, WithAuthorizer(NewRoleAuthorizer(roles)))
	return Middleware(authenticator, opts...)
}

// RequirePermissions 便捷函数，创建需要指定权限的中间件.
func RequirePermissions(authenticator Authenticator, permissions []string, opts ...Option) transport.Middleware {
	opts = append(opts, WithAuthorizer(NewPermissionAuthorizer(permissions)))
	return Middleware(authenticator, opts...)
}
