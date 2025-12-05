package server

import (
	"time"

	"github.com/Tsukikage7/microservice-kit/logger"
	"google.golang.org/grpc"
)

// Option 配置选项函数.
type Option func(*options)

// options 服务器配置.
type options struct {
	name               string
	addr               string
	enableReflection   bool
	keepaliveTime      time.Duration
	keepaliveTimeout   time.Duration
	minPingInterval    time.Duration
	logger             logger.Logger
	services           []Registrar
	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor
	serverOptions      []grpc.ServerOption
}

// defaultOptions 返回默认配置.
func defaultOptions() *options {
	return &options{
		name:             "gRPC",
		addr:             ":9090",
		enableReflection: true,
		keepaliveTime:    60 * time.Second,
		keepaliveTimeout: 20 * time.Second,
		minPingInterval:  20 * time.Second,
	}
}

// WithName 设置服务器名称.
func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// WithAddr 设置监听地址.
func WithAddr(addr string) Option {
	return func(o *options) {
		o.addr = addr
	}
}

// WithReflection 设置是否启用反射.
func WithReflection(enabled bool) Option {
	return func(o *options) {
		o.enableReflection = enabled
	}
}

// WithKeepalive 设置 Keepalive 参数.
func WithKeepalive(time, timeout time.Duration) Option {
	return func(o *options) {
		o.keepaliveTime = time
		o.keepaliveTimeout = timeout
	}
}

// WithLogger 设置日志记录器（必需）.
func WithLogger(log logger.Logger) Option {
	return func(o *options) {
		o.logger = log
	}
}

// WithUnaryInterceptor 添加一元拦截器.
func WithUnaryInterceptor(interceptors ...grpc.UnaryServerInterceptor) Option {
	return func(o *options) {
		o.unaryInterceptors = append(o.unaryInterceptors, interceptors...)
	}
}

// WithStreamInterceptor 添加流拦截器.
func WithStreamInterceptor(interceptors ...grpc.StreamServerInterceptor) Option {
	return func(o *options) {
		o.streamInterceptors = append(o.streamInterceptors, interceptors...)
	}
}

// WithServerOption 添加自定义 gRPC 服务器选项.
func WithServerOption(opts ...grpc.ServerOption) Option {
	return func(o *options) {
		o.serverOptions = append(o.serverOptions, opts...)
	}
}
