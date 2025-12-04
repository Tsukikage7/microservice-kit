package messaging

// Config 消息队列配置.
//
// 用于初始化生产者和消费者，支持多种消息队列实现.
//
// 示例:
//
//	cfg := &messaging.Config{
//	    Type:    "kafka",
//	    Brokers: []string{"localhost:9092", "localhost:9093"},
//	}
type Config struct {
	// Type 消息队列类型.
	// 目前支持: kafka（默认）.
	Type string `json:"type" yaml:"type" mapstructure:"type"`

	// Brokers 服务器地址列表.
	// 格式为 host:port，例如 "localhost:9092".
	Brokers []string `json:"brokers" yaml:"brokers" mapstructure:"brokers"`
}
