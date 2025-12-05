// Package scheduler 提供任务调度功能.
//
// 特性：
//   - 支持秒级 Cron 表达式
//   - 单例模式：防止同一任务重叠执行
//   - 分布式锁：多实例部署时保证只有一个实例执行（复用 cache 包）
//   - 任务状态跟踪和统计
//   - Hook 机制：BeforeJob/AfterJob/OnError/OnSkip
//   - 失败重试
//   - 优雅关闭
//
// 示例：
//
//	// 创建调度器（使用 cache 包实现分布式锁）
//	s := scheduler.MustNew(
//	    scheduler.WithLogger(log),
//	    scheduler.WithCache(redisCache),  // 复用 cache.Cache 接口
//	)
//
//	// 添加任务
//	s.Add(scheduler.NewJob("sync-data").
//	    Schedule("0 */5 * * * *").
//	    Handler(syncHandler).
//	    Singleton().      // 本地幂等
//	    Distributed().    // 分布式幂等
//	    MustBuild(),
//	)
//
//	// 启动
//	s.Start()
//	defer s.Stop()
package scheduler

import "context"

// Scheduler 调度器接口.
type Scheduler interface {
	// Add 添加任务.
	Add(job *Job) error

	// Remove 移除任务.
	Remove(name string) error

	// Get 获取任务.
	Get(name string) (*Job, bool)

	// List 列出所有任务.
	List() []*Job

	// Start 启动调度器.
	Start() error

	// Stop 停止调度器.
	Stop()

	// Shutdown 优雅关闭.
	Shutdown(ctx context.Context) error

	// Running 检查是否运行中.
	Running() bool

	// Trigger 立即触发任务执行（不影响正常调度）.
	Trigger(name string) error
}

// New 创建调度器.
func New(opts ...Option) (Scheduler, error) {
	return newCronScheduler(opts...)
}

// MustNew 创建调度器，失败时 panic.
func MustNew(opts ...Option) Scheduler {
	s, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return s
}
