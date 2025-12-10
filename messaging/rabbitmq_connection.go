package messaging

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// rabbitMQConnection RabbitMQ 连接管理器.
type rabbitMQConnection struct {
	url            string
	conn           *amqp.Connection
	mu             sync.RWMutex
	closed         atomic.Bool
	reconnectDelay time.Duration
	maxRetries     int
	logger         logger.Logger

	notifyClose chan *amqp.Error
	reconnectCh chan struct{}
}

// rabbitMQConnectionOption 连接配置选项.
type rabbitMQConnectionOption func(*rabbitMQConnection)

func withRabbitMQReconnectDelay(delay time.Duration) rabbitMQConnectionOption {
	return func(c *rabbitMQConnection) {
		c.reconnectDelay = delay
	}
}

func withRabbitMQMaxRetries(retries int) rabbitMQConnectionOption {
	return func(c *rabbitMQConnection) {
		c.maxRetries = retries
	}
}

func withRabbitMQConnectionLogger(log logger.Logger) rabbitMQConnectionOption {
	return func(c *rabbitMQConnection) {
		c.logger = log
	}
}

func newRabbitMQConnection(url string, opts ...rabbitMQConnectionOption) (*rabbitMQConnection, error) {
	c := &rabbitMQConnection{
		url:            url,
		reconnectDelay: 5 * time.Second,
		maxRetries:     -1,
		reconnectCh:    make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	if err := c.connect(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreateClient, err)
	}

	go c.handleReconnect()

	return c, nil
}

func (c *rabbitMQConnection) connect() error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.notifyClose = make(chan *amqp.Error, 1)
	c.conn.NotifyClose(c.notifyClose)
	c.mu.Unlock()

	c.log("RabbitMQ 连接已建立")
	return nil
}

func (c *rabbitMQConnection) handleReconnect() {
	for {
		select {
		case err, ok := <-c.notifyClose:
			if !ok || c.closed.Load() {
				return
			}

			c.log("RabbitMQ 连接断开: %v, 开始重连...", err)

			retries := 0
			for {
				if c.closed.Load() {
					return
				}

				if c.maxRetries > 0 && retries >= c.maxRetries {
					c.log("RabbitMQ 重连失败，已达最大重试次数")
					return
				}

				time.Sleep(c.reconnectDelay)

				if err := c.connect(); err != nil {
					retries++
					c.log("RabbitMQ 重连失败 (%d): %v", retries, err)
					continue
				}

				select {
				case c.reconnectCh <- struct{}{}:
				default:
				}
				break
			}
		}
	}
}

func (c *rabbitMQConnection) Channel() (*amqp.Channel, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed.Load() {
		return nil, ErrClientClosed
	}

	if c.conn == nil {
		return nil, ErrNoBrokersAvailable
	}

	return c.conn.Channel()
}

func (c *rabbitMQConnection) ReconnectNotify() <-chan struct{} {
	return c.reconnectCh
}

func (c *rabbitMQConnection) IsClosed() bool {
	return c.closed.Load()
}

func (c *rabbitMQConnection) Close() error {
	if c.closed.Swap(true) {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *rabbitMQConnection) log(format string, args ...any) {
	if c.logger != nil {
		c.logger.Info(fmt.Sprintf(format, args...))
	}
}
