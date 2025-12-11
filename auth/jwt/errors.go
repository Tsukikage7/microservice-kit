package jwt

import "errors"

// 预定义错误.
var (
	// ErrTokenInvalid 令牌无效.
	ErrTokenInvalid = errors.New("jwt: 令牌无效或已过期")

	// ErrTokenRevoked 令牌已撤销.
	ErrTokenRevoked = errors.New("jwt: 令牌已撤销")

	// ErrTokenEmpty 令牌为空.
	ErrTokenEmpty = errors.New("jwt: 令牌不能为空")

	// ErrTokenNotFound 未找到令牌.
	ErrTokenNotFound = errors.New("jwt: 未找到认证令牌")

	// ErrSigningMethod 签名方法无效.
	ErrSigningMethod = errors.New("jwt: 无效的签名方法")

	// ErrClaimsInvalid Claims 无效.
	ErrClaimsInvalid = errors.New("jwt: 无效的 Claims")

	// ErrRefreshExpired 刷新窗口已过期.
	ErrRefreshExpired = errors.New("jwt: 令牌已超出刷新窗口")
)
