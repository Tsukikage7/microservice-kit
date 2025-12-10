package response

import (
	"net/http"

	"google.golang.org/grpc/codes"
)

// Code 业务错误码.
type Code struct {
	Num        int        // 数字错误码
	Message    string     // 默认错误消息
	HTTPStatus int        // 对应的 HTTP 状态码
	GRPCCode   codes.Code // 对应的 gRPC 状态码
}

// Error 实现 error 接口.
func (c Code) Error() string {
	return c.Message
}

// WithMessage 创建带自定义消息的错误码副本.
func (c Code) WithMessage(msg string) Code {
	c.Message = msg
	return c
}

// Is 判断是否为指定错误码.
func (c Code) Is(target Code) bool {
	return c.Num == target.Num
}

// 预定义错误码.
//
// 错误码规范：
//   - 0: 成功
//   - 1xxxx: 通用错误
//   - 2xxxx: 认证/授权错误
//   - 3xxxx: 请求参数错误
//   - 4xxxx: 资源错误
//   - 5xxxx: 服务器内部错误
//   - 6xxxx: 外部服务错误
var (
	// 成功
	CodeSuccess = Code{0, "成功", http.StatusOK, codes.OK}

	// 通用错误 1xxxx
	CodeUnknown  = Code{10000, "未知错误", http.StatusInternalServerError, codes.Unknown}
	CodeCanceled = Code{10001, "请求已取消", http.StatusRequestTimeout, codes.Canceled}
	CodeTimeout  = Code{10002, "请求超时", http.StatusGatewayTimeout, codes.DeadlineExceeded}

	// 认证/授权错误 2xxxx
	CodeUnauthorized = Code{20001, "未授权", http.StatusUnauthorized, codes.Unauthenticated}
	CodeForbidden    = Code{20002, "禁止访问", http.StatusForbidden, codes.PermissionDenied}
	CodeTokenExpired = Code{20003, "令牌已过期", http.StatusUnauthorized, codes.Unauthenticated}
	CodeTokenInvalid = Code{20004, "令牌无效", http.StatusUnauthorized, codes.Unauthenticated}

	// 请求参数错误 3xxxx
	CodeInvalidParam     = Code{30001, "参数无效", http.StatusBadRequest, codes.InvalidArgument}
	CodeMissingParam     = Code{30002, "缺少必需参数", http.StatusBadRequest, codes.InvalidArgument}
	CodeValidationFailed = Code{30003, "参数验证失败", http.StatusBadRequest, codes.InvalidArgument}

	// 资源错误 4xxxx
	CodeNotFound          = Code{40001, "资源不存在", http.StatusNotFound, codes.NotFound}
	CodeAlreadyExists     = Code{40002, "资源已存在", http.StatusConflict, codes.AlreadyExists}
	CodeConflict          = Code{40003, "资源冲突", http.StatusConflict, codes.Aborted}
	CodeResourceExhausted = Code{40004, "资源耗尽", http.StatusTooManyRequests, codes.ResourceExhausted}

	// 服务器内部错误 5xxxx
	CodeInternal       = Code{50001, "服务器内部错误", http.StatusInternalServerError, codes.Internal}
	CodeNotImplemented = Code{50002, "功能未实现", http.StatusNotImplemented, codes.Unimplemented}
	CodeDatabaseError  = Code{50003, "数据库错误", http.StatusInternalServerError, codes.Internal}

	// 外部服务错误 6xxxx
	CodeServiceUnavailable = Code{60001, "服务不可用", http.StatusServiceUnavailable, codes.Unavailable}
	CodeUpstreamError      = Code{60002, "上游服务错误", http.StatusBadGateway, codes.Unavailable}
)

// NewCode 创建自定义错误码.
func NewCode(num int, message string, httpStatus int, grpcCode codes.Code) Code {
	return Code{
		Num:        num,
		Message:    message,
		HTTPStatus: httpStatus,
		GRPCCode:   grpcCode,
	}
}
