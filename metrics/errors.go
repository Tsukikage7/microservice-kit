package metrics

import "errors"

// 预定义错误.
var (
	// ErrNilConfig 指标配置为空.
	ErrNilConfig = errors.New("指标配置为空")
	// ErrRegisterMetric 注册指标失败.
	ErrRegisterMetric = errors.New("注册指标失败")
	// ErrEmptyNamespace 命名空间为空.
	ErrEmptyNamespace = errors.New("命名空间为空")
)
