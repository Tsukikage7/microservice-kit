package messaging

// Config 消息队列配置.
//
// 用于初始化生产者和消费者，支持多种消息队列实现.
//
// Kafka 示例:
//
//	cfg := &messaging.Config{
//	    Type:    "kafka",
//	    Brokers: []string{"localhost:9092", "localhost:9093"},
//	}
//
// RabbitMQ 示例:
//
//	cfg := &messaging.Config{
//	    Type: "rabbitmq",
//	    URL:  "amqp://user:pass@localhost:5672/vhost",
//	}
type Config struct {
	// Type 消息队列类型.
	// 支持: kafka（默认）、rabbitmq.
	Type string `json:"type" yaml:"type" mapstructure:"type"`

	// Brokers 服务器地址列表（Kafka 使用）.
	// 格式为 host:port，例如 "localhost:9092".
	Brokers []string `json:"brokers" yaml:"brokers" mapstructure:"brokers"`

	// URL 连接地址（RabbitMQ 使用）.
	// 格式: amqp://user:pass@host:port/vhost
	URL string `json:"url" yaml:"url" mapstructure:"url"`

	// RabbitMQ 特定配置
	RabbitMQ *RabbitMQConfig `json:"rabbitmq" yaml:"rabbitmq" mapstructure:"rabbitmq"`
}

// RabbitMQConfig RabbitMQ 特定配置.
type RabbitMQConfig struct {
	// Exchange 交换机名称.
	Exchange string `json:"exchange" yaml:"exchange" mapstructure:"exchange"`

	// ExchangeType 交换机类型: direct, fanout, topic, headers.
	ExchangeType string `json:"exchange_type" yaml:"exchange_type" mapstructure:"exchange_type"`

	// Durable 是否持久化.
	Durable bool `json:"durable" yaml:"durable" mapstructure:"durable"`

	// AutoAck 是否自动确认.
	AutoAck bool `json:"auto_ack" yaml:"auto_ack" mapstructure:"auto_ack"`

	// PrefetchCount 预取数量.
	PrefetchCount int `json:"prefetch_count" yaml:"prefetch_count" mapstructure:"prefetch_count"`

	// Confirm 是否启用发布确认.
	Confirm bool `json:"confirm" yaml:"confirm" mapstructure:"confirm"`
}
