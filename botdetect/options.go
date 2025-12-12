package botdetect

// Option 配置选项函数.
type Option func(*options)

type options struct {
	threshold float64 // 判定为机器人的阈值
}

func defaultOptions() *options {
	return &options{
		threshold: 0.5,
	}
}

// WithThreshold 设置机器人判定阈值.
// 值越低越容易被判定为机器人，默认 0.5.
func WithThreshold(threshold float64) Option {
	return func(o *options) {
		if threshold > 0 && threshold <= 1 {
			o.threshold = threshold
		}
	}
}
