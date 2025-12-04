# Discovery 服务发现包

提供微服务架构中的服务注册与发现功能，支持 Consul 作为服务注册中心。

## 功能特性

- 服务注册与注销
- 服务发现
- 多协议支持（HTTP/gRPC）
- 健康检查配置
- 中文错误信息

## 安装

```bash
go get github.com/Tsukikage7/microservice-kit/discovery
```

## 快速开始

### 基础用法

```go
package main

import (
    "context"
    "log"

    "github.com/Tsukikage7/microservice-kit/discovery"
    "github.com/Tsukikage7/microservice-kit/logger"
)

func main() {
    // 创建日志记录器
    log, err := logger.NewLogger(&logger.Config{
        Level:  "debug",
        Format: "console",
    })
    if err != nil {
        panic(err)
    }
    defer log.Close()

    // 创建服务发现配置
    config := &discovery.Config{
        Type: discovery.TypeConsul,
        Addr: "localhost:8500",
    }

    // 创建服务发现实例
    d, err := discovery.New(config, log)
    if err != nil {
        panic(err)
    }
    defer d.Close()

    ctx := context.Background()

    // 注册服务
    serviceID, err := d.Register(ctx, "my-service", "localhost:8080")
    if err != nil {
        log.Errorf("服务注册失败: %v", err)
        return
    }
    log.Infof("服务注册成功, ID: %s", serviceID)

    // 发现服务
    addresses, err := d.Discover(ctx, "my-service")
    if err != nil {
        log.Errorf("服务发现失败: %v", err)
        return
    }
    log.Infof("发现服务实例: %v", addresses)

    // 注销服务
    if err := d.Unregister(ctx, serviceID); err != nil {
        log.Errorf("服务注销失败: %v", err)
        return
    }
}
```

### 按协议注册

```go
// 注册 gRPC 服务
grpcID, err := d.RegisterWithProtocol(ctx, "my-service", "localhost:9090", discovery.ProtocolGRPC)

// 注册 HTTP 服务
httpID, err := d.RegisterWithProtocol(ctx, "my-service", "localhost:8080", discovery.ProtocolHTTP)
```

### 自定义配置

```go
config := &discovery.Config{
    Type: discovery.TypeConsul,
    Addr: "consul.example.com:8500",
    HealthCheck: discovery.HealthCheckConfig{
        Interval:        "15s",
        Timeout:         "5s",
        DeregisterAfter: "60s",
    },
    Services: discovery.ServiceConfig{
        HTTP: discovery.ServiceMetaConfig{
            Version:  "2.0.0",
            Protocol: "http",
            Tags:     []string{"http", "api", "v2"},
        },
        GRPC: discovery.ServiceMetaConfig{
            Version:  "2.0.0",
            Protocol: "grpc",
            Tags:     []string{"grpc", "internal", "v2"},
        },
    },
}
```

### YAML 配置示例

```yaml
discovery:
  type: consul
  addr: localhost:8500
  health_check:
    interval: 10s
    timeout: 3s
    deregister_after: 30s
  services:
    http:
      version: "1.0.0"
      protocol: http
      tags:
        - http
        - api
        - v1
    grpc:
      version: "1.0.0"
      protocol: grpc
      tags:
        - grpc
        - internal
        - v1
```

## API 参考

### 类型常量

```go
const (
    TypeConsul = "consul"  // Consul 服务发现
)

const (
    ProtocolHTTP = "http"  // HTTP 协议
    ProtocolGRPC = "grpc"  // gRPC 协议
)
```

### 默认值

| 配置项 | 默认值 |
|--------|--------|
| 健康检查间隔 | 10s |
| 健康检查超时 | 3s |
| 失败后注销时间 | 30s |
| 服务版本 | 1.0.0 |

### 错误类型

| 错误 | 说明 |
|------|------|
| `ErrNilConfig` | 配置为空 |
| `ErrNilLogger` | 日志记录器为空 |
| `ErrEmptyName` | 服务名称为空 |
| `ErrEmptyAddress` | 服务地址为空 |
| `ErrEmptyServiceID` | 服务ID为空 |
| `ErrUnsupportedType` | 不支持的服务发现类型 |
| `ErrUnsupportedProtocol` | 不支持的协议类型 |
| `ErrInvalidAddress` | 无效的地址格式 |
| `ErrInvalidPort` | 无效的端口号 |
| `ErrNotFound` | 未发现任何服务实例 |

## 文件结构

```
discovery/
├── discovery.go      # 接口定义和错误常量
├── config.go         # 配置结构体
├── factory.go        # 工厂函数
├── consul.go         # Consul 实现
├── config_test.go    # 配置测试
├── consul_test.go    # Consul 测试
├── factory_test.go   # 工厂测试
├── discovery_test.go # 接口测试
└── README.md         # 文档
```

## 测试

```bash
# 运行测试
go test -v ./discovery/...

# 运行测试并查看覆盖率
go test -v ./discovery/... -cover

# 生成覆盖率报告
go test ./discovery/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

当前测试覆盖率：**87.8%**
