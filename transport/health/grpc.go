package health

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// 服务名常量.
const (
	// ServiceLiveness 存活检查服务名.
	ServiceLiveness = "liveness"
	// ServiceReadiness 就绪检查服务名.
	ServiceReadiness = "readiness"
)

// GRPCServer gRPC 健康检查服务实现.
//
// 实现 grpc.health.v1.Health 服务接口.
type GRPCServer struct {
	grpc_health_v1.UnimplementedHealthServer

	mu     sync.RWMutex
	health *Health

	// statusOverrides 允许手动覆盖特定服务的状态
	statusOverrides map[string]grpc_health_v1.HealthCheckResponse_ServingStatus
}

// NewGRPCServer 创建 gRPC 健康检查服务.
func NewGRPCServer(h *Health) *GRPCServer {
	return &GRPCServer{
		health:          h,
		statusOverrides: make(map[string]grpc_health_v1.HealthCheckResponse_ServingStatus),
	}
}

// Check 实现 grpc.health.v1.Health.Check 方法.
//
// 根据请求的 service 名称返回对应的健康状态:
//   - 空字符串或 "readiness": 执行就绪检查
//   - "liveness": 执行存活检查
//   - 其他服务名: 检查是否有状态覆盖
func (s *GRPCServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	s.mu.RLock()
	// 检查是否有手动覆盖的状态
	if status, ok := s.statusOverrides[req.Service]; ok {
		s.mu.RUnlock()
		return &grpc_health_v1.HealthCheckResponse{Status: status}, nil
	}
	s.mu.RUnlock()

	// 根据服务名执行对应检查
	var resp Response
	switch req.Service {
	case "", ServiceReadiness:
		resp = s.health.Readiness(ctx)
	case ServiceLiveness:
		resp = s.health.Liveness(ctx)
	default:
		// 未知服务，返回 NOT_FOUND
		return nil, status.Error(codes.NotFound, "unknown service")
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: s.convertStatus(resp.Status),
	}, nil
}

// Watch 实现 grpc.health.v1.Health.Watch 方法.
//
// 注意：当前实现为简化版本，立即返回当前状态后结束流.
// 如需真正的流式监控，需要实现更复杂的状态变更通知机制.
func (s *GRPCServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	// 简化实现：返回当前状态
	resp, err := s.Check(stream.Context(), req)
	if err != nil {
		return err
	}

	if err := stream.Send(resp); err != nil {
		return err
	}

	// 等待上下文取消
	<-stream.Context().Done()
	return stream.Context().Err()
}

// SetServingStatus 手动设置特定服务的状态.
//
// 用于在维护期间或特殊情况下手动控制服务状态.
func (s *GRPCServer) SetServingStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusOverrides[service] = status
}

// ClearServingStatus 清除特定服务的手动状态，恢复使用检查器.
func (s *GRPCServer) ClearServingStatus(service string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.statusOverrides, service)
}

// Register 注册到 gRPC 服务器.
func (s *GRPCServer) Register(server *grpc.Server) {
	grpc_health_v1.RegisterHealthServer(server, s)
}

// convertStatus 将内部状态转换为 gRPC 健康检查状态.
func (s *GRPCServer) convertStatus(status Status) grpc_health_v1.HealthCheckResponse_ServingStatus {
	switch status {
	case StatusUp:
		return grpc_health_v1.HealthCheckResponse_SERVING
	case StatusDown:
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING
	default:
		return grpc_health_v1.HealthCheckResponse_UNKNOWN
	}
}

// RegisterGRPC 实现 grpc server 的 Registrar 接口.
func (s *GRPCServer) RegisterGRPC(server *grpc.Server) {
	s.Register(server)
}

// RegisterGRPCToServiceRegistrar 注册到 grpc.ServiceRegistrar.
func (s *GRPCServer) RegisterGRPCToServiceRegistrar(registrar grpc.ServiceRegistrar) {
	grpc_health_v1.RegisterHealthServer(registrar, s)
}
