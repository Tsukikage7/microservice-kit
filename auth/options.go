package auth

import (
	"github.com/Tsukikage7/microservice-kit/logger"
)

// options 中间件配置.
type options struct {
	authenticator        Authenticator
	authorizer           Authorizer
	credentialsExtractor CredentialsExtractor
	skipper              Skipper
	errorHandler         ErrorHandler
	logger               logger.Logger

	// 授权参数
	action   string
	resource string
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
//
// 当函数返回 true 时，跳过认证/授权检查.
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

// WithActionResource 设置授权的操作和资源.
func WithActionResource(action, resource string) Option {
	return func(o *options) {
		o.action = action
		o.resource = resource
	}
}

// WithAction 设置授权的操作.
func WithAction(action string) Option {
	return func(o *options) {
		o.action = action
	}
}

// WithResource 设置授权的资源.
func WithResource(resource string) Option {
	return func(o *options) {
		o.resource = resource
	}
}
