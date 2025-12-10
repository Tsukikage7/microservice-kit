package response

import (
	"github.com/Tsukikage7/microservice-kit/pagination"
)

// Response 统一响应体.
//
// 泛型参数 T 表示业务数据类型.
//
// 响应格式：
//
//	{
//	    "code": 0,
//	    "message": "成功",
//	    "data": { ... }
//	}
type Response[T any] struct {
	Code    int    `json:"code"`           // 业务状态码
	Message string `json:"message"`        // 响应消息
	Data    T      `json:"data,omitempty"` // 业务数据
}

// PagedResponse 分页响应体.
//
// 响应格式：
//
//	{
//	    "code": 0,
//	    "message": "成功",
//	    "data": [...],
//	    "pagination": { ... }
//	}
type PagedResponse[T any] struct {
	Code       int       `json:"code"`                 // 业务状态码
	Message    string    `json:"message"`              // 响应消息
	Data       []T       `json:"data,omitempty"`       // 业务数据列表
	Pagination *PageInfo `json:"pagination,omitempty"` // 分页信息
}

// PageInfo 分页元数据.
type PageInfo struct {
	Page       int32 `json:"page"`        // 当前页码
	PageSize   int32 `json:"page_size"`   // 每页数量
	Total      int32 `json:"total"`       // 总数
	TotalPages int32 `json:"total_pages"` // 总页数
}

// OK 创建成功响应.
func OK[T any](data T) Response[T] {
	return Response[T]{
		Code:    CodeSuccess.Num,
		Message: CodeSuccess.Message,
		Data:    data,
	}
}

// OKWithMessage 创建带自定义消息的成功响应.
func OKWithMessage[T any](data T, message string) Response[T] {
	return Response[T]{
		Code:    CodeSuccess.Num,
		Message: message,
		Data:    data,
	}
}

// Fail 创建失败响应.
func Fail[T any](code Code) Response[T] {
	var zero T
	return Response[T]{
		Code:    code.Num,
		Message: code.Message,
		Data:    zero,
	}
}

// FailWithMessage 创建带自定义消息的失败响应.
func FailWithMessage[T any](code Code, message string) Response[T] {
	var zero T
	return Response[T]{
		Code:    code.Num,
		Message: message,
		Data:    zero,
	}
}

// FailWithError 从 error 创建失败响应.
func FailWithError[T any](err error) Response[T] {
	code := ExtractCode(err)
	message := ExtractMessage(err)
	var zero T
	return Response[T]{
		Code:    code.Num,
		Message: message,
		Data:    zero,
	}
}

// Paged 创建分页响应.
func Paged[T any](result pagination.Result[T]) PagedResponse[T] {
	return PagedResponse[T]{
		Code:    CodeSuccess.Num,
		Message: CodeSuccess.Message,
		Data:    result.Items,
		Pagination: &PageInfo{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages(),
		},
	}
}

// PagedWithMessage 创建带自定义消息的分页响应.
func PagedWithMessage[T any](result pagination.Result[T], message string) PagedResponse[T] {
	return PagedResponse[T]{
		Code:    CodeSuccess.Num,
		Message: message,
		Data:    result.Items,
		Pagination: &PageInfo{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages(),
		},
	}
}

// PagedFail 创建分页失败响应.
func PagedFail[T any](code Code) PagedResponse[T] {
	return PagedResponse[T]{
		Code:    code.Num,
		Message: code.Message,
	}
}

// PagedFailWithMessage 创建带自定义消息的分页失败响应.
func PagedFailWithMessage[T any](code Code, message string) PagedResponse[T] {
	return PagedResponse[T]{
		Code:    code.Num,
		Message: message,
	}
}

// IsSuccess 判断是否成功响应.
func (r Response[T]) IsSuccess() bool {
	return r.Code == CodeSuccess.Num
}

// IsSuccess 判断是否成功响应.
func (r PagedResponse[T]) IsSuccess() bool {
	return r.Code == CodeSuccess.Num
}
