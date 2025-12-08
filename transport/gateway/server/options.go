package server

import (
	"time"

	"github.com/Tsukikage7/microservice-kit/logger"
	"github.com/Tsukikage7/microservice-kit/tracing"
	"github.com/Tsukikage7/microservice-kit/transport/health"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

// Option 配置选项.
type Option func(*options)

type options struct {
	name     string
	services []Registrar

	// gRPC
	grpcAddr           string
	enableReflection   bool
	keepaliveTime      time.Duration
	keepaliveTimeout   time.Duration
	minPingInterval    time.Duration
	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor
	grpcServerOpts     []grpc.ServerOption

	// HTTP
	httpAddr         string
	httpReadTimeout  time.Duration
	httpWriteTimeout time.Duration
	httpIdleTimeout  time.Duration

	// Gateway
	dialOptions    []grpc.DialOption
	serveMuxOpts   []runtime.ServeMuxOption
	marshalOptions protojson.MarshalOptions

	// Health (内置)
	healthTimeout time.Duration
	healthOptions []health.Option

	// Trace
	tracerName string // 链路追踪服务名，为空则不启用

	logger logger.Logger
}

func defaultOptions() *options {
	return &options{
		name:             "Gateway",
		grpcAddr:         ":9090",
		httpAddr:         ":8080",
		enableReflection: true,
		keepaliveTime:    60 * time.Second,
		keepaliveTimeout: 20 * time.Second,
		minPingInterval:  20 * time.Second,
		httpReadTimeout:  30 * time.Second,
		httpWriteTimeout: 30 * time.Second,
		httpIdleTimeout:  120 * time.Second,
		healthTimeout:    5 * time.Second,
	}
}

// WithName 设置服务名称.
func WithName(name string) Option {
	return func(o *options) { o.name = name }
}

// WithGRPCAddr 设置 gRPC 地址.
func WithGRPCAddr(addr string) Option {
	return func(o *options) { o.grpcAddr = addr }
}

// WithHTTPAddr 设置 HTTP 地址.
func WithHTTPAddr(addr string) Option {
	return func(o *options) { o.httpAddr = addr }
}

// WithLogger 设置日志记录器.
func WithLogger(log logger.Logger) Option {
	return func(o *options) { o.logger = log }
}

// WithReflection 启用/禁用 gRPC 反射.
func WithReflection(enabled bool) Option {
	return func(o *options) { o.enableReflection = enabled }
}

// WithKeepalive 设置 gRPC keepalive 参数.
func WithKeepalive(t, timeout time.Duration) Option {
	return func(o *options) {
		o.keepaliveTime = t
		o.keepaliveTimeout = timeout
	}
}

// WithUnaryInterceptor 添加 gRPC 一元拦截器.
func WithUnaryInterceptor(interceptors ...grpc.UnaryServerInterceptor) Option {
	return func(o *options) {
		o.unaryInterceptors = append(o.unaryInterceptors, interceptors...)
	}
}

// WithStreamInterceptor 添加 gRPC 流拦截器.
func WithStreamInterceptor(interceptors ...grpc.StreamServerInterceptor) Option {
	return func(o *options) {
		o.streamInterceptors = append(o.streamInterceptors, interceptors...)
	}
}

// WithGRPCServerOption 添加 gRPC 服务器选项.
func WithGRPCServerOption(opts ...grpc.ServerOption) Option {
	return func(o *options) {
		o.grpcServerOpts = append(o.grpcServerOpts, opts...)
	}
}

// WithHTTPTimeout 设置 HTTP 超时.
func WithHTTPTimeout(read, write, idle time.Duration) Option {
	return func(o *options) {
		o.httpReadTimeout = read
		o.httpWriteTimeout = write
		o.httpIdleTimeout = idle
	}
}

// WithDialOptions 添加 gRPC 拨号选项.
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(o *options) {
		o.dialOptions = append(o.dialOptions, opts...)
	}
}

// WithServeMuxOptions 添加 ServeMux 选项.
func WithServeMuxOptions(opts ...runtime.ServeMuxOption) Option {
	return func(o *options) {
		o.serveMuxOpts = append(o.serveMuxOpts, opts...)
	}
}

// WithMarshalOptions 设置 JSON 序列化选项.
func WithMarshalOptions(opts protojson.MarshalOptions) Option {
	return func(o *options) { o.marshalOptions = opts }
}

// WithHealthTimeout 设置健康检查超时时间.
func WithHealthTimeout(d time.Duration) Option {
	return func(o *options) { o.healthTimeout = d }
}

// WithHealthOptions 添加健康检查选项.
//
// 例如添加就绪检查器:
//
//	WithHealthOptions(
//	    health.WithReadinessChecker(health.NewDBChecker("db", db)),
//	)
func WithHealthOptions(opts ...health.Option) Option {
	return func(o *options) {
		o.healthOptions = append(o.healthOptions, opts...)
	}
}

// WithReadinessChecker 添加就绪检查器（便捷方法）.
func WithReadinessChecker(checkers ...health.Checker) Option {
	return func(o *options) {
		o.healthOptions = append(o.healthOptions, health.WithReadinessChecker(checkers...))
	}
}

// WithLivenessChecker 添加存活检查器（便捷方法）.
func WithLivenessChecker(checkers ...health.Checker) Option {
	return func(o *options) {
		o.healthOptions = append(o.healthOptions, health.WithLivenessChecker(checkers...))
	}
}

// WithTrace 启用链路追踪（gRPC + HTTP 双端）.
//
// 注意: 需要先调用 tracing.NewTracer() 初始化全局 TracerProvider.
func WithTrace(serviceName string) Option {
	return func(o *options) {
		o.tracerName = serviceName
		// 将 trace 拦截器添加到拦截器链最前面
		o.unaryInterceptors = append(
			[]grpc.UnaryServerInterceptor{tracing.UnaryServerInterceptor(serviceName)},
			o.unaryInterceptors...,
		)
		o.streamInterceptors = append(
			[]grpc.StreamServerInterceptor{tracing.StreamServerInterceptor(serviceName)},
			o.streamInterceptors...,
		)
	}
}
