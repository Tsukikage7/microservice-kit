package transport

import "context"

// Hook 生命周期钩子函数.
type Hook func(ctx context.Context) error

// Hooks 生命周期钩子集合.
type Hooks struct {
	// BeforeStart 启动前钩子（按添加顺序执行）.
	BeforeStart []Hook

	// AfterStart 启动后钩子（按添加顺序执行）.
	AfterStart []Hook

	// BeforeStop 停止前钩子（按添加顺序执行）.
	BeforeStop []Hook

	// AfterStop 停止后钩子（按添加顺序执行）.
	AfterStop []Hook
}

// run 执行一组钩子.
func (h *Hooks) run(ctx context.Context, hooks []Hook) error {
	if h == nil {
		return nil
	}
	for _, hook := range hooks {
		if err := hook(ctx); err != nil {
			return err
		}
	}
	return nil
}

// runBeforeStart 执行启动前钩子.
func (h *Hooks) runBeforeStart(ctx context.Context) error {
	if h == nil {
		return nil
	}
	return h.run(ctx, h.BeforeStart)
}

// runAfterStart 执行启动后钩子.
func (h *Hooks) runAfterStart(ctx context.Context) error {
	if h == nil {
		return nil
	}
	return h.run(ctx, h.AfterStart)
}

// runBeforeStop 执行停止前钩子.
func (h *Hooks) runBeforeStop(ctx context.Context) error {
	if h == nil {
		return nil
	}
	return h.run(ctx, h.BeforeStop)
}

// runAfterStop 执行停止后钩子.
func (h *Hooks) runAfterStop(ctx context.Context) error {
	if h == nil {
		return nil
	}
	return h.run(ctx, h.AfterStop)
}

// HooksBuilder 钩子构建器.
type HooksBuilder struct {
	hooks *Hooks
}

// NewHooks 创建钩子构建器.
func NewHooks() *HooksBuilder {
	return &HooksBuilder{
		hooks: &Hooks{},
	}
}

// BeforeStart 添加启动前钩子.
func (b *HooksBuilder) BeforeStart(hook Hook) *HooksBuilder {
	b.hooks.BeforeStart = append(b.hooks.BeforeStart, hook)
	return b
}

// AfterStart 添加启动后钩子.
func (b *HooksBuilder) AfterStart(hook Hook) *HooksBuilder {
	b.hooks.AfterStart = append(b.hooks.AfterStart, hook)
	return b
}

// BeforeStop 添加停止前钩子.
func (b *HooksBuilder) BeforeStop(hook Hook) *HooksBuilder {
	b.hooks.BeforeStop = append(b.hooks.BeforeStop, hook)
	return b
}

// AfterStop 添加停止后钩子.
func (b *HooksBuilder) AfterStop(hook Hook) *HooksBuilder {
	b.hooks.AfterStop = append(b.hooks.AfterStop, hook)
	return b
}

// Build 构建钩子.
func (b *HooksBuilder) Build() *Hooks {
	return b.hooks
}
