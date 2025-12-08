// Package server 提供 gRPC 服务器实现.
package server

import (
	"context"
	"net"

	"github.com/Tsukikage7/microservice-kit/transport"
	"github.com/Tsukikage7/microservice-kit/transport/health"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// Registrar gRPC 服务注册器接口.
type Registrar interface {
	RegisterGRPC(server *grpc.Server)
}

// Server gRPC 服务器.
type Server struct {
	opts     *options
	server   *grpc.Server
	listener net.Listener

	// 内置健康检查
	health       *health.Health
	healthServer *health.GRPCServer
}

// New 创建 gRPC 服务器，如果未设置 logger 会 panic.
func New(opts ...Option) *Server {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	if o.logger == nil {
		panic("grpc server: 必须设置 logger")
	}

	// 创建内置健康检查管理器
	healthOpts := []health.Option{health.WithTimeout(o.healthTimeout)}
	healthOpts = append(healthOpts, o.healthOptions...)
	h := health.New(healthOpts...)

	return &Server{
		opts:   o,
		health: h,
	}
}

// Register 注册 gRPC 服务，支持链式调用.
func (s *Server) Register(services ...Registrar) *Server {
	s.opts.services = append(s.opts.services, services...)
	return s
}

// GRPCServer 返回底层 grpc.Server，未启动时返回 nil.
func (s *Server) GRPCServer() *grpc.Server {
	return s.server
}

// Health 返回健康检查管理器.
func (s *Server) Health() *health.Health {
	return s.health
}

// HealthEndpoint 返回健康检查端点信息.
func (s *Server) HealthEndpoint() *transport.HealthEndpoint {
	return &transport.HealthEndpoint{
		Type: transport.HealthCheckTypeGRPC,
		Addr: s.opts.addr,
	}
}

// HealthServer 返回 gRPC 健康检查服务器.
func (s *Server) HealthServer() *health.GRPCServer {
	return s.healthServer
}

// Start 启动 gRPC 服务器.
func (s *Server) Start(ctx context.Context) error {
	// 创建监听器
	listener, err := net.Listen("tcp", s.opts.addr)
	if err != nil {
		return err
	}
	s.listener = listener

	// 构建服务器选项
	serverOpts := s.buildServerOptions()

	// 创建 gRPC 服务器
	s.server = grpc.NewServer(serverOpts...)

	// 注册所有业务服务
	for _, service := range s.opts.services {
		service.RegisterGRPC(s.server)
	}

	// 注册 gRPC 健康检查服务
	s.healthServer = health.NewGRPCServer(s.health)
	s.healthServer.Register(s.server)

	// 启用反射
	if s.opts.enableReflection {
		reflection.Register(s.server)
	}

	s.opts.logger.Infof("[%s] 服务器启动 [addr:%s]", s.opts.name, s.opts.addr)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.Serve(listener)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		// 上下文取消，正常退出
	}

	return nil
}

// Stop 停止 gRPC 服务器.
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	s.opts.logger.Infof("[%s] 服务器停止中", s.opts.name)

	// 优雅关闭
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		// 超时，强制关闭
		s.server.Stop()
		return ctx.Err()
	}
}

// Name 返回服务器名称.
func (s *Server) Name() string {
	return s.opts.name
}

// Addr 返回服务器地址.
func (s *Server) Addr() string {
	return s.opts.addr
}

// buildServerOptions 构建 gRPC 服务器选项.
func (s *Server) buildServerOptions() []grpc.ServerOption {
	opts := []grpc.ServerOption{
		// Keepalive 执行策略（防止客户端 ping 过于频繁）
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             s.opts.minPingInterval,
			PermitWithoutStream: true,
		}),
		// Keepalive 服务端参数（主动检测客户端健康）
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    s.opts.keepaliveTime,
			Timeout: s.opts.keepaliveTimeout,
		}),
	}

	// 添加拦截器
	if len(s.opts.unaryInterceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(s.opts.unaryInterceptors...))
	}

	if len(s.opts.streamInterceptors) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(s.opts.streamInterceptors...))
	}

	// 添加自定义选项
	opts = append(opts, s.opts.serverOptions...)

	return opts
}

// 确保 Server 实现了 transport.HealthCheckable 接口.
var _ transport.HealthCheckable = (*Server)(nil)
