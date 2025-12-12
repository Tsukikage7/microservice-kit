package sorting

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
		{
			name:     "单字段默认降序",
			input:    "created_time",
			expected: "created_time desc",
		},
		{
			name:     "单字段升序",
			input:    "name:asc",
			expected: "name asc",
		},
		{
			name:     "单字段降序",
			input:    "updated_time:desc",
			expected: "updated_time desc",
		},
		{
			name:     "多字段排序",
			input:    "created_time:desc,id:asc",
			expected: "created_time desc, id asc",
		},
		{
			name:     "带空格的输入",
			input:    " name : asc , id : desc ",
			expected: "name asc, id desc",
		},
		{
			name:     "大写排序方向",
			input:    "name:ASC",
			expected: "name asc",
		},
		{
			name:     "无效排序方向使用默认值",
			input:    "name:invalid",
			expected: "name desc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.input)
			if result := s.String(); result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSort_String(t *testing.T) {
	sort := Sort{Field: "created_time", Order: Desc}
	if str := sort.String(); str != "created_time desc" {
		t.Errorf("String() = %q, want %q", str, "created_time desc")
	}
}

func TestSorting_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"空字符串", "", true},
		{"有排序", "name:asc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.input)
			if isEmpty := s.IsEmpty(); isEmpty != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", isEmpty, tt.expected)
			}
		})
	}
}

func TestSorting_First(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedField string
		expectedOrder Order
	}{
		{"空排序", "", "", ""},
		{"单字段", "name:asc", "name", Asc},
		{"多字段返回第一个", "created_time:desc,id:asc", "created_time", Desc},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.input)
			first := s.First()
			if first.Field != tt.expectedField {
				t.Errorf("First().Field = %q, want %q", first.Field, tt.expectedField)
			}
			if first.Order != tt.expectedOrder {
				t.Errorf("First().Order = %q, want %q", first.Order, tt.expectedOrder)
			}
		})
	}
}

func TestSorting_Filter(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		allowedFields []string
		expected      string
	}{
		{
			name:          "过滤不允许的字段",
			input:         "name:asc,password:desc,id:asc",
			allowedFields: []string{"name", "id", "created_time"},
			expected:      "name asc, id asc",
		},
		{
			name:          "所有字段都允许",
			input:         "name:asc,id:desc",
			allowedFields: []string{"name", "id"},
			expected:      "name asc, id desc",
		},
		{
			name:          "所有字段都不允许",
			input:         "password:asc",
			allowedFields: []string{"name", "id"},
			expected:      "",
		},
		{
			name:          "空白名单不过滤",
			input:         "name:asc",
			allowedFields: []string{},
			expected:      "name asc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.input).Filter(tt.allowedFields...)
			if result := s.String(); result != tt.expected {
				t.Errorf("Filter().String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSorting_WithDefault(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultSort  string
		expected     string
	}{
		{
			name:        "空排序使用默认值",
			input:       "",
			defaultSort: "created_time:desc",
			expected:    "created_time desc",
		},
		{
			name:        "有排序不使用默认值",
			input:       "name:asc",
			defaultSort: "created_time:desc",
			expected:    "name asc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.input).WithDefault(tt.defaultSort)
			if result := s.String(); result != tt.expected {
				t.Errorf("WithDefault().String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSorting_ChainedOperations(t *testing.T) {
	// 测试链式调用: 解析 -> 过滤 -> 默认值
	s := New("created_time:desc,password:asc").
		Filter("created_time", "id", "name").
		WithDefault("id:desc")

	expected := "created_time desc"
	if result := s.String(); result != expected {
		t.Errorf("Chained operations result = %q, want %q", result, expected)
	}
}
