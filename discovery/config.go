package discovery

// TypeConsul 表示 Consul 服务发现类型.
const TypeConsul = "consul"

// 协议类型常量.
const (
	// ProtocolHTTP 表示 HTTP 协议.
	ProtocolHTTP = "http"
	// ProtocolGRPC 表示 gRPC 协议.
	ProtocolGRPC = "grpc"
)

// 健康检查内部默认值.
const (
	defaultHealthCheckInterval        = "10s"
	defaultHealthCheckTimeout         = "3s"
	defaultHealthCheckDeregisterAfter = "30s"
	defaultHealthCheckHTTPPath        = "/healthz"
)

// DefaultVersion 是默认服务版本.
const DefaultVersion = "1.0.0"

// Config 表示服务发现配置.
type Config struct {
	Type     string        `json:"type" toml:"type" yaml:"type" mapstructure:"type"`
	Addr     string        `json:"addr" toml:"addr" yaml:"addr" mapstructure:"addr"`
	Services ServiceConfig `json:"services" toml:"services" yaml:"services" mapstructure:"services"`
}

// ServiceMetaConfig 表示服务元数据配置.
type ServiceMetaConfig struct {
	Version  string   `json:"version" toml:"version" yaml:"version" mapstructure:"version"`
	Protocol string   `json:"protocol" toml:"protocol" yaml:"protocol" mapstructure:"protocol"`
	Tags     []string `json:"tags" toml:"tags" yaml:"tags" mapstructure:"tags"`
}

// ServiceConfig 包含协议特定的服务配置.
type ServiceConfig struct {
	HTTP ServiceMetaConfig `json:"http" toml:"http" yaml:"http" mapstructure:"http"`
	GRPC ServiceMetaConfig `json:"grpc" toml:"grpc" yaml:"grpc" mapstructure:"grpc"`
}

// Validate 验证配置有效性.
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

// SetDefaults 设置配置的默认值.
func (c *Config) SetDefaults() {
	if c.Services.HTTP.Version == "" {
		c.Services.HTTP.Version = DefaultVersion
	}
	if c.Services.HTTP.Protocol == "" {
		c.Services.HTTP.Protocol = ProtocolHTTP
	}
	if len(c.Services.HTTP.Tags) == 0 {
		c.Services.HTTP.Tags = []string{"http", "v1"}
	}

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

// GetServiceConfig 返回指定协议的服务配置.
func (c *Config) GetServiceConfig(protocol string) ServiceMetaConfig {
	switch protocol {
	case ProtocolHTTP:
		return c.Services.HTTP
	case ProtocolGRPC:
		return c.Services.GRPC
	default:
		return ServiceMetaConfig{}
	}
}

