package server

import (
	"context"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/Tsukikage7/microservice-kit/auth"
	"github.com/Tsukikage7/microservice-kit/logger"
	"github.com/Tsukikage7/microservice-kit/recovery"
	"github.com/Tsukikage7/microservice-kit/tracing"
	"github.com/Tsukikage7/microservice-kit/transport"
	"github.com/Tsukikage7/microservice-kit/transport/health"
	"github.com/Tsukikage7/microservice-kit/transport/response"
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

	// Response
	enableResponse bool // 是否启用统一响应格式

	// Recovery
	enableRecovery bool // 是否启用 panic 恢复

	// Auth
	authenticator       auth.Authenticator
	authOptions         []auth.Option
	publicMethods       []string          // 公开方法（无需认证）
	enableAutoDiscovery bool              // 启用 proto option 自动发现
	discoveredMethods   map[string]bool   // 自动发现的公开方法（延迟填充）

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

// WithConfig 从配置结构体设置服务器选项.
// 仅设置非零值字段，零值字段将保持默认值.
func WithConfig(cfg transport.GatewayConfig) Option {
	return func(o *options) {
		if cfg.Name != "" {
			o.name = cfg.Name
		}
		if cfg.GRPCAddr != "" {
			o.grpcAddr = cfg.GRPCAddr
		}
		if cfg.HTTPAddr != "" {
			o.httpAddr = cfg.HTTPAddr
		}
		if cfg.KeepaliveTime > 0 {
			o.keepaliveTime = cfg.KeepaliveTime
		}
		if len(cfg.PublicMethods) > 0 {
			o.publicMethods = cfg.PublicMethods
		}
	}
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

// WithResponse 启用统一响应格式（gRPC + HTTP 双端）.
//
// 启用后:
//   - HTTP 错误响应将使用统一的 JSON 格式: {"code": xxx, "message": "xxx"}
//   - gRPC 错误将自动映射到正确的状态码
//   - 内部错误（5xxxx）的详细信息将被隐藏
func WithResponse() Option {
	return func(o *options) {
		o.enableResponse = true
		// 将 response 拦截器添加到拦截器链末尾（在业务逻辑之后处理错误）
		o.unaryInterceptors = append(o.unaryInterceptors, response.UnaryServerInterceptor())
	}
}

// WithRecovery 启用 panic 恢复（gRPC + HTTP 双端）.
//
// 启用后，handler 中的 panic 会被捕获并记录:
//   - gRPC: 返回 codes.Internal 错误
//   - HTTP: 返回 500 状态码
func WithRecovery() Option {
	return func(o *options) {
		o.enableRecovery = true
	}
}

// WithAuth 启用认证（gRPC + HTTP 双端）.
//
// 由于 gRPC-Gateway 会将 HTTP 请求转换为 gRPC 调用，
// 只需在 gRPC 层添加认证拦截器即可同时保护两种协议。
//
// 示例:
//
//	jwtSrv := jwt.NewJWT(jwt.WithSecretKey("secret"))
//	authenticator := jwt.NewAuthenticator(jwtSrv)
//
//	server := gateway.New(
//	    gateway.WithAuth(authenticator),
//	    gateway.WithPublicMethods(
//	        "/api.user.v1.AuthService/Login",
//	        "/api.user.v1.AuthService/Register",
//	    ),
//	)
func WithAuth(authenticator auth.Authenticator, opts ...auth.Option) Option {
	return func(o *options) {
		o.authenticator = authenticator
		o.authOptions = opts
	}
}

// WithPublicMethods 设置公开方法（无需认证）.
//
// 方法名格式为 gRPC 完整方法名，如:
//   - "/api.user.v1.AuthService/Login"
//   - "/api.user.v1.AuthService/Register"
//   - "/api.product.v1.ProductService/List"
//
// 也支持服务级别的通配:
//   - "/api.user.v1.AuthService/*" (该服务下所有方法)
func WithPublicMethods(methods ...string) Option {
	return func(o *options) {
		o.publicMethods = append(o.publicMethods, methods...)
	}
}

// WithAutoDiscovery 启用 proto option 自动发现.
//
// 启用后，服务器会在启动时自动扫描注册的 gRPC 服务，
// 从 proto 定义中发现标记为 public 的方法，无需手动配置 WithPublicMethods.
//
// 在 proto 中标记公开方法:
//
//	import "github.com/Tsukikage7/microservice-kit/auth/proto/auth.proto";
//
//	service AuthService {
//	  rpc Login(LoginRequest) returns (LoginResponse) {
//	    option (microservice.kit.auth.method) = {
//	      public: true
//	    };
//	  }
//	}
//
// 也可以标记整个服务为公开:
//
//	service PublicService {
//	  option (microservice.kit.auth.service) = {
//	    public: true
//	  };
//	}
//
// 注意: 自动发现会与手动配置的 WithPublicMethods 合并.
func WithAutoDiscovery() Option {
	return func(o *options) {
		o.enableAutoDiscovery = true
	}
}

// applyRecoveryInterceptors 应用 recovery 拦截器到拦截器链最前面.
func applyRecoveryInterceptors(o *options) {
	if !o.enableRecovery || o.logger == nil {
		return
	}
	o.unaryInterceptors = append(
		[]grpc.UnaryServerInterceptor{recovery.UnaryServerInterceptor(recovery.WithLogger(o.logger))},
		o.unaryInterceptors...,
	)
	o.streamInterceptors = append(
		[]grpc.StreamServerInterceptor{recovery.StreamServerInterceptor(recovery.WithLogger(o.logger))},
		o.streamInterceptors...,
	)
}

// applyAuthInterceptors 应用 auth 拦截器.
func applyAuthInterceptors(o *options) {
	if o.authenticator == nil {
		return
	}

	// 如果启用自动发现，初始化 map
	if o.enableAutoDiscovery {
		o.discoveredMethods = make(map[string]bool)
	}

	// 构建 skipper（支持手动配置 + 自动发现）
	skipper := buildCombinedSkipper(o)

	// 合并选项
	authOpts := append([]auth.Option{}, o.authOptions...)
	if skipper != nil {
		authOpts = append(authOpts, auth.WithSkipper(skipper))
	}
	if o.logger != nil {
		authOpts = append(authOpts, auth.WithLogger(o.logger))
	}

	// 添加到拦截器链（在 recovery 之后）
	o.unaryInterceptors = append(
		o.unaryInterceptors,
		auth.UnaryServerInterceptor(o.authenticator, authOpts...),
	)
	o.streamInterceptors = append(
		o.streamInterceptors,
		auth.StreamServerInterceptor(o.authenticator, authOpts...),
	)
}

// buildCombinedSkipper 构建组合跳过器（手动配置 + 自动发现）.
func buildCombinedSkipper(o *options) auth.Skipper {
	// 解析手动配置的公开方法
	exact := make(map[string]bool)
	prefixes := make([]string, 0)

	for _, m := range o.publicMethods {
		if strings.HasSuffix(m, "/*") {
			prefixes = append(prefixes, strings.TrimSuffix(m, "*"))
		} else {
			exact[m] = true
		}
	}

	// 如果没有任何配置，返回 nil
	if len(exact) == 0 && len(prefixes) == 0 && !o.enableAutoDiscovery {
		return nil
	}

	return func(ctx context.Context, _ any) bool {
		method, ok := grpc.Method(ctx)
		if !ok {
			return false
		}

		// 1. 检查手动配置的精确匹配
		if exact[method] {
			return true
		}

		// 2. 检查手动配置的前缀匹配
		for _, prefix := range prefixes {
			if strings.HasPrefix(method, prefix) {
				return true
			}
		}

		// 3. 检查自动发现的方法（延迟填充）
		if o.discoveredMethods != nil && o.discoveredMethods[method] {
			return true
		}

		return false
	}
}

// buildMethodSkipper 构建方法跳过器.
func buildMethodSkipper(publicMethods []string) auth.Skipper {
	// 分离精确匹配和通配符
	exact := make(map[string]bool)
	prefixes := make([]string, 0)

	for _, m := range publicMethods {
		if strings.HasSuffix(m, "/*") {
			// 服务级别通配: "/api.user.v1.AuthService/*" -> "/api.user.v1.AuthService/"
			prefixes = append(prefixes, strings.TrimSuffix(m, "*"))
		} else {
			exact[m] = true
		}
	}

	return func(ctx context.Context, _ any) bool {
		method, ok := grpc.Method(ctx)
		if !ok {
			return false
		}
		// 精确匹配
		if exact[method] {
			return true
		}
		// 前缀匹配
		for _, prefix := range prefixes {
			if strings.HasPrefix(method, prefix) {
				return true
			}
		}
		return false
	}
}
