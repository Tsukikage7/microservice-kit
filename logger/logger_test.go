package logger

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// LoggerTestSuite logger 测试套件.
type LoggerTestSuite struct {
	suite.Suite
	tmpDir string
}

func TestLoggerSuite(t *testing.T) {
	suite.Run(t, new(LoggerTestSuite))
}

func (s *LoggerTestSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
}

func (s *LoggerTestSuite) TestNewLogger_NilConfig() {
	log, err := NewLogger(nil)
	s.Error(err)
	s.Nil(log)
}

func (s *LoggerTestSuite) TestNewLogger_DefaultConfig() {
	log, err := NewLogger(DefaultConfig())
	s.NoError(err)
	s.NotNil(log)
	defer log.Close()
}

func (s *LoggerTestSuite) TestNewLogger_DevConfig() {
	log, err := NewLogger(NewDevConfig())
	s.NoError(err)
	s.NotNil(log)
	defer log.Close()
}

func (s *LoggerTestSuite) TestNewLogger_InvalidLevel() {
	config := &Config{Level: "invalid"}
	log, err := NewLogger(config)
	s.Error(err)
	s.Nil(log)
}

func (s *LoggerTestSuite) TestNewLogger_InvalidFormat() {
	config := &Config{Format: "invalid"}
	log, err := NewLogger(config)
	s.Error(err)
	s.Nil(log)
}

func (s *LoggerTestSuite) TestNewLogger_InvalidOutput() {
	config := &Config{Output: "invalid"}
	log, err := NewLogger(config)
	s.Error(err)
	s.Nil(log)
}

func (s *LoggerTestSuite) TestNewLogger_FileOutputWithoutLogDir() {
	config := &Config{Output: OutputFile}
	log, err := NewLogger(config)
	s.Error(err)
	s.Nil(log)
}

func (s *LoggerTestSuite) TestNewLogger_UnsupportedType() {
	config := &Config{
		Type:   "unsupported",
		Output: OutputConsole,
	}
	log, err := NewLogger(config)
	s.Error(err)
	s.Nil(log)
}

func (s *LoggerTestSuite) TestNewLogger_WithFileOutput() {
	config := &Config{
		Level:  LevelDebug,
		Format: FormatJSON,
		Output: OutputFile,
		LogDir: s.tmpDir,
		Type:   TypeZap,
	}

	log, err := NewLogger(config)
	s.Require().NoError(err)
	defer log.Close()

	log.Info("test message")
	log.Sync()

	// 验证文件创建
	serviceDir := filepath.Join(s.tmpDir, "service")
	_, err = os.Stat(serviceDir)
	s.NoError(err, "log directory was not created")
}

func (s *LoggerTestSuite) TestNewLogger_WithBothOutput() {
	config := &Config{
		Level:  LevelInfo,
		Format: FormatConsole,
		Output: OutputBoth,
		LogDir: s.tmpDir,
	}

	log, err := NewLogger(config)
	s.Require().NoError(err)
	defer log.Close()

	log.Info("test both output")
}

func (s *LoggerTestSuite) TestLoggerInterface() {
	log, err := NewLogger(NewDevConfig())
	s.Require().NoError(err)
	defer log.Close()

	// 测试所有日志级别方法不会 panic
	s.NotPanics(func() {
		log.Debug("debug message")
		log.Debugf("debug %s", "formatted")
		log.Info("info message")
		log.Infof("info %s", "formatted")
		log.Warn("warn message")
		log.Warnf("warn %s", "formatted")
		log.Error("error message")
		log.Errorf("error %s", "formatted")
	})

	// 测试 With
	logWithFields := log.With(String("key", "value"), Int("count", 42))
	s.NotNil(logWithFields)
	logWithFields.Info("message with fields")

	// 测试 WithContext
	ctx := context.WithValue(context.Background(), TraceIDKey, "abc123")
	logWithCtx := log.WithContext(ctx)
	s.NotNil(logWithCtx)
	logWithCtx.Info("message with context")

	// 测试 Sync 不会 panic
	s.NotPanics(func() {
		_ = log.Sync()
	})
}

func (s *LoggerTestSuite) TestLoggerWith() {
	log, err := NewLogger(DefaultConfig())
	s.Require().NoError(err)
	defer log.Close()

	// 链式调用不会 panic
	s.NotPanics(func() {
		log.With(String("service", "test")).
			With(Int("port", 8080)).
			With(Bool("debug", true)).
			Info("chained fields")
	})
}

func (s *LoggerTestSuite) TestLoggerWithContext() {
	log, err := NewLogger(DefaultConfig())
	s.Require().NoError(err)
	defer log.Close()

	// empty context (使用 context.TODO() 代替 nil)
	logEmpty := log.WithContext(context.TODO())
	s.NotNil(logEmpty, "WithContext(context.TODO()) should return logger")

	// background context
	logBg := log.WithContext(context.Background())
	s.NotNil(logBg)
	logBg.Info("empty context")

	// context with values
	ctx := context.Background()
	ctx = context.WithValue(ctx, TraceIDKey, "trace-123")
	ctx = context.WithValue(ctx, RequestIDKey, "req-456")
	logCtx := log.WithContext(ctx)
	s.NotNil(logCtx)
	logCtx.Info("context with trace and request id")
}

// ConstantsTestSuite 常量测试套件.
type ConstantsTestSuite struct {
	suite.Suite
}

func TestConstantsSuite(t *testing.T) {
	suite.Run(t, new(ConstantsTestSuite))
}

func (s *ConstantsTestSuite) TestTypeConstants() {
	s.Equal("zap", TypeZap)
}

func (s *ConstantsTestSuite) TestLevelConstants() {
	s.Equal("debug", LevelDebug)
	s.Equal("info", LevelInfo)
	s.Equal("warn", LevelWarn)
	s.Equal("error", LevelError)
	s.Equal("fatal", LevelFatal)
	s.Equal("panic", LevelPanic)
}

func (s *ConstantsTestSuite) TestFormatConstants() {
	s.Equal("json", FormatJSON)
	s.Equal("console", FormatConsole)
}

func (s *ConstantsTestSuite) TestOutputConstants() {
	s.Equal("console", OutputConsole)
	s.Equal("file", OutputFile)
	s.Equal("both", OutputBoth)
}

func (s *ConstantsTestSuite) TestContextKeyConstants() {
	s.Equal(ContextKey("trace_id"), TraceIDKey)
	s.Equal(ContextKey("request_id"), RequestIDKey)
}
