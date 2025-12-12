package treemap

import (
	"cmp"
	"time"
)

// OrderedCompare 用于 cmp.Ordered 类型的比较器.
// 支持 int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
// float32, float64, string, uintptr 等类型.
func OrderedCompare[T cmp.Ordered](a, b T) int {
	return cmp.Compare(a, b)
}

// ReverseCompare 用于 cmp.Ordered 类型的逆序比较器.
func ReverseCompare[T cmp.Ordered](a, b T) int {
	return cmp.Compare(b, a)
}

// TimeCompare 时间比较器.
func TimeCompare(a, b time.Time) int {
	switch {
	case a.Before(b):
		return -1
	case a.After(b):
		return 1
	default:
		return 0
	}
}

// ReverseTimeCompare 时间逆序比较器.
func ReverseTimeCompare(a, b time.Time) int {
	return TimeCompare(b, a)
}

// Reverse 返回逆序比较器.
func Reverse[K any](cmp Comparator[K]) Comparator[K] {
	return func(a, b K) int {
		return cmp(b, a)
	}
}
