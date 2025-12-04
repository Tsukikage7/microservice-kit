// Package logger 提供结构化日志记录功能.
package logger

import (
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// buildEncoderConfig 构建 zap 编码器配置.
func buildEncoderConfig(config *Config) zapcore.EncoderConfig {
	encoderConfig := zap.NewProductionEncoderConfig()

	encoderConfig.TimeKey = config.TimeKey
	encoderConfig.LevelKey = config.LevelKey
	encoderConfig.MessageKey = config.MessageKey
	encoderConfig.CallerKey = config.CallerKey

	encoderConfig.EncodeTime = getTimeEncoder(config.TimeFormat)
	encoderConfig.EncodeLevel = getLevelEncoder(config.EncodeLevel)
	encoderConfig.EncodeCaller = getCallerEncoder(config.EncodeCaller)

	return encoderConfig
}

// buildEncoder 构建编码器.
func buildEncoder(config *Config) zapcore.Encoder {
	encoderConfig := buildEncoderConfig(config)

	if strings.ToLower(config.Format) == FormatJSON {
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// getTimeEncoder 获取时间编码器.
func getTimeEncoder(format string) zapcore.TimeEncoder {
	switch strings.ToLower(format) {
	case TimeFormatISO8601:
		return zapcore.ISO8601TimeEncoder
	case TimeFormatRFC3339:
		return zapcore.RFC3339TimeEncoder
	case TimeFormatRFC3339Nano:
		return zapcore.RFC3339NanoTimeEncoder
	case TimeFormatEpoch:
		return zapcore.EpochTimeEncoder
	case TimeFormatEpochMillis:
		return zapcore.EpochMillisTimeEncoder
	case TimeFormatEpochNanos:
		return zapcore.EpochNanosTimeEncoder
	case TimeFormatDateTime:
		return datetimeEncoder
	default:
		return zapcore.TimeEncoderOfLayout(format)
	}
}

// datetimeEncoder 自定义日期时间编码器.
func datetimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}

// getLevelEncoder 获取级别编码器.
func getLevelEncoder(encode string) zapcore.LevelEncoder {
	switch strings.ToLower(encode) {
	case EncodeLevelCapital:
		return zapcore.CapitalLevelEncoder
	case EncodeLevelCapitalColor:
		return zapcore.CapitalColorLevelEncoder
	case EncodeLevelLower:
		return zapcore.LowercaseLevelEncoder
	case EncodeLevelLowerColor:
		return zapcore.LowercaseColorLevelEncoder
	default:
		return zapcore.CapitalLevelEncoder
	}
}

// getCallerEncoder 获取调用者编码器.
func getCallerEncoder(encode string) zapcore.CallerEncoder {
	switch strings.ToLower(encode) {
	case EncodeCallerShort:
		return zapcore.ShortCallerEncoder
	case EncodeCallerFull:
		return zapcore.FullCallerEncoder
	default:
		return zapcore.ShortCallerEncoder
	}
}

// parseLevel 解析日志级别.
func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelInfo:
		return zapcore.InfoLevel
	case LevelWarn, "warning":
		return zapcore.WarnLevel
	case LevelError:
		return zapcore.ErrorLevel
	case LevelFatal:
		return zapcore.FatalLevel
	case LevelPanic:
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}
