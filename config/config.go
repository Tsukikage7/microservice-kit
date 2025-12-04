// Package config 提供配置加载和管理功能.
package config

import (
	"errors"
	"path/filepath"
	"strings"
)

// 常见错误.
var (
	ErrNilConfig    = errors.New("配置为空")
	ErrFileNotFound = errors.New("配置文件不存在")
	ErrInvalidType  = errors.New("不支持的配置文件类型")
)

// Validatable 可验证的配置接口.
type Validatable interface {
	Validate() error
}

// GetConfigType 根据文件扩展名获取配置类型.
func GetConfigType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".toml":
		return "toml"
	case ".ini":
		return "ini"
	case ".env":
		return "env"
	case ".properties":
		return "properties"
	default:
		return ""
	}
}
