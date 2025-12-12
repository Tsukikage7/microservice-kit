package pagination

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name             string
		page             int32
		pageSize         int32
		expectedPage     int32
		expectedPageSize int32
	}{
		{
			name:             "正常参数",
			page:             2,
			pageSize:         10,
			expectedPage:     2,
			expectedPageSize: 10,
		},
		{
			name:             "页码为0时使用默认值",
			page:             0,
			pageSize:         10,
			expectedPage:     DefaultPage,
			expectedPageSize: 10,
		},
		{
			name:             "页码为负数时使用默认值",
			page:             -1,
			pageSize:         10,
			expectedPage:     DefaultPage,
			expectedPageSize: 10,
		},
		{
			name:             "每页数量为0时使用默认值",
			page:             1,
			pageSize:         0,
			expectedPage:     1,
			expectedPageSize: DefaultPageSize,
		},
		{
			name:             "每页数量超过最大值时限制",
			page:             1,
			pageSize:         200,
			expectedPage:     1,
			expectedPageSize: MaxPageSize,
		},
		{
			name:             "所有参数都无效时使用默认值",
			page:             -1,
			pageSize:         -1,
			expectedPage:     DefaultPage,
			expectedPageSize: DefaultPageSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.page, tt.pageSize)
			if p.Page != tt.expectedPage {
				t.Errorf("Page = %d, want %d", p.Page, tt.expectedPage)
			}
			if p.PageSize != tt.expectedPageSize {
				t.Errorf("PageSize = %d, want %d", p.PageSize, tt.expectedPageSize)
			}
		})
	}
}

func TestPagination_Offset(t *testing.T) {
	tests := []struct {
		name           string
		page           int32
		pageSize       int32
		expectedOffset int
	}{
		{"第1页", 1, 20, 0},
		{"第2页", 2, 20, 20},
		{"第3页每页10条", 3, 10, 20},
		{"第5页每页50条", 5, 50, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.page, tt.pageSize)
			if offset := p.Offset(); offset != tt.expectedOffset {
				t.Errorf("Offset() = %d, want %d", offset, tt.expectedOffset)
			}
		})
	}
}

func TestPagination_Limit(t *testing.T) {
	p := New(1, 25)
	if limit := p.Limit(); limit != 25 {
		t.Errorf("Limit() = %d, want 25", limit)
	}
}

func TestResult(t *testing.T) {
	items := []string{"a", "b", "c"}
	p := New(2, 10)
	result := NewResult(items, 35, p)

	if result.Total != 35 {
		t.Errorf("Total = %d, want 35", result.Total)
	}
	if result.Page != 2 {
		t.Errorf("Page = %d, want 2", result.Page)
	}
	if result.PageSize != 10 {
		t.Errorf("PageSize = %d, want 10", result.PageSize)
	}
	if len(result.Items) != 3 {
		t.Errorf("Items count = %d, want 3", len(result.Items))
	}
}

func TestResult_TotalPages(t *testing.T) {
	tests := []struct {
		name          string
		total         int32
		pageSize      int32
		expectedPages int32
	}{
		{"整除", 100, 10, 10},
		{"有余数", 101, 10, 11},
		{"总数为0", 0, 10, 0},
		{"总数小于每页数量", 5, 10, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(1, tt.pageSize)
			result := NewResult([]string{}, tt.total, p)
			if pages := result.TotalPages(); pages != tt.expectedPages {
				t.Errorf("TotalPages() = %d, want %d", pages, tt.expectedPages)
			}
		})
	}
}

func TestResult_HasNextAndHasPrev(t *testing.T) {
	tests := []struct {
		name       string
		page       int32
		total      int32
		pageSize   int32
		expectNext bool
		expectPrev bool
	}{
		{"第1页共3页", 1, 30, 10, true, false},
		{"第2页共3页", 2, 30, 10, true, true},
		{"第3页共3页(最后一页)", 3, 30, 10, false, true},
		{"只有1页", 1, 5, 10, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.page, tt.pageSize)
			result := NewResult([]string{}, tt.total, p)

			if hasNext := result.HasNext(); hasNext != tt.expectNext {
				t.Errorf("HasNext() = %v, want %v", hasNext, tt.expectNext)
			}
			if hasPrev := result.HasPrev(); hasPrev != tt.expectPrev {
				t.Errorf("HasPrev() = %v, want %v", hasPrev, tt.expectPrev)
			}
		})
	}
}
