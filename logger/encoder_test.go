package logger

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zapcore"
)

// EncoderTestSuite 编码器测试套件.
type EncoderTestSuite struct {
	suite.Suite
}

func TestEncoderSuite(t *testing.T) {
	suite.Run(t, new(EncoderTestSuite))
}

func (s *EncoderTestSuite) TestBuildEncoderConfig() {
	config := &Config{
		TimeKey:      "ts",
		LevelKey:     "lvl",
		MessageKey:   "message",
		CallerKey:    "source",
		TimeFormat:   TimeFormatISO8601,
		EncodeLevel:  EncodeLevelLower,
		EncodeCaller: EncodeCallerFull,
	}

	encoderConfig := buildEncoderConfig(config)

	s.Equal("ts", encoderConfig.TimeKey)
	s.Equal("lvl", encoderConfig.LevelKey)
	s.Equal("message", encoderConfig.MessageKey)
	s.Equal("source", encoderConfig.CallerKey)
}

func (s *EncoderTestSuite) TestBuildEncoder_JSON() {
	config := &Config{
		Format:     FormatJSON,
		TimeKey:    "timestamp",
		LevelKey:   "level",
		MessageKey: "msg",
		CallerKey:  "caller",
		TimeFormat: TimeFormatDateTime,
	}

	encoder := buildEncoder(config)
	s.NotNil(encoder)
}

func (s *EncoderTestSuite) TestBuildEncoder_Console() {
	config := &Config{
		Format:     FormatConsole,
		TimeKey:    "timestamp",
		LevelKey:   "level",
		MessageKey: "msg",
		CallerKey:  "caller",
		TimeFormat: TimeFormatDateTime,
	}

	encoder := buildEncoder(config)
	s.NotNil(encoder)
}

func (s *EncoderTestSuite) TestGetTimeEncoder() {
	testCases := []string{
		TimeFormatISO8601,
		TimeFormatRFC3339,
		TimeFormatRFC3339Nano,
		TimeFormatEpoch,
		TimeFormatEpochMillis,
		TimeFormatEpochNanos,
		TimeFormatDateTime,
		"2006-01-02", // custom format
	}

	for _, format := range testCases {
		encoder := getTimeEncoder(format)
		s.NotNil(encoder, "format: %s", format)
	}
}

func (s *EncoderTestSuite) TestDatetimeEncoder() {
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	enc := &mockPrimitiveArrayEncoder{}

	datetimeEncoder(testTime, enc)

	s.Equal("2024-01-15 10:30:45", enc.value)
}

func (s *EncoderTestSuite) TestGetLevelEncoder() {
	testCases := []string{
		EncodeLevelCapital,
		EncodeLevelCapitalColor,
		EncodeLevelLower,
		EncodeLevelLowerColor,
		"unknown", // should default to capital
	}

	for _, encode := range testCases {
		encoder := getLevelEncoder(encode)
		s.NotNil(encoder, "encode: %s", encode)
	}
}

func (s *EncoderTestSuite) TestGetCallerEncoder() {
	testCases := []string{
		EncodeCallerShort,
		EncodeCallerFull,
		"unknown", // should default to short
	}

	for _, encode := range testCases {
		encoder := getCallerEncoder(encode)
		s.NotNil(encoder, "encode: %s", encode)
	}
}

func (s *EncoderTestSuite) TestParseLevel() {
	testCases := []struct {
		level string
		want  zapcore.Level
	}{
		{LevelDebug, zapcore.DebugLevel},
		{"DEBUG", zapcore.DebugLevel},
		{LevelInfo, zapcore.InfoLevel},
		{"INFO", zapcore.InfoLevel},
		{LevelWarn, zapcore.WarnLevel},
		{"warning", zapcore.WarnLevel},
		{"WARNING", zapcore.WarnLevel},
		{LevelError, zapcore.ErrorLevel},
		{"ERROR", zapcore.ErrorLevel},
		{LevelFatal, zapcore.FatalLevel},
		{"FATAL", zapcore.FatalLevel},
		{LevelPanic, zapcore.PanicLevel},
		{"PANIC", zapcore.PanicLevel},
		{"unknown", zapcore.InfoLevel}, // default to info
		{"", zapcore.InfoLevel},        // default to info
	}

	for _, tc := range testCases {
		got := parseLevel(tc.level)
		s.Equal(tc.want, got, "level: %s", tc.level)
	}
}

// mockPrimitiveArrayEncoder 用于测试的 mock encoder.
type mockPrimitiveArrayEncoder struct {
	value string
}

func (m *mockPrimitiveArrayEncoder) AppendBool(v bool)              {}
func (m *mockPrimitiveArrayEncoder) AppendByteString(v []byte)      {}
func (m *mockPrimitiveArrayEncoder) AppendComplex128(v complex128)  {}
func (m *mockPrimitiveArrayEncoder) AppendComplex64(v complex64)    {}
func (m *mockPrimitiveArrayEncoder) AppendFloat64(v float64)        {}
func (m *mockPrimitiveArrayEncoder) AppendFloat32(v float32)        {}
func (m *mockPrimitiveArrayEncoder) AppendInt(v int)                {}
func (m *mockPrimitiveArrayEncoder) AppendInt64(v int64)            {}
func (m *mockPrimitiveArrayEncoder) AppendInt32(v int32)            {}
func (m *mockPrimitiveArrayEncoder) AppendInt16(v int16)            {}
func (m *mockPrimitiveArrayEncoder) AppendInt8(v int8)              {}
func (m *mockPrimitiveArrayEncoder) AppendString(v string)          { m.value = v }
func (m *mockPrimitiveArrayEncoder) AppendUint(v uint)              {}
func (m *mockPrimitiveArrayEncoder) AppendUint64(v uint64)          {}
func (m *mockPrimitiveArrayEncoder) AppendUint32(v uint32)          {}
func (m *mockPrimitiveArrayEncoder) AppendUint16(v uint16)          {}
func (m *mockPrimitiveArrayEncoder) AppendUint8(v uint8)            {}
func (m *mockPrimitiveArrayEncoder) AppendUintptr(v uintptr)        {}
func (m *mockPrimitiveArrayEncoder) AppendDuration(v time.Duration) {}
func (m *mockPrimitiveArrayEncoder) AppendTime(v time.Time)         {}
func (m *mockPrimitiveArrayEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	return nil
}
func (m *mockPrimitiveArrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	return nil
}
func (m *mockPrimitiveArrayEncoder) AppendReflected(v interface{}) error {
	return nil
}
