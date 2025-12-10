package response

import (
	"errors"
	"fmt"
)

// 注意：ExtractMessage 对内部错误（5xxxx、6xxxx）会隐藏详细信息。
// 如需完整错误信息（用于日志），请使用 ExtractMessageUnsafe。

// BusinessError 业务错误.
//
// 实现 error 接口，可用于在业务层传递错误码信息.
type BusinessError struct {
	Code    Code   // 错误码
	Message string // 自定义错误消息（可选）
	Cause   error  // 原始错误（可选）
}

// Error 实现 error 接口.
func (e *BusinessError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.Code.Message
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", msg, e.Cause)
	}
	return msg
}

// Unwrap 返回原始错误.
func (e *BusinessError) Unwrap() error {
	return e.Cause
}

// GetCode 获取错误码.
func (e *BusinessError) GetCode() Code {
	return e.Code
}

// GetMessage 获取错误消息.
func (e *BusinessError) GetMessage() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code.Message
}

// NewError 创建业务错误.
func NewError(code Code) *BusinessError {
	return &BusinessError{
		Code: code,
	}
}

// NewErrorWithMessage 创建带自定义消息的业务错误.
func NewErrorWithMessage(code Code, message string) *BusinessError {
	return &BusinessError{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithCause 创建带原始错误的业务错误.
func NewErrorWithCause(code Code, cause error) *BusinessError {
	return &BusinessError{
		Code:  code,
		Cause: cause,
	}
}

// NewErrorFull 创建完整的业务错误.
func NewErrorFull(code Code, message string, cause error) *BusinessError {
	return &BusinessError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Wrap 包装错误为业务错误.
func Wrap(code Code, err error) *BusinessError {
	return &BusinessError{
		Code:  code,
		Cause: err,
	}
}

// WrapWithMessage 包装错误为带消息的业务错误.
func WrapWithMessage(code Code, message string, err error) *BusinessError {
	return &BusinessError{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// IsBusinessError 判断是否为业务错误.
func IsBusinessError(err error) bool {
	var bizErr *BusinessError
	return errors.As(err, &bizErr)
}

// AsBusinessError 将错误转换为业务错误.
//
// 如果不是业务错误，返回 nil.
func AsBusinessError(err error) *BusinessError {
	var bizErr *BusinessError
	if errors.As(err, &bizErr) {
		return bizErr
	}
	return nil
}

// ExtractCode 从错误中提取错误码.
//
// 如果是业务错误，返回对应的错误码；
// 否则返回 CodeInternal.
func ExtractCode(err error) Code {
	if err == nil {
		return CodeSuccess
	}

	var bizErr *BusinessError
	if errors.As(err, &bizErr) {
		return bizErr.Code
	}

	// 检查是否直接是 Code 类型
	var code Code
	if errors.As(err, &code) {
		return code
	}

	return CodeInternal
}

// ExtractMessage 从错误中提取错误消息.
//
// 对于内部错误（5xxxx、6xxxx），返回通用消息，避免暴露敏感信息.
func ExtractMessage(err error) string {
	if err == nil {
		return CodeSuccess.Message
	}

	code := ExtractCode(err)

	// 内部错误：不暴露详细信息
	if code.Num >= 50000 {
		return code.Message
	}

	// 业务错误：返回具体消息
	var bizErr *BusinessError
	if errors.As(err, &bizErr) {
		return bizErr.GetMessage()
	}

	return code.Message
}

// ExtractMessageUnsafe 从错误中提取完整错误消息（包含敏感信息）.
//
// 仅用于日志记录，不应返回给客户端.
func ExtractMessageUnsafe(err error) string {
	if err == nil {
		return CodeSuccess.Message
	}

	var bizErr *BusinessError
	if errors.As(err, &bizErr) {
		if bizErr.Cause != nil {
			return fmt.Sprintf("%s: %v", bizErr.GetMessage(), bizErr.Cause)
		}
		return bizErr.GetMessage()
	}

	return err.Error()
}
