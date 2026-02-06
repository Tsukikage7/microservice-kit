package app

import "context"

// Hook 生命周期钩子函数.
type Hook func(ctx context.Context) error

// Hooks 生命周期钩子集合.
type Hooks struct {
	BeforeStart []Hook
	AfterStart  []Hook
	BeforeStop  []Hook
	AfterStop   []Hook
}

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

func (h *Hooks) runBeforeStart(ctx context.Context) error {
	if h == nil {
		return nil
	}
	return h.run(ctx, h.BeforeStart)
}

func (h *Hooks) runAfterStart(ctx context.Context) error {
	if h == nil {
		return nil
	}
	return h.run(ctx, h.AfterStart)
}

func (h *Hooks) runBeforeStop(ctx context.Context) error {
	if h == nil {
		return nil
	}
	return h.run(ctx, h.BeforeStop)
}

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
	return &HooksBuilder{hooks: &Hooks{}}
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
