package client

import (
	"github.com/Tsukikage7/microservice-kit/discovery"
	"github.com/Tsukikage7/microservice-kit/logger"
	"google.golang.org/grpc"
)

// Option 配置选项函数.
type Option func(*options)

// options 客户端配置.
type options struct {
	name         string
	serviceName  string
	discovery    discovery.Discovery
	logger       logger.Logger
	interceptors []grpc.UnaryClientInterceptor
	dialOptions  []grpc.DialOption
}

// defaultOptions 返回默认配置.
func defaultOptions() *options {
	return &options{
		name: "gRPC-Client",
	}
}

// WithName 设置客户端名称（用于日志）.
func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// WithServiceName 设置目标服务名称（必需）.
func WithServiceName(name string) Option {
	return func(o *options) {
		o.serviceName = name
	}
}

// WithDiscovery 设置服务发现实例（必需）.
func WithDiscovery(d discovery.Discovery) Option {
	return func(o *options) {
		o.discovery = d
	}
}

// WithLogger 设置日志实例（必需）.
func WithLogger(l logger.Logger) Option {
	return func(o *options) {
		o.logger = l
	}
}

// WithInterceptors 添加自定义拦截器.
func WithInterceptors(interceptors ...grpc.UnaryClientInterceptor) Option {
	return func(o *options) {
		o.interceptors = append(o.interceptors, interceptors...)
	}
}

// WithDialOptions 添加额外的 dial 选项.
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(o *options) {
		o.dialOptions = append(o.dialOptions, opts...)
	}
}
