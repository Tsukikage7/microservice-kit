package auth

import "errors"

// 认证授权错误定义.
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

	// ErrPrincipalExpired 主体已过期错误.
	ErrPrincipalExpired = errors.New("auth: 主体已过期")

	// ErrInvalidPrincipal 无效主体错误.
	ErrInvalidPrincipal = errors.New("auth: 无效主体")

	// ErrRoleNotFound 角色未找到错误.
	ErrRoleNotFound = errors.New("auth: 角色未找到")

	// ErrPermissionDenied 权限被拒绝错误.
	ErrPermissionDenied = errors.New("auth: 权限被拒绝")
)

// IsUnauthenticated 检查是否为未认证错误.
func IsUnauthenticated(err error) bool {
	return errors.Is(err, ErrUnauthenticated)
}

// IsForbidden 检查是否为无权限错误.
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// IsInvalidCredentials 检查是否为无效凭据错误.
func IsInvalidCredentials(err error) bool {
	return errors.Is(err, ErrInvalidCredentials)
}

// IsCredentialsExpired 检查是否为凭据过期错误.
func IsCredentialsExpired(err error) bool {
	return errors.Is(err, ErrCredentialsExpired)
}
