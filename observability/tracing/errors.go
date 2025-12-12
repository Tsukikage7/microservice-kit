package tracing

import "errors"

// 预定义错误常量.
var (
	// ErrNilConfig 链路追踪配置为空.
	ErrNilConfig = errors.New("tracing: 配置为空")

	// ErrEmptyServiceName 服务名称为空.
	ErrEmptyServiceName = errors.New("tracing: 服务名称为空")

	// ErrEmptyEndpoint OTLP端点为空.
	ErrEmptyEndpoint = errors.New("tracing: OTLP端点为空")

	// ErrCreateExporter 创建OTLP导出器失败.
	ErrCreateExporter = errors.New("tracing: 创建OTLP导出器失败")

	// ErrCreateResource 创建资源失败.
	ErrCreateResource = errors.New("tracing: 创建资源失败")
)
