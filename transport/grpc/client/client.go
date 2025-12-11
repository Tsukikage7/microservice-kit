// Package client 提供 gRPC 客户端工具.
package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// Client gRPC 客户端封装.
type Client struct {
	conn *grpc.ClientConn
	opts *options
}

// New 创建 gRPC 客户端，必需设置 serviceName、discovery、logger，否则会 panic.
func New(opts ...Option) (*Client, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	// 验证必需参数
	if o.serviceName == "" {
		panic("grpc client: 必须设置 serviceName")
	}
	if o.discovery == nil {
		panic("grpc client: 必须设置 discovery")
	}
	if o.logger == nil {
		panic("grpc client: 必须设置 logger")
	}

	// 服务发现
	addrs, err := o.discovery.Discover(context.Background(), o.serviceName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDiscoveryFailed, err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrServiceNotFound, o.serviceName)
	}
	target := addrs[0]

	// 构建 dial 选项
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                60 * time.Second,
			Timeout:             20 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		),
	}

	// 添加自定义拦截器
	if len(o.interceptors) > 0 {
		dialOpts = append(dialOpts, grpc.WithChainUnaryInterceptor(o.interceptors...))
	}

	dialOpts = append(dialOpts, o.dialOptions...)

	// 创建连接
	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	o.logger.With(
		logger.String("name", o.name),
		logger.String("service", o.serviceName),
		logger.String("target", target),
	).Info("[gRPC] 客户端初始化成功")

	return &Client{
		conn: conn,
		opts: o,
	}, nil
}

// Conn 返回底层 gRPC 连接.
func (c *Client) Conn() *grpc.ClientConn {
	return c.conn
}

// Close 关闭连接.
func (c *Client) Close() error {
	if c.conn != nil {
		c.opts.logger.With(
			logger.String("name", c.opts.name),
			logger.String("service", c.opts.serviceName),
		).Info("[gRPC] 关闭连接")
		return c.conn.Close()
	}
	return nil
}
