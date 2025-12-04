package messaging

import "time"

// Message 消息结构.
//
// 用于生产者发送和消费者接收消息.
// Key 和 Value 均为 []byte 类型，序列化由调用方控制.
//
// 发送示例:
//
//	msg := &messaging.Message{
//	    Topic: "orders",
//	    Key:   []byte("order-123"),
//	    Value: []byte(`{"id":"123","amount":100}`),
//	    Headers: map[string]string{
//	        "trace-id": "abc-123",
//	    },
//	}
//
// 接收示例:
//
//	handler := func(msg *messaging.Message) error {
//	    var order Order
//	    if err := json.Unmarshal(msg.Value, &order); err != nil {
//	        return err
//	    }
//	    // 处理订单...
//	    return nil
//	}
type Message struct {
	// Topic 消息主题，必填.
	Topic string

	// Key 消息键，用于分区路由.
	// 相同 Key 的消息会路由到同一分区，保证顺序性.
	Key []byte

	// Value 消息内容.
	Value []byte

	// Headers 消息头，用于传递元数据.
	Headers map[string]string

	// Partition 分区号.
	// 发送后由服务端返回填充.
	Partition int32

	// Offset 消息偏移量.
	// 发送后由服务端返回填充.
	Offset int64

	// Timestamp 消息时间戳.
	Timestamp time.Time
}
