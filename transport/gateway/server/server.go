// Package server 提供 gRPC + HTTP (gRPC-Gateway) 双协议服务器.
package server

import (
	"context"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/Tsukikage7/microservice-kit/logger"
	"github.com/Tsukikage7/microservice-kit/tracing"
	"github.com/Tsukikage7/microservice-kit/transport"
	"github.com/Tsukikage7/microservice-kit/transport/health"
)

// Registrar 服务注册器接口.
type Registrar interface {
	// RegisterGRPC 注册 gRPC 服务.
	RegisterGRPC(server grpc.ServiceRegistrar)
	// RegisterGateway 注册 gRPC-Gateway 处理器.
	RegisterGateway(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}

// Server gRPC + HTTP 双协议服务器.
type Server struct {
	opts *options

	grpcServer   *grpc.Server
	grpcListener net.Listener

	httpServer  *http.Server
	httpHandler http.Handler
	mux         *runtime.ServeMux
	conn        *grpc.ClientConn

	// 内置健康检查
	health       *health.Health
	healthServer *health.GRPCServer
}

// New 创建 Gateway 服务器.
func New(opts ...Option) *Server {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	if o.logger == nil {
		panic("gateway: logger is required")
	}

	muxOpts := []runtime.ServeMuxOption{
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions:   o.marshalOptions,
			UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: true},
		}),
	}
	muxOpts = append(muxOpts, o.serveMuxOpts...)

	// 创建内置健康检查管理器
	healthOpts := []health.Option{health.WithTimeout(o.healthTimeout)}
	healthOpts = append(healthOpts, o.healthOptions...)
	h := health.New(healthOpts...)

	return &Server{
		opts:   o,
		mux:    runtime.NewServeMux(muxOpts...),
		health: h,
	}
}

// Register 注册服务，支持链式调用.
func (s *Server) Register(services ...Registrar) *Server {
	s.opts.services = append(s.opts.services, services...)
	return s
}

// Start 启动服务器.
func (s *Server) Start(ctx context.Context) error {
	if err := s.startGRPC(); err != nil {
		return err
	}
	if err := s.connectGateway(); err != nil {
		return err
	}
	return s.startHTTP(ctx)
}

// Stop 停止服务器.
func (s *Server) Stop(ctx context.Context) error {
	var lastErr error

	if s.httpServer != nil {
		s.opts.logger.With(logger.String("name", s.opts.name)).Info("[Gateway] HTTP server stopping")
		if err := s.httpServer.Shutdown(ctx); err != nil {
			lastErr = err
		}
	}

	if s.conn != nil {
		s.conn.Close()
	}

	if s.grpcServer != nil {
		s.opts.logger.With(logger.String("name", s.opts.name)).Info("[Gateway] gRPC server stopping")
		done := make(chan struct{})
		go func() {
			s.grpcServer.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			s.grpcServer.Stop()
			lastErr = ctx.Err()
		}
	}

	return lastErr
}

// Name 返回服务器名称.
func (s *Server) Name() string {
	return s.opts.name
}

// Addr 返回 gRPC 地址.
func (s *Server) Addr() string {
	return s.opts.grpcAddr
}

// HTTPAddr 返回 HTTP 地址.
func (s *Server) HTTPAddr() string {
	return s.opts.httpAddr
}

// GRPCServer 返回底层 gRPC Server.
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpcServer
}

// Mux 返回底层 ServeMux.
func (s *Server) Mux() *runtime.ServeMux {
	return s.mux
}

// Health 返回健康检查管理器.
func (s *Server) Health() *health.Health {
	return s.health
}

// HealthEndpoint 返回健康检查端点信息.
//
// Gateway 使用 HTTP 健康检查（通过 HTTP 端口）.
func (s *Server) HealthEndpoint() *transport.HealthEndpoint {
	return &transport.HealthEndpoint{
		Type: transport.HealthCheckTypeHTTP,
		Addr: s.opts.httpAddr,
		Path: health.DefaultLivenessPath,
	}
}

// HealthServer 返回 gRPC 健康检查服务器.
func (s *Server) HealthServer() *health.GRPCServer {
	return s.healthServer
}

func (s *Server) startGRPC() error {
	lis, err := net.Listen("tcp", s.opts.grpcAddr)
	if err != nil {
		return err
	}
	s.grpcListener = lis

	serverOpts := []grpc.ServerOption{
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             s.opts.minPingInterval,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    s.opts.keepaliveTime,
			Timeout: s.opts.keepaliveTimeout,
		}),
	}
	if len(s.opts.unaryInterceptors) > 0 {
		serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(s.opts.unaryInterceptors...))
	}
	if len(s.opts.streamInterceptors) > 0 {
		serverOpts = append(serverOpts, grpc.ChainStreamInterceptor(s.opts.streamInterceptors...))
	}
	serverOpts = append(serverOpts, s.opts.grpcServerOpts...)

	s.grpcServer = grpc.NewServer(serverOpts...)

	// 注册业务服务
	for _, svc := range s.opts.services {
		svc.RegisterGRPC(s.grpcServer)
	}

	// 注册 gRPC 健康检查服务
	s.healthServer = health.NewGRPCServer(s.health)
	s.healthServer.Register(s.grpcServer)

	if s.opts.enableReflection {
		reflection.Register(s.grpcServer)
	}

	s.opts.logger.With(
		logger.String("name", s.opts.name),
		logger.String("addr", s.opts.grpcAddr),
	).Info("[Gateway] gRPC 服务启动")

	go s.grpcServer.Serve(lis)
	return nil
}

func (s *Server) connectGateway() error {
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	dialOpts = append(dialOpts, s.opts.dialOptions...)

	conn, err := grpc.NewClient(s.opts.grpcAddr, dialOpts...)
	if err != nil {
		return err
	}
	s.conn = conn

	ctx := context.Background()
	for _, svc := range s.opts.services {
		if err := svc.RegisterGateway(ctx, s.mux, conn); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) startHTTP(ctx context.Context) error {
	// 使用健康检查中间件包装 mux
	var handler http.Handler = health.Middleware(s.health)(s.mux)

	// 如果启用链路追踪，使用 trace 中间件包装（在最外层）
	if s.opts.tracerName != "" {
		handler = tracing.HTTPMiddleware(s.opts.tracerName)(handler)
	}

	s.httpHandler = handler

	s.httpServer = &http.Server{
		Addr:         s.opts.httpAddr,
		Handler:      handler,
		ReadTimeout:  s.opts.httpReadTimeout,
		WriteTimeout: s.opts.httpWriteTimeout,
		IdleTimeout:  s.opts.httpIdleTimeout,
	}

	s.opts.logger.With(
		logger.String("name", s.opts.name),
		logger.String("addr", s.opts.httpAddr),
	).Info("[Gateway] HTTP 服务启动")

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	default:
	}
	return nil
}

// 确保 Server 实现了 transport.HealthCheckable 接口.
var _ transport.HealthCheckable = (*Server)(nil)
