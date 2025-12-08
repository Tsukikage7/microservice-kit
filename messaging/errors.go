package messaging

import "errors"

// 预定义错误.
//
// 所有错误均可通过 errors.Is 进行判断:
//
//	if errors.Is(err, messaging.ErrProducerClosed) {
//	    // 处理生产者已关闭的情况
//	}
var (
	// ErrEmptyGroupID 消费者组ID为空.
	ErrEmptyGroupID = errors.New("messaging: 消费者组ID为空")

	// ErrProducerClosed 生产者已关闭.
	ErrProducerClosed = errors.New("messaging: 生产者已关闭")

	// ErrNilMessage 消息为空.
	ErrNilMessage = errors.New("messaging: 消息为空")

	// ErrEmptyTopic 消息主题为空.
	ErrEmptyTopic = errors.New("messaging: 消息主题为空")

	// ErrNoTopics 未指定消费主题.
	ErrNoTopics = errors.New("messaging: 未指定消费主题")

	// ErrNilHandler 消息处理器为空.
	ErrNilHandler = errors.New("messaging: 消息处理器为空")

	// ErrNoActiveSession 没有活跃的消费者会话.
	ErrNoActiveSession = errors.New("messaging: 没有活跃的消费者会话")

	// ErrUnsupportedType 不支持的消息队列类型.
	ErrUnsupportedType = errors.New("messaging: 不支持的消息队列类型")

	// ErrCreateProducer 创建生产者失败.
	ErrCreateProducer = errors.New("messaging: 创建生产者失败")

	// ErrCreateConsumer 创建消费者失败.
	ErrCreateConsumer = errors.New("messaging: 创建消费者失败")

	// ErrSendMessage 消息发送失败.
	ErrSendMessage = errors.New("messaging: 消息发送失败")

	// ErrNoBrokers 未配置服务器地址.
	ErrNoBrokers = errors.New("messaging: 未配置服务器地址")

	// ErrCreateClient 创建客户端失败.
	ErrCreateClient = errors.New("messaging: 创建客户端失败")

	// ErrClientClosed 客户端已关闭.
	ErrClientClosed = errors.New("messaging: 客户端已关闭")

	// ErrNoBrokersAvailable 没有可用的服务器.
	ErrNoBrokersAvailable = errors.New("messaging: 没有可用的服务器")

	// ErrHealthCheck 健康检查失败.
	ErrHealthCheck = errors.New("messaging: 健康检查失败")

	// ErrBatchSend 批量发送失败.
	ErrBatchSend = errors.New("messaging: 批量发送失败")
)
