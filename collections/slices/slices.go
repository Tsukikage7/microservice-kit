// Package slices 提供切片操作的工具函数.
package slices

// Filter 过滤切片，返回满足条件的元素.
//
// 示例:
//
//	nums := []int{1, 2, 3, 4, 5}
//	evens := slices.Filter(nums, func(n int) bool { return n%2 == 0 })
//	// evens: [2, 4]
func Filter[T any](slice []T, fn func(T) bool) []T {
	result := make([]T, 0)
	for _, item := range slice {
		if fn(item) {
			result = append(result, item)
		}
	}
	return result
}

// Map 对切片中的每个元素应用转换函数.
//
// 示例:
//
//	nums := []int{1, 2, 3}
//	strs := slices.Map(nums, strconv.Itoa)
//	// strs: ["1", "2", "3"]
func Map[T, R any](slice []T, fn func(T) R) []R {
	result := make([]R, len(slice))
	for i, item := range slice {
		result[i] = fn(item)
	}
	return result
}

// Reduce 将切片归约为单个值.
//
// 示例:
//
//	nums := []int{1, 2, 3, 4}
//	sum := slices.Reduce(nums, 0, func(acc, n int) int { return acc + n })
//	// sum: 10
func Reduce[T, R any](slice []T, initial R, fn func(R, T) R) R {
	result := initial
	for _, item := range slice {
		result = fn(result, item)
	}
	return result
}

// Unique 返回去重后的切片（保持原顺序）.
//
// 示例:
//
//	nums := []int{1, 2, 2, 3, 1}
//	unique := slices.Unique(nums)
//	// unique: [1, 2, 3]
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{}, len(slice))
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// UniqueBy 根据键函数去重（保持原顺序）.
//
// 示例:
//
//	users := []User{{ID: 1, Name: "a"}, {ID: 1, Name: "b"}}
//	unique := slices.UniqueBy(users, func(u User) int { return u.ID })
//	// unique: [{ID: 1, Name: "a"}]
func UniqueBy[T any, K comparable](slice []T, keyFn func(T) K) []T {
	seen := make(map[K]struct{}, len(slice))
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		key := keyFn(item)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// GroupBy 根据键函数对切片进行分组.
//
// 示例:
//
//	nums := []int{1, 2, 3, 4, 5}
//	groups := slices.GroupBy(nums, func(n int) string {
//	    if n%2 == 0 { return "even" }
//	    return "odd"
//	})
//	// groups: {"odd": [1, 3, 5], "even": [2, 4]}
func GroupBy[T any, K comparable](slice []T, keyFn func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, item := range slice {
		key := keyFn(item)
		result[key] = append(result[key], item)
	}
	return result
}

// Chunk 将切片分割为指定大小的块.
//
// 示例:
//
//	nums := []int{1, 2, 3, 4, 5}
//	chunks := slices.Chunk(nums, 2)
//	// chunks: [[1, 2], [3, 4], [5]]
func Chunk[T any](slice []T, size int) [][]T {
	if size <= 0 {
		return nil
	}

	n := len(slice)
	chunks := make([][]T, 0, (n+size-1)/size)

	for i := 0; i < n; i += size {
		end := i + size
		if end > n {
			end = n
		}
		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

// Partition 将切片分为满足条件和不满足条件的两部分.
//
// 示例:
//
//	nums := []int{1, 2, 3, 4, 5}
//	evens, odds := slices.Partition(nums, func(n int) bool { return n%2 == 0 })
//	// evens: [2, 4], odds: [1, 3, 5]
func Partition[T any](slice []T, fn func(T) bool) (matched, unmatched []T) {
	matched = make([]T, 0)
	unmatched = make([]T, 0)
	for _, item := range slice {
		if fn(item) {
			matched = append(matched, item)
		} else {
			unmatched = append(unmatched, item)
		}
	}
	return
}

// Find 查找第一个满足条件的元素.
//
// 示例:
//
//	nums := []int{1, 2, 3, 4}
//	val, ok := slices.Find(nums, func(n int) bool { return n > 2 })
//	// val: 3, ok: true
func Find[T any](slice []T, fn func(T) bool) (T, bool) {
	for _, item := range slice {
		if fn(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

// FindIndex 查找第一个满足条件的元素索引.
//
// 示例:
//
//	nums := []int{1, 2, 3, 4}
//	idx := slices.FindIndex(nums, func(n int) bool { return n > 2 })
//	// idx: 2
func FindIndex[T any](slice []T, fn func(T) bool) int {
	for i, item := range slice {
		if fn(item) {
			return i
		}
	}
	return -1
}

// Any 判断是否有任意元素满足条件.
//
// 示例:
//
//	nums := []int{1, 2, 3}
//	hasEven := slices.Any(nums, func(n int) bool { return n%2 == 0 })
//	// hasEven: true
func Any[T any](slice []T, fn func(T) bool) bool {
	for _, item := range slice {
		if fn(item) {
			return true
		}
	}
	return false
}

// All 判断是否所有元素都满足条件.
//
// 示例:
//
//	nums := []int{2, 4, 6}
//	allEven := slices.All(nums, func(n int) bool { return n%2 == 0 })
//	// allEven: true
func All[T any](slice []T, fn func(T) bool) bool {
	for _, item := range slice {
		if !fn(item) {
			return false
		}
	}
	return true
}

// None 判断是否没有元素满足条件.
//
// 示例:
//
//	nums := []int{1, 3, 5}
//	noEven := slices.None(nums, func(n int) bool { return n%2 == 0 })
//	// noEven: true
func None[T any](slice []T, fn func(T) bool) bool {
	return !Any(slice, fn)
}

// Count 统计满足条件的元素数量.
//
// 示例:
//
//	nums := []int{1, 2, 3, 4}
//	count := slices.Count(nums, func(n int) bool { return n%2 == 0 })
//	// count: 2
func Count[T any](slice []T, fn func(T) bool) int {
	count := 0
	for _, item := range slice {
		if fn(item) {
			count++
		}
	}
	return count
}

// Flatten 将二维切片扁平化为一维切片.
//
// 示例:
//
//	nested := [][]int{{1, 2}, {3, 4}, {5}}
//	flat := slices.Flatten(nested)
//	// flat: [1, 2, 3, 4, 5]
func Flatten[T any](slices [][]T) []T {
	total := 0
	for _, s := range slices {
		total += len(s)
	}

	result := make([]T, 0, total)
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// Zip 将两个切片合并为键值对切片.
//
// 示例:
//
//	keys := []string{"a", "b", "c"}
//	vals := []int{1, 2, 3}
//	pairs := slices.Zip(keys, vals)
//	// pairs: [{"a", 1}, {"b", 2}, {"c", 3}]
func Zip[K, V any](keys []K, values []V) []Pair[K, V] {
	n := len(keys)
	if len(values) < n {
		n = len(values)
	}

	result := make([]Pair[K, V], n)
	for i := 0; i < n; i++ {
		result[i] = Pair[K, V]{First: keys[i], Second: values[i]}
	}
	return result
}

// Pair 键值对.
type Pair[K, V any] struct {
	First  K
	Second V
}

// ToMap 将键值对切片转换为 map.
//
// 示例:
//
//	pairs := []slices.Pair[string, int]{{"a", 1}, {"b", 2}}
//	m := slices.ToMap(pairs)
//	// m: {"a": 1, "b": 2}
func ToMap[K comparable, V any](pairs []Pair[K, V]) map[K]V {
	result := make(map[K]V, len(pairs))
	for _, p := range pairs {
		result[p.First] = p.Second
	}
	return result
}

// KeyBy 将切片转换为 map，使用键函数生成键.
//
// 示例:
//
//	users := []User{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}
//	m := slices.KeyBy(users, func(u User) int { return u.ID })
//	// m: {1: {ID: 1, Name: "a"}, 2: {ID: 2, Name: "b"}}
func KeyBy[T any, K comparable](slice []T, keyFn func(T) K) map[K]T {
	result := make(map[K]T, len(slice))
	for _, item := range slice {
		result[keyFn(item)] = item
	}
	return result
}

// Compact 移除切片中的零值元素.
//
// 示例:
//
//	strs := []string{"a", "", "b", "", "c"}
//	compact := slices.Compact(strs)
//	// compact: ["a", "b", "c"]
func Compact[T comparable](slice []T) []T {
	var zero T
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if item != zero {
			result = append(result, item)
		}
	}
	return result
}

// First 返回第一个元素.
func First[T any](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	return slice[0], true
}

// Last 返回最后一个元素.
func Last[T any](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	return slice[len(slice)-1], true
}

// Take 返回前 n 个元素.
func Take[T any](slice []T, n int) []T {
	if n <= 0 {
		return nil
	}
	if n > len(slice) {
		n = len(slice)
	}
	return slice[:n]
}

// Skip 跳过前 n 个元素.
func Skip[T any](slice []T, n int) []T {
	if n <= 0 {
		return slice
	}
	if n >= len(slice) {
		return nil
	}
	return slice[n:]
}

// TakeWhile 返回满足条件的前缀元素.
func TakeWhile[T any](slice []T, fn func(T) bool) []T {
	for i, item := range slice {
		if !fn(item) {
			return slice[:i]
		}
	}
	return slice
}

// SkipWhile 跳过满足条件的前缀元素.
func SkipWhile[T any](slice []T, fn func(T) bool) []T {
	for i, item := range slice {
		if !fn(item) {
			return slice[i:]
		}
	}
	return nil
}
