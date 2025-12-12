package deviceinfo

// Option 配置选项函数.
type Option func(*options)

type options struct {
	enableUAFallback bool // 是否启用 User-Agent 回退
	setAcceptCH      bool // 是否设置 Accept-CH 响应头
}

func defaultOptions() *options {
	return &options{
		enableUAFallback: true,
		setAcceptCH:      false,
	}
}

// WithUAFallback 设置是否启用 User-Agent 回退.
// 当 Client Hints 不可用时，回退到解析 User-Agent.
// 默认启用.
func WithUAFallback(enable bool) Option {
	return func(o *options) {
		o.enableUAFallback = enable
	}
}

// WithAcceptCH 设置是否在响应中添加 Accept-CH 头.
// 启用后，中间件会自动添加 Accept-CH 头请求 Client Hints.
// 默认禁用.
func WithAcceptCH(enable bool) Option {
	return func(o *options) {
		o.setAcceptCH = enable
	}
}
