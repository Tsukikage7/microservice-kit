package messaging

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

// RabbitMQ 集成测试
// 需要设置环境变量 RABBITMQ_URL 指向 RabbitMQ 服务器
// 例如: export RABBITMQ_URL=amqp://guest:guest@localhost:5672/

type RabbitMQTestSuite struct {
	suite.Suite
	url string
}

func TestRabbitMQSuite(t *testing.T) {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		t.Skip("RABBITMQ_URL not set, skipping integration tests")
	}
	suite.Run(t, &RabbitMQTestSuite{url: url})
}

func (s *RabbitMQTestSuite) TestProducerCreate() {
	cfg := &Config{
		Type: "rabbitmq",
		URL:  s.url,
	}
	producer, err := NewProducer(cfg)
	s.Require().NoError(err)
	s.NotNil(producer)

	defer producer.Close()
}

func (s *RabbitMQTestSuite) TestProducerWithExchange() {
	cfg := &Config{
		Type: "rabbitmq",
		URL:  s.url,
		RabbitMQ: &RabbitMQConfig{
			Exchange:     "test_exchange",
			ExchangeType: "direct",
			Durable:      false,
		},
	}
	producer, err := NewProducer(cfg)
	s.Require().NoError(err)
	s.NotNil(producer)

	defer producer.Close()
}

func (s *RabbitMQTestSuite) TestConsumerCreate() {
	cfg := &Config{
		Type: "rabbitmq",
		URL:  s.url,
	}
	consumer, err := NewConsumer(cfg, "test-group")
	s.Require().NoError(err)
	s.NotNil(consumer)

	defer consumer.Close()
}

func (s *RabbitMQTestSuite) TestConsumerEmptyGroupID() {
	cfg := &Config{
		Type: "rabbitmq",
		URL:  s.url,
	}
	_, err := NewConsumer(cfg, "")
	s.Error(err)
	s.ErrorIs(err, ErrEmptyGroupID)
}

func (s *RabbitMQTestSuite) TestConsumerWithOptions() {
	cfg := &Config{
		Type: "rabbitmq",
		URL:  s.url,
		RabbitMQ: &RabbitMQConfig{
			Exchange:      "test_exchange",
			ExchangeType:  "direct",
			Durable:       false,
			AutoAck:       true,
			PrefetchCount: 5,
		},
	}
	consumer, err := NewConsumer(cfg, "test-group")
	s.Require().NoError(err)
	s.NotNil(consumer)

	defer consumer.Close()
}

// 单元测试 - 不需要 RabbitMQ 服务器

type RabbitMQUnitTestSuite struct {
	suite.Suite
}

func TestRabbitMQUnitSuite(t *testing.T) {
	suite.Run(t, new(RabbitMQUnitTestSuite))
}

func (s *RabbitMQUnitTestSuite) TestConfigWithRabbitMQ() {
	cfg := &Config{
		Type: "rabbitmq",
		URL:  "amqp://localhost:5672",
		RabbitMQ: &RabbitMQConfig{
			Exchange:      "test",
			ExchangeType:  "direct",
			Durable:       true,
			AutoAck:       false,
			PrefetchCount: 10,
			Confirm:       true,
		},
	}

	s.Equal("rabbitmq", cfg.Type)
	s.Equal("amqp://localhost:5672", cfg.URL)
	s.NotNil(cfg.RabbitMQ)
	s.Equal("test", cfg.RabbitMQ.Exchange)
}

func (s *RabbitMQUnitTestSuite) TestNewProducerWithRabbitMQType() {
	cfg := &Config{
		Type: "rabbitmq",
		URL:  "", // 空 URL 应该返回错误
	}

	_, err := NewProducer(cfg)
	s.Error(err)
	s.ErrorIs(err, ErrNoBrokers)
}

func (s *RabbitMQUnitTestSuite) TestNewConsumerWithRabbitMQType() {
	cfg := &Config{
		Type: "rabbitmq",
		URL:  "", // 空 URL 应该返回错误
	}

	_, err := NewConsumer(cfg, "test-group")
	s.Error(err)
	s.ErrorIs(err, ErrNoBrokers)
}

func (s *RabbitMQUnitTestSuite) TestNewProducerUnsupportedType() {
	cfg := &Config{
		Type: "unsupported",
	}

	_, err := NewProducer(cfg)
	s.Error(err)
	s.ErrorIs(err, ErrUnsupportedType)
}

func (s *RabbitMQUnitTestSuite) TestNewConsumerUnsupportedType() {
	cfg := &Config{
		Type: "unsupported",
	}

	_, err := NewConsumer(cfg, "test-group")
	s.Error(err)
	s.ErrorIs(err, ErrUnsupportedType)
}

func (s *RabbitMQUnitTestSuite) TestExchangeTypes() {
	// 测试交换机类型常量
	s.Equal(exchangeType("direct"), exchangeDirect)
	s.Equal(exchangeType("fanout"), exchangeFanout)
	s.Equal(exchangeType("topic"), exchangeTopic)
	s.Equal(exchangeType("headers"), exchangeHeaders)
}
