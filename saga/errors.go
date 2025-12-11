package saga

import "errors"

// 预定义错误.
var (
	// ErrSagaFailed Saga 执行失败.
	ErrSagaFailed = errors.New("saga: 执行失败")

	// ErrCompensationFailed 补偿执行失败.
	ErrCompensationFailed = errors.New("saga: 补偿执行失败")

	// ErrStepFailed 步骤执行失败.
	ErrStepFailed = errors.New("saga: 步骤执行失败")

	// ErrNoSteps 没有定义步骤.
	ErrNoSteps = errors.New("saga: 没有定义步骤")

	// ErrInvalidStep 步骤无效.
	ErrInvalidStep = errors.New("saga: 步骤无效")

	// ErrSagaNotFound Saga 不存在.
	ErrSagaNotFound = errors.New("saga: Saga 不存在")

	// ErrNilLogger 日志记录器为空.
	ErrNilLogger = errors.New("saga: 日志记录器不能为空")
)
