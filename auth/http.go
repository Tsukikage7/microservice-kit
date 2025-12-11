package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// HTTP 相关常量.
const (
	// AuthorizationHeader Authorization 请求头.
	AuthorizationHeader = "Authorization"

	// BearerPrefix Bearer 前缀.
	BearerPrefix = "Bearer "

	// APIKeyHeader API Key 请求头.
	APIKeyHeader = "X-API-Key"
)

// HTTPMiddleware 返回 HTTP 认证授权中间件.
//
// 默认从 Authorization 请求头提取 Bearer Token.
//
// 示例:
//
//	authenticator := authjwt.NewAuthenticator(jwtService)
//	handler = auth.HTTPMiddleware(authenticator)(handler)
func HTTPMiddleware(authenticator Authenticator, opts ...Option) func(http.Handler) http.Handler {
	if authenticator == nil {
		panic("auth: 认证器不能为空")
	}

	o := defaultOptions(authenticator)
	for _, opt := range opts {
		opt(o)
	}

	// 设置默认的 HTTP 凭据提取器
	if o.credentialsExtractor == nil {
		o.credentialsExtractor = DefaultHTTPCredentialsExtractor
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 检查是否跳过
			if o.skipper != nil && o.skipper(ctx, r) {
				next.ServeHTTP(w, r)
				return
			}

			// 提取凭据
			creds, err := o.credentialsExtractor(ctx, r)
			if err != nil {
				if o.logger != nil {
					o.logger.WithContext(ctx).Debug(
						"[Auth] HTTP凭据提取失败",
						logger.String("method", r.Method),
						logger.String("path", r.URL.Path),
						logger.Err(err),
					)
				}
				writeHTTPError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			// 认证
			principal, err := authenticator.Authenticate(ctx, *creds)
			if err != nil {
				if o.logger != nil {
					o.logger.WithContext(ctx).Warn(
						"[Auth] HTTP认证失败",
						logger.String("method", r.Method),
						logger.String("path", r.URL.Path),
						logger.Err(err),
					)
				}
				writeHTTPError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			// 将主体存入 context
			ctx = WithPrincipal(ctx, principal)

			// 授权
			if o.authorizer != nil {
				action := o.action
				resource := o.resource

				// 如果没有指定，使用 HTTP 方法和路径
				if action == "" {
					action = httpMethodToAction(r.Method)
				}
				if resource == "" {
					resource = r.URL.Path
				}

				if err := o.authorizer.Authorize(ctx, principal, action, resource); err != nil {
					if o.logger != nil {
						o.logger.WithContext(ctx).Warn(
							"[Auth] HTTP授权失败",
							logger.String("principal_id", principal.ID),
							logger.String("method", r.Method),
							logger.String("path", r.URL.Path),
							logger.Err(err),
						)
					}
					writeHTTPError(w, http.StatusForbidden, "Forbidden")
					return
				}
			}

			if o.logger != nil {
				o.logger.WithContext(ctx).Debug(
					"[Auth] HTTP认证成功",
					logger.String("principal_id", principal.ID),
					logger.String("method", r.Method),
					logger.String("path", r.URL.Path),
				)
			}

			// 继续处理
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// DefaultHTTPCredentialsExtractor 默认的 HTTP 凭据提取器.
//
// 按以下顺序尝试提取:
//  1. Authorization: Bearer <token>
//  2. X-API-Key: <key>
//  3. Query 参数: access_token
func DefaultHTTPCredentialsExtractor(_ context.Context, request any) (*Credentials, error) {
	r, ok := request.(*http.Request)
	if !ok {
		return nil, ErrCredentialsNotFound
	}

	// 1. 尝试 Authorization Header (Bearer)
	if auth := r.Header.Get(AuthorizationHeader); auth != "" {
		if strings.HasPrefix(auth, BearerPrefix) {
			token := strings.TrimPrefix(auth, BearerPrefix)
			return &Credentials{
				Type:  CredentialTypeBearer,
				Token: token,
			}, nil
		}
		// Basic Auth
		if strings.HasPrefix(auth, "Basic ") {
			return &Credentials{
				Type:  CredentialTypeBasic,
				Token: strings.TrimPrefix(auth, "Basic "),
			}, nil
		}
	}

	// 2. 尝试 X-API-Key Header
	if apiKey := r.Header.Get(APIKeyHeader); apiKey != "" {
		return &Credentials{
			Type:  CredentialTypeAPIKey,
			Token: apiKey,
		}, nil
	}

	// 3. 尝试 Query 参数
	if token := r.URL.Query().Get("access_token"); token != "" {
		return &Credentials{
			Type:  CredentialTypeBearer,
			Token: token,
		}, nil
	}

	return nil, ErrCredentialsNotFound
}

// BearerExtractor 仅提取 Bearer Token.
func BearerExtractor(_ context.Context, request any) (*Credentials, error) {
	r, ok := request.(*http.Request)
	if !ok {
		return nil, ErrCredentialsNotFound
	}

	auth := r.Header.Get(AuthorizationHeader)
	if auth == "" || !strings.HasPrefix(auth, BearerPrefix) {
		return nil, ErrCredentialsNotFound
	}

	return &Credentials{
		Type:  CredentialTypeBearer,
		Token: strings.TrimPrefix(auth, BearerPrefix),
	}, nil
}

// APIKeyExtractor 仅提取 API Key.
func APIKeyExtractor(header string) CredentialsExtractor {
	if header == "" {
		header = APIKeyHeader
	}
	return func(_ context.Context, request any) (*Credentials, error) {
		r, ok := request.(*http.Request)
		if !ok {
			return nil, ErrCredentialsNotFound
		}

		apiKey := r.Header.Get(header)
		if apiKey == "" {
			return nil, ErrCredentialsNotFound
		}

		return &Credentials{
			Type:  CredentialTypeAPIKey,
			Token: apiKey,
		}, nil
	}
}

// writeHTTPError 写入 HTTP 错误响应.
func writeHTTPError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}

// httpMethodToAction 将 HTTP 方法映射为操作.
func httpMethodToAction(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return "read"
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return strings.ToLower(method)
	}
}

// HTTPSkipPaths 返回跳过指定路径的 Skipper.
func HTTPSkipPaths(paths ...string) Skipper {
	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[p] = true
	}
	return func(_ context.Context, request any) bool {
		if r, ok := request.(*http.Request); ok {
			return pathSet[r.URL.Path]
		}
		return false
	}
}

// HTTPSkipMethods 返回跳过指定 HTTP 方法的 Skipper.
func HTTPSkipMethods(methods ...string) Skipper {
	methodSet := make(map[string]bool)
	for _, m := range methods {
		methodSet[strings.ToUpper(m)] = true
	}
	return func(_ context.Context, request any) bool {
		if r, ok := request.(*http.Request); ok {
			return methodSet[r.Method]
		}
		return false
	}
}
