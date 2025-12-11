package auth

import "github.com/Tsukikage7/microservice-kit/logger"

// HTTP 相关常量.
const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
	APIKeyHeader        = "X-API-Key"
)

// gRPC 相关常量.
const (
	GRPCAuthorizationMetadata = "authorization"
	GRPCAPIKeyMetadata        = "x-api-key"
)

// options 中间件配置.
type options struct {
	authenticator        Authenticator
	authorizer           Authorizer
	credentialsExtractor CredentialsExtractor
	skipper              Skipper
	errorHandler         ErrorHandler
	logger               logger.Logger
}

// Option 中间件配置选项.
type Option func(*options)

// defaultOptions 返回默认配置.
func defaultOptions(authenticator Authenticator) *options {
	return &options{
		authenticator: authenticator,
	}
}

// WithAuthorizer 设置授权器.
func WithAuthorizer(authorizer Authorizer) Option {
	return func(o *options) {
		o.authorizer = authorizer
	}
}

// WithCredentialsExtractor 设置凭据提取器.
func WithCredentialsExtractor(extractor CredentialsExtractor) Option {
	return func(o *options) {
		o.credentialsExtractor = extractor
	}
}

// WithSkipper 设置跳过函数.
func WithSkipper(skipper Skipper) Option {
	return func(o *options) {
		o.skipper = skipper
	}
}

// WithErrorHandler 设置错误处理函数.
func WithErrorHandler(handler ErrorHandler) Option {
	return func(o *options) {
		o.errorHandler = handler
	}
}

// WithLogger 设置日志记录器.
func WithLogger(log logger.Logger) Option {
	return func(o *options) {
		o.logger = log
	}
}
