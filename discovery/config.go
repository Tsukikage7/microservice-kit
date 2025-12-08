package discovery

import "fmt"

// 服务发现类型常量.
const (
	TypeConsul = "consul"
)

// 协议类型常量.
const (
	ProtocolHTTP = "http"
	ProtocolGRPC = "grpc"
)

// 健康检查内部默认值（不对外暴露配置）.
const (
	defaultHealthCheckInterval        = "10s"
	defaultHealthCheckTimeout         = "3s"
	defaultHealthCheckDeregisterAfter = "30s"
	defaultHealthCheckHTTPPath        = "/healthz"
)

// 服务元数据默认值.
const (
	DefaultVersion = "1.0.0"
)

// Config 服务发现配置.
type Config struct {
	Type string `json:"type" toml:"type" yaml:"type" mapstructure:"type"` // 服务发现类型
	Addr string `json:"addr" toml:"addr" yaml:"addr" mapstructure:"addr"` // 服务发现地址

	// 协议特定的服务元数据配置
	Services ServiceConfig `json:"services" toml:"services" yaml:"services" mapstructure:"services"`
}

// ServiceMetaConfig 服务元数据配置.
type ServiceMetaConfig struct {
	Version  string   `json:"version" toml:"version" yaml:"version" mapstructure:"version"`     // 服务版本
	Protocol string   `json:"protocol" toml:"protocol" yaml:"protocol" mapstructure:"protocol"` // 协议类型
	Tags     []string `json:"tags" toml:"tags" yaml:"tags" mapstructure:"tags"`                 // 服务标签
}

// ServiceConfig 服务配置.
type ServiceConfig struct {
	HTTP ServiceMetaConfig `json:"http" toml:"http" yaml:"http" mapstructure:"http"` // HTTP服务元数据
	GRPC ServiceMetaConfig `json:"grpc" toml:"grpc" yaml:"grpc" mapstructure:"grpc"` // gRPC服务元数据
}

// Validate 验证配置.
func (c *Config) Validate() error {
	if c == nil {
		return ErrNilConfig
	}
	if c.Type == "" {
		return ErrEmptyType
	}
	if c.Type != TypeConsul {
		return ErrUnsupportedType
	}
	return nil
}

// SetDefaults 设置默认配置.
func (c *Config) SetDefaults() {
	// HTTP服务默认配置
	if c.Services.HTTP.Version == "" {
		c.Services.HTTP.Version = DefaultVersion
	}
	if c.Services.HTTP.Protocol == "" {
		c.Services.HTTP.Protocol = ProtocolHTTP
	}
	if len(c.Services.HTTP.Tags) == 0 {
		c.Services.HTTP.Tags = []string{"http", "v1"}
	}

	// gRPC服务默认配置
	if c.Services.GRPC.Version == "" {
		c.Services.GRPC.Version = DefaultVersion
	}
	if c.Services.GRPC.Protocol == "" {
		c.Services.GRPC.Protocol = ProtocolGRPC
	}
	if len(c.Services.GRPC.Tags) == 0 {
		c.Services.GRPC.Tags = []string{"grpc", "v1"}
	}
}

// GetServiceConfig 获取协议特定的服务配置.
func (c *Config) GetServiceConfig(protocol string) ServiceMetaConfig {
	switch protocol {
	case ProtocolHTTP:
		return c.Services.HTTP
	case ProtocolGRPC:
		return c.Services.GRPC
	default:
		// 不支持的协议返回空配置
		return ServiceMetaConfig{}
	}
}

// ConfigError 配置错误.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("配置错误 [%s]: %s", e.Field, e.Message)
}
