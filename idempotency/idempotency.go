// Package idempotency 提供请求幂等性控制.
//
// 幂等性保证同一请求多次执行的效果与执行一次相同，
// 适用于支付、订单等关键业务场景。
//
// 工作原理:
//  1. 客户端在请求头中携带幂等键（Idempotency-Key）
//  2. 服务端检查该键是否已存在
//  3. 如果存在，返回之前的结果
//  4. 如果不存在，执行请求并保存结果
//
// 基本用法:
//
//	store := idempotency.NewRedisStore(redisClient)
//	handler = idempotency.HTTPMiddleware(store)(handler)
//
// 自定义键提取:
//
//	handler = idempotency.HTTPMiddleware(store,
//	    idempotency.WithKeyExtractor(func(r *http.Request) string {
//	        return r.Header.Get("X-Request-ID")
//	    }),
//	)(handler)
package idempotency

import (
	"context"
	"encoding/json"
	"time"
)

// Result 幂等请求的结果.
type Result struct {
	// StatusCode HTTP 状态码（HTTP 请求）或 gRPC 状态码
	StatusCode int `json:"status_code"`

	// Headers 响应头（HTTP 请求）
	Headers map[string]string `json:"headers,omitempty"`

	// Body 响应体
	Body []byte `json:"body,omitempty"`

	// Error 错误信息
	Error string `json:"error,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
}

// Encode 将 Result 编码为字节数组.
func (r *Result) Encode() ([]byte, error) {
	return json.Marshal(r)
}

// DecodeResult 从字节数组解码 Result.
func DecodeResult(data []byte) (*Result, error) {
	var r Result
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// Store 幂等性存储接口.
type Store interface {
	// Get 获取幂等键对应的结果.
	// 如果键不存在，返回 nil, nil.
	Get(ctx context.Context, key string) (*Result, error)

	// Set 设置幂等键和结果.
	// ttl 为过期时间，过期后键会被自动删除.
	Set(ctx context.Context, key string, result *Result, ttl time.Duration) error

	// SetNX 仅在键不存在时设置.
	// 返回 true 表示设置成功（键不存在），false 表示键已存在.
	SetNX(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Delete 删除幂等键.
	Delete(ctx context.Context, key string) error
}

// KeyExtractor 从请求中提取幂等键的函数.
//
// 对于 HTTP 请求，参数类型为 *http.Request.
// 对于 gRPC 请求，参数类型为 context.Context.
// 对于 Endpoint 请求，参数为请求对象.
type KeyExtractor func(ctx any) string

// DefaultHTTPKeyHeader 默认的 HTTP 幂等键请求头.
const DefaultHTTPKeyHeader = "Idempotency-Key"

// DefaultGRPCKeyMetadata 默认的 gRPC 幂等键元数据.
const DefaultGRPCKeyMetadata = "x-idempotency-key"

// DefaultTTL 默认的幂等键过期时间.
const DefaultTTL = 24 * time.Hour
