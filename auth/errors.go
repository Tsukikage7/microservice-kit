package auth

import "errors"

// 认证授权错误.
var (
	// ErrUnauthenticated 未认证错误.
	ErrUnauthenticated = errors.New("auth: 未认证")

	// ErrForbidden 无权限错误.
	ErrForbidden = errors.New("auth: 无权限")

	// ErrInvalidCredentials 无效凭据错误.
	ErrInvalidCredentials = errors.New("auth: 无效凭据")

	// ErrCredentialsExpired 凭据已过期错误.
	ErrCredentialsExpired = errors.New("auth: 凭据已过期")

	// ErrCredentialsNotFound 凭据未找到错误.
	ErrCredentialsNotFound = errors.New("auth: 凭据未找到")
)

// IsUnauthenticated 检查是否为未认证错误.
func IsUnauthenticated(err error) bool {
	return errors.Is(err, ErrUnauthenticated)
}

// IsForbidden 检查是否为无权限错误.
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}
