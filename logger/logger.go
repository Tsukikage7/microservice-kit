// Package logger 提供结构化日志记录功能.
package logger

import "context"

// 日志类型常量.
const (
	TypeZap = "zap"
)

// 日志级别常量.
const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
	LevelFatal = "fatal"
	LevelPanic = "panic"
)

// 输出格式常量.
const (
	FormatJSON    = "json"
	FormatConsole = "console"
)

// 输出目标常量.
const (
	OutputConsole = "console"
	OutputFile    = "file"
	OutputBoth    = "both"
)

// 轮转时间常量.
const (
	RotationDaily  = "daily"
	RotationHourly = "hourly"
)

// 时间格式常量.
const (
	TimeFormatISO8601     = "iso8601"
	TimeFormatRFC3339     = "rfc3339"
	TimeFormatRFC3339Nano = "rfc3339nano"
	TimeFormatEpoch       = "epoch"
	TimeFormatEpochMillis = "epochmillis"
	TimeFormatEpochNanos  = "epochnanos"
	TimeFormatDateTime    = "datetime"
)

// 级别编码常量.
const (
	EncodeLevelCapital      = "capital"
	EncodeLevelCapitalColor = "capitalcolor"
	EncodeLevelLower        = "lower"
	EncodeLevelLowerColor   = "lowercolor"
)

// 调用者编码常量.
const (
	EncodeCallerShort = "short"
	EncodeCallerFull  = "full"
)

// contextKey context 键类型.
type contextKey string

// 预定义的 context key，用于存储 trace 信息.
const (
	// TraceIDKey 用于在 context 中存储 traceId.
	TraceIDKey contextKey = "logger:traceId"
	// SpanIDKey 用于在 context 中存储 spanId.
	SpanIDKey contextKey = "logger:spanId"
)

// Field 表示一个日志字段.
type Field struct {
	Key   string
	Value any
}

// Logger 日志记录器接口.
type Logger interface {
	// 基础日志方法
	Debug(args ...any)
	Debugf(format string, args ...any)
	Info(args ...any)
	Infof(format string, args ...any)
	Warn(args ...any)
	Warnf(format string, args ...any)
	Error(args ...any)
	Errorf(format string, args ...any)
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Panic(args ...any)
	Panicf(format string, args ...any)

	// 结构化日志方法
	With(fields ...Field) Logger
	WithContext(ctx context.Context) Logger

	// 生命周期管理
	Sync() error
	Close() error
}

// ContextWithTraceID 将 traceId 注入到 context.
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// ContextWithSpanID 将 spanId 注入到 context.
func ContextWithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// NewLogger 创建 logger 实例.
func NewLogger(config *Config) (Logger, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config.ApplyDefaults()

	switch config.Type {
	case TypeZap, "":
		return newZapLogger(config)
	default:
		return nil, &ConfigError{Field: "type", Message: "unsupported logger type: " + config.Type}
	}
}

// MustNewLogger 创建 logger 实例，失败时 panic.
func MustNewLogger(config *Config) Logger {
	l, err := NewLogger(config)
	if err != nil {
		panic(err)
	}
	return l
}
