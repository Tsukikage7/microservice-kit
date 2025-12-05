package scheduler

import "errors"

// 预定义错误.
var (
	// ErrJobNameEmpty 任务名称为空.
	ErrJobNameEmpty = errors.New("scheduler: job name is required")

	// ErrScheduleEmpty 调度表达式为空.
	ErrScheduleEmpty = errors.New("scheduler: schedule expression is required")

	// ErrHandlerNil 任务处理函数为空.
	ErrHandlerNil = errors.New("scheduler: job handler is required")

	// ErrScheduleInvalid 无效的调度表达式.
	ErrScheduleInvalid = errors.New("scheduler: invalid schedule expression")

	// ErrSchedulerClosed 调度器已关闭.
	ErrSchedulerClosed = errors.New("scheduler: scheduler is closed")

	// ErrJobNotFound 任务未找到.
	ErrJobNotFound = errors.New("scheduler: job not found")

	// ErrJobExists 任务已存在.
	ErrJobExists = errors.New("scheduler: job already exists")

	// ErrJobRunning 任务正在执行中.
	ErrJobRunning = errors.New("scheduler: job is already running")

	// ErrLockAcquireFailed 获取锁失败.
	ErrLockAcquireFailed = errors.New("scheduler: failed to acquire lock")

	// ErrJobSkipped 任务被跳过（上一次执行未完成）.
	ErrJobSkipped = errors.New("scheduler: job skipped due to previous execution still running")
)
