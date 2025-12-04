package metrics

// Config 指标监控配置.
type Config struct {
	// Path 指标暴露路径，默认 /metrics
	Path string `json:"path" yaml:"path" mapstructure:"path"`
	// Namespace 指标命名空间
	Namespace string `json:"namespace" yaml:"namespace" mapstructure:"namespace"`
}

// DefaultConfig 返回默认配置.
func DefaultConfig() *Config {
	return &Config{
		Path:      "/metrics",
		Namespace: "app",
	}
}
