package server

import (
	"context"
	"net"

	"github.com/Tsukikage7/microservice-kit/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// GRPCRegistrar gRPC 服务注册器接口.
type GRPCRegistrar interface {
	RegisterGRPC(server *grpc.Server)
}

// GRPC gRPC 服务器.
type GRPC struct {
	opts     *grpcOptions
	server   *grpc.Server
	listener net.Listener
}

// NewGRPC 创建 gRPC 服务器.
//
// 示例:
//
//	srv := server.NewGRPC(
//	    server.WithGRPCAddr(":9090"),
//	    server.WithGRPCLogger(log),
//	)
//	srv.Register(userService)
func NewGRPC(opts ...GRPCOption) *GRPC {
	o := defaultGRPCOptions()
	for _, opt := range opts {
		opt(o)
	}

	return &GRPC{
		opts: o,
	}
}

// Register 注册 gRPC 服务.
//
// 支持链式调用.
func (s *GRPC) Register(services ...GRPCRegistrar) *GRPC {
	s.opts.services = append(s.opts.services, services...)
	return s
}

// Server 获取底层 grpc.Server.
//
// 如果服务器尚未启动，返回 nil.
func (s *GRPC) Server() *grpc.Server {
	return s.server
}

// Start 启动 gRPC 服务器.
func (s *GRPC) Start(ctx context.Context) error {
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

	// 注册所有服务
	for _, service := range s.opts.services {
		service.RegisterGRPC(s.server)
	}

	// 启用反射
	if s.opts.enableReflection {
		reflection.Register(s.server)
	}

	s.logDebugf("gRPC 服务器启动 [addr:%s]", s.opts.addr)

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
func (s *GRPC) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	s.logDebug("gRPC 服务器停止中...")

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
func (s *GRPC) Name() string {
	return "grpc"
}

// Addr 返回服务器地址.
func (s *GRPC) Addr() string {
	return s.opts.addr
}

// buildServerOptions 构建 gRPC 服务器选项.
func (s *GRPC) buildServerOptions() []grpc.ServerOption {
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

// 日志辅助方法.

func (s *GRPC) logger() logger.Logger {
	return s.opts.logger
}

func (s *GRPC) logDebug(msg string) {
	if log := s.logger(); log != nil {
		log.Debug("[gRPC] " + msg)
	}
}

func (s *GRPC) logDebugf(format string, args ...any) {
	if log := s.logger(); log != nil {
		log.Debugf("[gRPC] "+format, args...)
	}
}
