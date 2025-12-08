// Package transport 提供传输层抽象.
package transport

import "github.com/Tsukikage7/microservice-kit/transport/health"

// HealthCheckType 健康检查类型.
type HealthCheckType string

const (
	// HealthCheckTypeTCP TCP 端口检查.
	HealthCheckTypeTCP HealthCheckType = "tcp"
	// HealthCheckTypeHTTP HTTP 端点检查.
	HealthCheckTypeHTTP HealthCheckType = "http"
	// HealthCheckTypeGRPC gRPC 健康检查.
	HealthCheckTypeGRPC HealthCheckType = "grpc"
)

// HealthEndpoint 健康检查端点信息.
type HealthEndpoint struct {
	Type HealthCheckType // 检查类型: tcp, http, grpc
	Addr string          // 检查地址 (host:port)
	Path string          // HTTP 路径 (仅 HTTP 类型)
}

// HealthCheckable 可选接口，表示 Server 支持健康检查.
type HealthCheckable interface {
	Server
	// Health 返回健康检查管理器.
	Health() *health.Health
	// HealthEndpoint 返回健康检查端点信息.
	HealthEndpoint() *HealthEndpoint
}
