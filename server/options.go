package server

import (
	"os"
	"time"

	"github.com/Tsukikage7/microservice-kit/logger"
	"google.golang.org/grpc"
)

// AppOption App 配置选项.
type AppOption func(*appOptions)

// appOptions App 内部配置.
type appOptions struct {
	name            string
	version         string
	logger          logger.Logger
	hooks           *Hooks
	gracefulTimeout time.Duration
	signals         []os.Signal
}

// defaultAppOptions 返回默认配置.
func defaultAppOptions() *appOptions {
	return &appOptions{
		name:            "app",
		version:         "1.0.0",
		gracefulTimeout: 30 * time.Second,
	}
}

// WithName 设置应用名称.
func WithName(name string) AppOption {
	return func(o *appOptions) {
		o.name = name
	}
}

// WithVersion 设置应用版本.
func WithVersion(version string) AppOption {
	return func(o *appOptions) {
		o.version = version
	}
}

// WithLogger 设置日志记录器.
func WithLogger(log logger.Logger) AppOption {
	return func(o *appOptions) {
		o.logger = log
	}
}

// WithHooks 设置生命周期钩子.
func WithHooks(hooks *Hooks) AppOption {
	return func(o *appOptions) {
		o.hooks = hooks
	}
}

// WithGracefulTimeout 设置优雅关闭超时时间.
//
// 默认: 30 秒.
func WithGracefulTimeout(d time.Duration) AppOption {
	return func(o *appOptions) {
		o.gracefulTimeout = d
	}
}

// WithSignals 设置监听的系统信号.
//
// 默认: SIGINT, SIGTERM.
func WithSignals(signals ...os.Signal) AppOption {
	return func(o *appOptions) {
		o.signals = signals
	}
}

// HTTPOption HTTP 服务器配置选项.
type HTTPOption func(*httpOptions)

// httpOptions HTTP 服务器内部配置.
type httpOptions struct {
	name         string
	addr         string
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
	logger       logger.Logger
}

// defaultHTTPOptions 返回默认 HTTP 配置.
func defaultHTTPOptions() *httpOptions {
	return &httpOptions{
		name:         "HTTP",
		addr:         ":8080",
		readTimeout:  30 * time.Second,
		writeTimeout: 30 * time.Second,
		idleTimeout:  120 * time.Second,
	}
}

// WithHTTPName 设置 HTTP 服务器名称.
func WithHTTPName(name string) HTTPOption {
	return func(o *httpOptions) {
		o.name = name
	}
}

// WithHTTPAddr 设置 HTTP 监听地址.
func WithHTTPAddr(addr string) HTTPOption {
	return func(o *httpOptions) {
		o.addr = addr
	}
}

// WithHTTPReadTimeout 设置读取超时.
func WithHTTPReadTimeout(d time.Duration) HTTPOption {
	return func(o *httpOptions) {
		o.readTimeout = d
	}
}

// WithHTTPWriteTimeout 设置写入超时.
func WithHTTPWriteTimeout(d time.Duration) HTTPOption {
	return func(o *httpOptions) {
		o.writeTimeout = d
	}
}

// WithHTTPIdleTimeout 设置空闲超时.
func WithHTTPIdleTimeout(d time.Duration) HTTPOption {
	return func(o *httpOptions) {
		o.idleTimeout = d
	}
}

// WithHTTPLogger 设置日志记录器.
func WithHTTPLogger(log logger.Logger) HTTPOption {
	return func(o *httpOptions) {
		o.logger = log
	}
}

// GRPCOption gRPC 服务器配置选项.
type GRPCOption func(*grpcOptions)

// grpcOptions gRPC 服务器内部配置.
type grpcOptions struct {
	name               string
	addr               string
	enableReflection   bool
	keepaliveTime      time.Duration
	keepaliveTimeout   time.Duration
	minPingInterval    time.Duration
	logger             logger.Logger
	services           []GRPCRegistrar
	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor
	serverOptions      []grpc.ServerOption
}

// defaultGRPCOptions 返回默认 gRPC 配置.
func defaultGRPCOptions() *grpcOptions {
	return &grpcOptions{
		name:             "gRPC",
		addr:             ":9090",
		enableReflection: true,
		keepaliveTime:    60 * time.Second,
		keepaliveTimeout: 20 * time.Second,
		minPingInterval:  20 * time.Second,
	}
}

// WithGRPCName 设置 gRPC 服务器名称.
func WithGRPCName(name string) GRPCOption {
	return func(o *grpcOptions) {
		o.name = name
	}
}

// WithGRPCAddr 设置 gRPC 监听地址.
func WithGRPCAddr(addr string) GRPCOption {
	return func(o *grpcOptions) {
		o.addr = addr
	}
}

// WithGRPCReflection 设置是否启用反射.
func WithGRPCReflection(enabled bool) GRPCOption {
	return func(o *grpcOptions) {
		o.enableReflection = enabled
	}
}

// WithGRPCKeepalive 设置 Keepalive 参数.
func WithGRPCKeepalive(time, timeout time.Duration) GRPCOption {
	return func(o *grpcOptions) {
		o.keepaliveTime = time
		o.keepaliveTimeout = timeout
	}
}

// WithGRPCLogger 设置日志记录器.
func WithGRPCLogger(log logger.Logger) GRPCOption {
	return func(o *grpcOptions) {
		o.logger = log
	}
}

// WithGRPCUnaryInterceptor 添加一元拦截器.
func WithGRPCUnaryInterceptor(interceptors ...grpc.UnaryServerInterceptor) GRPCOption {
	return func(o *grpcOptions) {
		o.unaryInterceptors = append(o.unaryInterceptors, interceptors...)
	}
}

// WithGRPCStreamInterceptor 添加流拦截器.
func WithGRPCStreamInterceptor(interceptors ...grpc.StreamServerInterceptor) GRPCOption {
	return func(o *grpcOptions) {
		o.streamInterceptors = append(o.streamInterceptors, interceptors...)
	}
}

// WithGRPCServerOption 添加自定义 gRPC 服务器选项.
func WithGRPCServerOption(opts ...grpc.ServerOption) GRPCOption {
	return func(o *grpcOptions) {
		o.serverOptions = append(o.serverOptions, opts...)
	}
}
