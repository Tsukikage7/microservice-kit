// Package transport 提供传输层抽象.
package transport

import "github.com/Tsukikage7/microservice-kit/transport/health"

// HealthCheckType 表示健康检查类型.
type HealthCheckType string

// 健康检查类型常量.
const (
	// HealthCheckTypeTCP 表示 TCP 端口检查.
	HealthCheckTypeTCP HealthCheckType = "tcp"
	// HealthCheckTypeHTTP 表示 HTTP 端点检查.
	HealthCheckTypeHTTP HealthCheckType = "http"
	// HealthCheckTypeGRPC 表示 gRPC 健康检查.
	HealthCheckTypeGRPC HealthCheckType = "grpc"
)

// HealthEndpoint 表示健康检查端点信息.
type HealthEndpoint struct {
	Type HealthCheckType
	Addr string
	Path string
}

// HealthCheckable 是可选接口，表示 Server 支持健康检查.
type HealthCheckable interface {
	Server
	// Health 返回健康检查管理器.
	Health() *health.Health
	// HealthEndpoint 返回健康检查端点信息.
	HealthEndpoint() *HealthEndpoint
}
