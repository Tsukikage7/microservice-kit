package pagination

const (
	// DefaultPage 默认页码.
	DefaultPage = 1
	// DefaultPageSize 默认每页数量.
	DefaultPageSize = 20
	// MaxPageSize 最大每页数量.
	MaxPageSize = 100
)

// Pagination 分页参数.
type Pagination struct {
	Page     int32 // 页码，从1开始
	PageSize int32 // 每页数量
}

// New 创建分页参数，自动应用默认值和边界校验.
func New(page, pageSize int32) Pagination {
	p := Pagination{
		Page:     page,
		PageSize: pageSize,
	}
	p.normalize()
	return p
}

// normalize 标准化分页参数.
func (p *Pagination) normalize() {
	if p.Page <= 0 {
		p.Page = DefaultPage
	}
	if p.PageSize <= 0 {
		p.PageSize = DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
}

// Offset 计算偏移量.
func (p Pagination) Offset() int {
	return int((p.Page - 1) * p.PageSize)
}

// Limit 返回每页数量.
func (p Pagination) Limit() int {
	return int(p.PageSize)
}

// Result 分页查询结果.
type Result[T any] struct {
	Items    []T   // 数据列表
	Total    int32 // 总数
	Page     int32 // 当前页码
	PageSize int32 // 每页数量
}

// NewResult 创建分页结果.
func NewResult[T any](items []T, total int32, p Pagination) Result[T] {
	return Result[T]{
		Items:    items,
		Total:    total,
		Page:     p.Page,
		PageSize: p.PageSize,
	}
}

// TotalPages 计算总页数.
func (r Result[T]) TotalPages() int32 {
	if r.Total == 0 || r.PageSize == 0 {
		return 0
	}
	pages := r.Total / r.PageSize
	if r.Total%r.PageSize > 0 {
		pages++
	}
	return pages
}

// HasNext 是否有下一页.
func (r Result[T]) HasNext() bool {
	return r.Page < r.TotalPages()
}

// HasPrev 是否有上一页.
func (r Result[T]) HasPrev() bool {
	return r.Page > 1
}
