package idempotency

import "errors"

// 预定义错误.
var (
	// ErrDuplicateRequest 重复请求.
	ErrDuplicateRequest = errors.New("idempotency: 重复请求")

	// ErrRequestInProgress 请求正在处理中.
	ErrRequestInProgress = errors.New("idempotency: 请求正在处理中")

	// ErrMissingKey 缺少幂等键.
	ErrMissingKey = errors.New("idempotency: 缺少幂等键")

	// ErrInvalidKey 幂等键无效.
	ErrInvalidKey = errors.New("idempotency: 幂等键无效")

	// ErrStoreFailure 存储操作失败.
	ErrStoreFailure = errors.New("idempotency: 存储操作失败")

	// ErrNilStore 存储为空.
	ErrNilStore = errors.New("idempotency: 存储不能为空")

	// ErrNilLogger 日志记录器为空.
	ErrNilLogger = errors.New("idempotency: 日志记录器不能为空")
)
