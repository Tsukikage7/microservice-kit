package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// ZapLoggerTestSuite zap logger 测试套件.
type ZapLoggerTestSuite struct {
	suite.Suite
	tmpDir string
}

func TestZapLoggerSuite(t *testing.T) {
	suite.Run(t, new(ZapLoggerTestSuite))
}

func (s *ZapLoggerTestSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
}

func (s *ZapLoggerTestSuite) TestNewZapLogger() {
	config := DefaultConfig()
	config.ApplyDefaults()

	log, err := newZapLogger(config)

	s.NoError(err)
	s.NotNil(log)
	defer log.Close()
}

func (s *ZapLoggerTestSuite) TestNewZapLogger_WithFileOutput() {
	config := &Config{
		Level:           LevelDebug,
		Format:          FormatJSON,
		Output:          OutputFile,
		LogDir:          s.tmpDir,
		ServiceName:     "test-service",
		RotationEnabled: true,
		RotationTime:    RotationDaily,
		MaxAge:          7,
	}
	config.ApplyDefaults()

	log, err := newZapLogger(config)
	s.Require().NoError(err)
	defer log.Close()

	log.Info("test message")
	log.Sync()

	// 验证文件创建
	serviceDir := filepath.Join(s.tmpDir, "test-service")
	files, err := os.ReadDir(serviceDir)
	s.NoError(err)
	s.NotEmpty(files)
}

func (s *ZapLoggerTestSuite) TestNewZapLogger_LevelSeparate() {
	config := &Config{
		Level:           LevelDebug,
		Format:          FormatJSON,
		Output:          OutputFile,
		LogDir:          s.tmpDir,
		LevelSeparate:   true,
		RotationEnabled: true,
	}
	config.ApplyDefaults()

	log, err := newZapLogger(config)
	s.Require().NoError(err)
	defer log.Close()

	log.Debug("debug message")
	log.Info("info message")
	log.Warn("warn message")
	log.Error("error message")
	log.Sync()

	// 验证各级别目录创建
	expectedDirs := []string{"debug", "info", "warn", "error"}
	for _, dir := range expectedDirs {
		levelDir := filepath.Join(s.tmpDir, dir)
		_, err := os.Stat(levelDir)
		s.NoError(err, "level directory %v was not created", dir)
	}
}

func (s *ZapLoggerTestSuite) TestNewZapLogger_StaticFile() {
	config := &Config{
		Level:           LevelInfo,
		Format:          FormatJSON,
		Output:          OutputFile,
		LogDir:          s.tmpDir,
		ServiceName:     "static-service",
		RotationEnabled: false,
	}
	config.ApplyDefaults()

	log, err := newZapLogger(config)
	s.Require().NoError(err)
	defer log.Close()

	log.Info("test static file")
	log.Sync()

	staticFile := filepath.Join(s.tmpDir, "static-service", "static-service.log")
	_, err = os.Stat(staticFile)
	s.NoError(err, "static log file was not created")
}

func (s *ZapLoggerTestSuite) TestAllLogLevels() {
	log, err := NewLogger(NewDevConfig())
	s.Require().NoError(err)
	defer log.Close()

	// 测试所有日志方法不会 panic
	s.NotPanics(func() {
		log.Debug("debug")
		log.Debugf("debug %s", "formatted")
		log.Info("info")
		log.Infof("info %s", "formatted")
		log.Warn("warn")
		log.Warnf("warn %s", "formatted")
		log.Error("error")
		log.Errorf("error %s", "formatted")
	})
}

func (s *ZapLoggerTestSuite) TestWith() {
	log, err := NewLogger(DefaultConfig())
	s.Require().NoError(err)
	defer log.Close()

	logWithFields := log.With(String("key", "value"))
	s.NotNil(logWithFields)

	// 原 logger 不应该受影响
	logWithFields.Info("with fields")
	log.Info("without fields")
}

func (s *ZapLoggerTestSuite) TestWithMultipleFields() {
	log, err := NewLogger(DefaultConfig())
	s.Require().NoError(err)
	defer log.Close()

	logWithFields := log.With(
		String("service", "test"),
		Int("port", 8080),
		Bool("debug", true),
		Float64("rate", 0.95),
		Int64("count", 1000000),
	)

	s.NotPanics(func() {
		logWithFields.Info("multiple fields")
	})
}

func (s *ZapLoggerTestSuite) TestWithContext() {
	log, err := NewLogger(DefaultConfig())
	s.Require().NoError(err)
	defer log.Close()

	// empty context
	logEmpty := log.WithContext(context.TODO())
	s.NotNil(logEmpty)

	// background context
	logBg := log.WithContext(context.Background())
	s.NotNil(logBg)

	// context with trace_id
	ctx := context.WithValue(context.Background(), TraceIDKey, "abc123")
	logWithTrace := log.WithContext(ctx)
	logWithTrace.Info("with trace")

	// context with request_id
	ctx = context.WithValue(ctx, RequestIDKey, "req-456")
	logWithBoth := log.WithContext(ctx)
	logWithBoth.Info("with trace and request")
}

func (s *ZapLoggerTestSuite) TestSync() {
	log, err := NewLogger(DefaultConfig())
	s.Require().NoError(err)
	defer log.Close()

	log.Info("test sync")

	// Sync 可能返回错误（对于 stdout），但不应该 panic
	s.NotPanics(func() {
		_ = log.Sync()
	})
}

func (s *ZapLoggerTestSuite) TestClose() {
	config := &Config{
		Level:           LevelInfo,
		Format:          FormatJSON,
		Output:          OutputFile,
		LogDir:          s.tmpDir,
		RotationEnabled: true,
	}

	log, err := NewLogger(config)
	s.Require().NoError(err)

	log.Info("before close")
	err = log.Close()
	s.NoError(err)
}

func (s *ZapLoggerTestSuite) TestJSONOutput() {
	config := &Config{
		Level:           LevelInfo,
		Format:          FormatJSON,
		Output:          OutputFile,
		LogDir:          s.tmpDir,
		ServiceName:     "json-test",
		RotationEnabled: false,
		TimeKey:         "timestamp",
		LevelKey:        "level",
		MessageKey:      "msg",
	}

	log, err := NewLogger(config)
	s.Require().NoError(err)

	log.Info("test json output")
	log.Close()

	// 读取并验证 JSON 格式
	logFile := filepath.Join(s.tmpDir, "json-test", "json-test.log")
	content, err := os.ReadFile(logFile)
	s.Require().NoError(err)

	var logEntry map[string]interface{}
	err = json.Unmarshal(bytes.TrimSpace(content), &logEntry)
	s.Require().NoError(err)

	s.Equal("test json output", logEntry["msg"])
	s.Equal("INFO", logEntry["level"])
}

func (s *ZapLoggerTestSuite) TestWithCaller() {
	config := &Config{
		Level:           LevelInfo,
		Format:          FormatJSON,
		Output:          OutputFile,
		LogDir:          s.tmpDir,
		ServiceName:     "caller-test",
		EnableCaller:    true,
		CallerSkip:      1,
		RotationEnabled: false,
	}

	log, err := NewLogger(config)
	s.Require().NoError(err)

	log.Info("test with caller")
	log.Close()

	logFile := filepath.Join(s.tmpDir, "caller-test", "caller-test.log")
	content, err := os.ReadFile(logFile)
	s.Require().NoError(err)

	var logEntry map[string]interface{}
	err = json.Unmarshal(bytes.TrimSpace(content), &logEntry)
	s.Require().NoError(err)

	caller, ok := logEntry["caller"]
	s.True(ok, "caller field not found")
	s.NotEmpty(caller)
}

func (s *ZapLoggerTestSuite) TestConsoleOutput() {
	config := &Config{
		Level:  LevelDebug,
		Format: FormatConsole,
		Output: OutputConsole,
	}

	log, err := NewLogger(config)
	s.Require().NoError(err)
	defer log.Close()

	s.NotPanics(func() {
		log.Debug("debug message")
		log.Info("info message")
		log.Warn("warn message")
		log.Error("error message")
	})
}

func (s *ZapLoggerTestSuite) TestBothOutput() {
	config := &Config{
		Level:           LevelInfo,
		Format:          FormatJSON,
		Output:          OutputBoth,
		LogDir:          s.tmpDir,
		ServiceName:     "both-test",
		RotationEnabled: true,
	}

	log, err := NewLogger(config)
	s.Require().NoError(err)
	defer log.Close()

	log.Info("both output test")
	log.Sync()

	serviceDir := filepath.Join(s.tmpDir, "both-test")
	files, err := os.ReadDir(serviceDir)
	s.NoError(err)
	s.NotEmpty(files)
}

func (s *ZapLoggerTestSuite) TestChainedWith() {
	log, err := NewLogger(DefaultConfig())
	s.Require().NoError(err)
	defer log.Close()

	s.NotPanics(func() {
		log.With(String("service", "api")).
			With(String("version", "v1")).
			With(Int("port", 8080)).
			Info("chained with")
	})
}

// BuildOptionsTestSuite buildOptions 测试套件.
type BuildOptionsTestSuite struct {
	suite.Suite
}

func TestBuildOptionsSuite(t *testing.T) {
	suite.Run(t, new(BuildOptionsTestSuite))
}

func (s *BuildOptionsTestSuite) TestBuildOptions() {
	testCases := []struct {
		name             string
		enableCaller     bool
		callerSkip       int
		enableStacktrace bool
		wantLen          int
	}{
		{"no options", false, 0, false, 0},
		{"caller only", true, 0, false, 1},
		{"caller with skip", true, 2, false, 2},
		{"stacktrace only", false, 0, true, 1},
		{"all options", true, 1, true, 3},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			config := &Config{
				EnableCaller:     tc.enableCaller,
				CallerSkip:       tc.callerSkip,
				EnableStacktrace: tc.enableStacktrace,
			}
			options := buildOptions(config)
			s.Len(options, tc.wantLen)
		})
	}
}

// FieldConstructorTestSuite 字段构造函数测试套件.
type FieldConstructorTestSuite struct {
	suite.Suite
}

func TestFieldConstructorSuite(t *testing.T) {
	suite.Run(t, new(FieldConstructorTestSuite))
}

func (s *FieldConstructorTestSuite) TestString() {
	f := String("key", "value")
	s.Equal("key", f.Key)
	s.Equal("value", f.Value)
}

func (s *FieldConstructorTestSuite) TestInt() {
	f := Int("count", 42)
	s.Equal("count", f.Key)
	s.Equal(42, f.Value)
}

func (s *FieldConstructorTestSuite) TestInt64() {
	f := Int64("bignum", 9223372036854775807)
	s.Equal("bignum", f.Key)
	s.Equal(int64(9223372036854775807), f.Value)
}

func (s *FieldConstructorTestSuite) TestFloat64() {
	f := Float64("rate", 3.14)
	s.Equal("rate", f.Key)
	s.Equal(3.14, f.Value)
}

func (s *FieldConstructorTestSuite) TestBool() {
	f := Bool("enabled", true)
	s.Equal("enabled", f.Key)
	s.Equal(true, f.Value)
}

func (s *FieldConstructorTestSuite) TestTime() {
	now := time.Now()
	f := Time("timestamp", now)
	s.Equal("timestamp", f.Key)
	s.Equal(now, f.Value)
}

func (s *FieldConstructorTestSuite) TestDuration() {
	d := 5 * time.Second
	f := Duration("elapsed", d)
	s.Equal("elapsed", f.Key)
	s.Equal(d, f.Value)
}

func (s *FieldConstructorTestSuite) TestErr() {
	err := errors.New("test error")
	f := Err(err)
	s.Equal("error", f.Key)
	s.Equal(err, f.Value)
}

func (s *FieldConstructorTestSuite) TestAny() {
	data := map[string]int{"a": 1}
	f := Any("data", data)
	s.Equal("data", f.Key)
}

// FileWriterTestSuite 文件写入器测试套件.
type FileWriterTestSuite struct {
	suite.Suite
	tmpDir string
}

func TestFileWriterSuite(t *testing.T) {
	suite.Run(t, new(FileWriterTestSuite))
}

func (s *FileWriterTestSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
}

func (s *FileWriterTestSuite) TestCreateFileWriter_RotationEnabled() {
	config := &Config{
		LogDir:          s.tmpDir,
		RotationEnabled: true,
		MaxAge:          7,
		Compress:        true,
		RotationTime:    RotationDaily,
	}

	writer, err := createFileWriter(config, "test")
	s.NoError(err)
	s.NotNil(writer)
	defer writer.Close()
}

func (s *FileWriterTestSuite) TestCreateFileWriter_StaticFile() {
	config := &Config{
		LogDir:          s.tmpDir,
		RotationEnabled: false,
	}

	writer, err := createFileWriter(config, "static")
	s.NoError(err)
	s.NotNil(writer)
	defer writer.Close()

	staticFile := filepath.Join(s.tmpDir, "static", "static.log")
	_, err = os.Stat(staticFile)
	s.NoError(err)
}
