// Package client 提供 HTTP 客户端工具.
package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Tsukikage7/microservice-kit/discovery"
	"github.com/Tsukikage7/microservice-kit/logger"
)

// Client HTTP 客户端封装.
type Client struct {
	httpClient *http.Client
	opts       *options
	baseURL    string
}

// New 创建 HTTP 客户端，必需设置 serviceName、discovery、logger，否则会 panic.
func New(opts ...Option) (*Client, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	// 验证必需参数
	if o.serviceName == "" {
		panic("http client: 必须设置 serviceName")
	}
	if o.discovery == nil {
		panic("http client: 必须设置 discovery")
	}
	if o.logger == nil {
		panic("http client: 必须设置 logger")
	}

	// 服务发现
	addrs, err := o.discovery.Discover(context.Background(), o.serviceName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDiscoveryFailed, err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrServiceNotFound, o.serviceName)
	}
	target := addrs[0]

	// 构建 HTTP 客户端
	transport := o.transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	httpClient := &http.Client{
		Timeout:   o.timeout,
		Transport: transport,
	}

	baseURL := fmt.Sprintf("%s://%s", o.scheme, target)
	o.logger.With(
		logger.String("name", o.name),
		logger.String("service", o.serviceName),
		logger.String("baseURL", baseURL),
	).Info("[HTTP] 客户端初始化成功")

	return &Client{
		httpClient: httpClient,
		opts:       o,
		baseURL:    baseURL,
	}, nil
}

// HTTPClient 返回底层 http.Client.
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// BaseURL 返回基础 URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Get 发送 GET 请求.
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	return c.Do(ctx, http.MethodGet, path, nil)
}

// Post 发送 POST 请求.
func (c *Client) Post(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	return c.Do(ctx, http.MethodPost, path, body)
}

// Put 发送 PUT 请求.
func (c *Client) Put(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	return c.Do(ctx, http.MethodPut, path, body)
}

// Delete 发送 DELETE 请求.
func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	return c.Do(ctx, http.MethodDelete, path, nil)
}

// Do 执行 HTTP 请求.
func (c *Client) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}

	// 添加默认 headers
	for key, value := range c.opts.headers {
		req.Header.Set(key, value)
	}

	return c.httpClient.Do(req)
}

// Option 配置选项函数.
type Option func(*options)

// options 客户端配置.
type options struct {
	name        string
	serviceName string
	scheme      string
	discovery   discovery.Discovery
	logger      logger.Logger
	timeout     time.Duration
	headers     map[string]string
	transport   http.RoundTripper
}

// defaultOptions 返回默认配置.
func defaultOptions() *options {
	return &options{
		name:    "HTTP-Client",
		scheme:  "http",
		timeout: 30 * time.Second,
		headers: make(map[string]string),
	}
}

// WithName 设置客户端名称（用于日志）.
func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// WithServiceName 设置目标服务名称（必需）.
func WithServiceName(name string) Option {
	return func(o *options) {
		o.serviceName = name
	}
}

// WithScheme 设置 URL scheme，默认 http.
func WithScheme(scheme string) Option {
	return func(o *options) {
		o.scheme = scheme
	}
}

// WithDiscovery 设置服务发现实例（必需）.
func WithDiscovery(d discovery.Discovery) Option {
	return func(o *options) {
		o.discovery = d
	}
}

// WithLogger 设置日志实例（必需）.
func WithLogger(l logger.Logger) Option {
	return func(o *options) {
		o.logger = l
	}
}

// WithTimeout 设置请求超时.
func WithTimeout(d time.Duration) Option {
	return func(o *options) {
		o.timeout = d
	}
}

// WithHeader 添加默认请求头.
func WithHeader(key, value string) Option {
	return func(o *options) {
		o.headers[key] = value
	}
}

// WithHeaders 设置多个默认请求头.
func WithHeaders(headers map[string]string) Option {
	return func(o *options) {
		for k, v := range headers {
			o.headers[k] = v
		}
	}
}

// WithTransport 设置自定义 Transport.
func WithTransport(transport http.RoundTripper) Option {
	return func(o *options) {
		o.transport = transport
	}
}
