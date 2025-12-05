package jwt

import (
	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT Claims 接口.
//
// 自定义 Claims 需要实现此接口.
type Claims interface {
	jwt.Claims
}

// StandardClaims 标准 Claims 实现.
//
// 提供基础字段，可嵌入自定义 Claims 中扩展.
type StandardClaims struct {
	jwt.RegisteredClaims
}

// MapClaims 基于 map 的 Claims.
//
// 适用于简单场景或动态字段.
type MapClaims = jwt.MapClaims
