// Package discovery 提供服务发现功能.
package discovery

import (
	"context"
	"strings"
	"sync"

	"github.com/Tsukikage7/microservice-kit/logger"
	"github.com/Tsukikage7/microservice-kit/transport"
)

// ServiceInfo 服务信息.
type ServiceInfo struct {
	Name           string                   // 服务名称
	Addr           string                   // 服务地址 (host:port)
	Protocol       string                   // 协议类型 (grpc/http)
	HealthEndpoint *transport.HealthEndpoint // 健康检查端点 (可选，自动检测)
}

// ServiceRegistry 服务注册器，管理多个服务的注册和注销.
type ServiceRegistry struct {
	discovery  Discovery
	logger     logger.Logger
	services   []ServiceInfo
	serviceIDs []string
	mu         sync.Mutex
}

// NewServiceRegistry 创建服务注册器.
func NewServiceRegistry(discovery Discovery, logger logger.Logger) *ServiceRegistry {
	return &ServiceRegistry{
		discovery:  discovery,
		logger:     logger,
		services:   make([]ServiceInfo, 0),
		serviceIDs: make([]string, 0),
	}
}

// AddService 添加要注册的服务.
func (r *ServiceRegistry) AddService(name, addr, protocol string) *ServiceRegistry {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services = append(r.services, ServiceInfo{
		Name:     name,
		Addr:     addr,
		Protocol: protocol,
	})
	return r
}

// AddGRPC 添加 gRPC 服务.
func (r *ServiceRegistry) AddGRPC(name, addr string) *ServiceRegistry {
	return r.AddService(name, addr, ProtocolGRPC)
}

// AddHTTP 添加 HTTP 服务.
func (r *ServiceRegistry) AddHTTP(name, addr string) *ServiceRegistry {
	return r.AddService(name, addr, ProtocolHTTP)
}

// AddServer 从 transport.Server 添加服务，自动检测健康检查端点.
// 如果 Server 实现了 HealthCheckable 接口，将自动提取健康检查配置.
func (r *ServiceRegistry) AddServer(name string, server transport.Server) *ServiceRegistry {
	r.mu.Lock()
	defer r.mu.Unlock()

	info := ServiceInfo{
		Name: name,
		Addr: server.Addr(),
	}

	// 根据 Server 名称推断协议类型
	info.Protocol = r.inferProtocol(server)

	// 如果 Server 实现了 HealthCheckable 接口，提取健康检查端点
	if hc, ok := server.(transport.HealthCheckable); ok {
		info.HealthEndpoint = hc.HealthEndpoint()
	}

	r.services = append(r.services, info)
	return r
}

// AddServerWithProtocol 从 transport.Server 添加服务，指定协议类型.
func (r *ServiceRegistry) AddServerWithProtocol(name string, server transport.Server, protocol string) *ServiceRegistry {
	r.mu.Lock()
	defer r.mu.Unlock()

	info := ServiceInfo{
		Name:     name,
		Addr:     server.Addr(),
		Protocol: protocol,
	}

	// 如果 Server 实现了 HealthCheckable 接口，提取健康检查端点
	if hc, ok := server.(transport.HealthCheckable); ok {
		info.HealthEndpoint = hc.HealthEndpoint()
	}

	r.services = append(r.services, info)
	return r
}

// inferProtocol 根据 Server 类型推断协议.
func (r *ServiceRegistry) inferProtocol(server transport.Server) string {
	// 尝试从健康检查端点推断
	if hc, ok := server.(transport.HealthCheckable); ok {
		endpoint := hc.HealthEndpoint()
		if endpoint != nil {
			switch endpoint.Type {
			case transport.HealthCheckTypeGRPC:
				return ProtocolGRPC
			case transport.HealthCheckTypeHTTP:
				return ProtocolHTTP
			}
		}
	}

	// 根据 Server 名称推断
	name := server.Name()
	switch {
	case containsIgnoreCase(name, "grpc"):
		return ProtocolGRPC
	case containsIgnoreCase(name, "http"):
		return ProtocolHTTP
	case containsIgnoreCase(name, "gateway"):
		// Gateway 服务默认使用 gRPC 协议进行服务发现
		return ProtocolGRPC
	default:
		return ProtocolGRPC
	}
}

// RegisterAll 注册所有服务.
func (r *ServiceRegistry) RegisterAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, svc := range r.services {
		var serviceID string
		var err error

		// 如果有健康检查端点，使用新的注册方法
		if svc.HealthEndpoint != nil {
			serviceID, err = r.discovery.RegisterWithHealthEndpoint(ctx, svc.Name, svc.Addr, svc.Protocol, svc.HealthEndpoint)
		} else {
			serviceID, err = r.discovery.RegisterWithProtocol(ctx, svc.Name, svc.Addr, svc.Protocol)
		}

		if err != nil {
			r.logger.Errorf("注册服务失败 [服务名:%s] [地址:%s] [协议:%s] [错误:%v]",
				svc.Name, svc.Addr, svc.Protocol, err)
			// 回滚已注册的服务
			r.unregisterAllLocked(ctx)
			return err
		}
		r.serviceIDs = append(r.serviceIDs, serviceID)

		healthType := "TCP"
		if svc.HealthEndpoint != nil {
			healthType = string(svc.HealthEndpoint.Type)
		}
		r.logger.Infof("服务已注册 [服务名:%s] [地址:%s] [协议:%s] [健康检查:%s] [服务ID:%s]",
			svc.Name, svc.Addr, svc.Protocol, healthType, serviceID)
	}
	return nil
}

// UnregisterAll 注销所有服务.
func (r *ServiceRegistry) UnregisterAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.unregisterAllLocked(ctx)
}

// unregisterAllLocked 注销所有服务（内部方法，需要持有锁）.
func (r *ServiceRegistry) unregisterAllLocked(ctx context.Context) error {
	var lastErr error
	for _, serviceID := range r.serviceIDs {
		if err := r.discovery.Unregister(ctx, serviceID); err != nil {
			r.logger.Errorf("注销服务失败 [服务ID:%s] [错误:%v]", serviceID, err)
			lastErr = err
		} else {
			r.logger.Infof("服务已注销 [服务ID:%s]", serviceID)
		}
	}
	r.serviceIDs = r.serviceIDs[:0]
	return lastErr
}

// AfterStartHook 返回服务启动后的注册钩子.
// 可直接用于 transport.HooksBuilder.AfterStart()
func (r *ServiceRegistry) AfterStartHook() func(ctx context.Context) error {
	return r.RegisterAll
}

// BeforeStopHook 返回服务停止前的注销钩子.
// 可直接用于 transport.HooksBuilder.BeforeStop()
func (r *ServiceRegistry) BeforeStopHook() func(ctx context.Context) error {
	return r.UnregisterAll
}

// containsIgnoreCase 检查字符串是否包含子串（忽略大小写）.
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
