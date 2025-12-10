package response

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCStatus 将错误转换为 gRPC Status.
//
// 如果是业务错误，使用对应的 gRPC 状态码；
// 否则返回 Internal 状态.
func GRPCStatus(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}

	code := ExtractCode(err)
	message := ExtractMessage(err)

	return status.New(code.GRPCCode, message)
}

// GRPCError 将错误转换为 gRPC error.
//
// 返回的 error 可直接作为 gRPC 方法的返回值.
func GRPCError(err error) error {
	return GRPCStatus(err).Err()
}

// FromGRPCStatus 从 gRPC Status 提取 Code.
//
// 根据 gRPC 状态码映射到业务错误码.
func FromGRPCStatus(s *status.Status) Code {
	switch s.Code() {
	case codes.OK:
		return CodeSuccess
	case codes.Canceled:
		return CodeCanceled
	case codes.Unknown:
		return CodeUnknown
	case codes.InvalidArgument:
		return CodeInvalidParam
	case codes.DeadlineExceeded:
		return CodeTimeout
	case codes.NotFound:
		return CodeNotFound
	case codes.AlreadyExists:
		return CodeAlreadyExists
	case codes.PermissionDenied:
		return CodeForbidden
	case codes.ResourceExhausted:
		return CodeResourceExhausted
	case codes.Aborted:
		return CodeConflict
	case codes.Unimplemented:
		return CodeNotImplemented
	case codes.Internal:
		return CodeInternal
	case codes.Unavailable:
		return CodeServiceUnavailable
	case codes.Unauthenticated:
		return CodeUnauthorized
	default:
		return CodeUnknown
	}
}

// FromGRPCError 从 gRPC error 提取 Code.
func FromGRPCError(err error) Code {
	if err == nil {
		return CodeSuccess
	}
	s, ok := status.FromError(err)
	if !ok {
		return CodeInternal
	}
	return FromGRPCStatus(s)
}

// GRPCCodeToHTTP 将 gRPC 状态码转换为 HTTP 状态码.
func GRPCCodeToHTTP(c codes.Code) int {
	code := FromGRPCStatus(status.New(c, ""))
	return code.HTTPStatus
}

// HTTPToGRPCCode 将 HTTP 状态码转换为 gRPC 状态码.
func HTTPToGRPCCode(httpStatus int) codes.Code {
	switch {
	case httpStatus >= 200 && httpStatus < 300:
		return codes.OK
	case httpStatus == 400:
		return codes.InvalidArgument
	case httpStatus == 401:
		return codes.Unauthenticated
	case httpStatus == 403:
		return codes.PermissionDenied
	case httpStatus == 404:
		return codes.NotFound
	case httpStatus == 408:
		return codes.DeadlineExceeded
	case httpStatus == 409:
		return codes.AlreadyExists
	case httpStatus == 429:
		return codes.ResourceExhausted
	case httpStatus == 499:
		return codes.Canceled
	case httpStatus == 500:
		return codes.Internal
	case httpStatus == 501:
		return codes.Unimplemented
	case httpStatus == 502, httpStatus == 503, httpStatus == 504:
		return codes.Unavailable
	default:
		return codes.Unknown
	}
}

// NewGRPCError 创建 gRPC 错误.
func NewGRPCError(code Code) error {
	return status.Error(code.GRPCCode, code.Message)
}

// NewGRPCErrorWithMessage 创建带自定义消息的 gRPC 错误.
func NewGRPCErrorWithMessage(code Code, message string) error {
	return status.Error(code.GRPCCode, message)
}

// GRPCInterceptorErrorHandler gRPC 拦截器错误处理.
//
// 将业务错误转换为带有正确状态码的 gRPC 错误.
// 可用于 gRPC 拦截器中统一处理错误.
func GRPCInterceptorErrorHandler(err error) error {
	if err == nil {
		return nil
	}
	return GRPCError(err)
}

// UnaryServerInterceptor 返回 gRPC 一元服务器拦截器.
//
// 自动将业务错误转换为正确的 gRPC 状态码.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return nil, GRPCError(err)
		}
		return resp, nil
	}
}

// StreamServerInterceptor 返回 gRPC 流服务器拦截器.
//
// 自动将业务错误转换为正确的 gRPC 状态码.
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		err := handler(srv, ss)
		if err != nil {
			return GRPCError(err)
		}
		return nil
	}
}
