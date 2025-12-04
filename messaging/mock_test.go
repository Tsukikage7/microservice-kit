package messaging

import (
	"context"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// mockLogger 用于测试的模拟日志器.
type mockLogger struct {
	debugCalled bool
	errorCalled bool
	warnCalled  bool
}

func (m *mockLogger) Debug(args ...any)                             { m.debugCalled = true }
func (m *mockLogger) Debugf(format string, args ...any)             { m.debugCalled = true }
func (m *mockLogger) Info(args ...any)                              {}
func (m *mockLogger) Infof(format string, args ...any)              {}
func (m *mockLogger) Warn(args ...any)                              { m.warnCalled = true }
func (m *mockLogger) Warnf(format string, args ...any)              { m.warnCalled = true }
func (m *mockLogger) Error(args ...any)                             { m.errorCalled = true }
func (m *mockLogger) Errorf(format string, args ...any)             { m.errorCalled = true }
func (m *mockLogger) Fatal(args ...any)                             {}
func (m *mockLogger) Fatalf(format string, args ...any)             {}
func (m *mockLogger) Panic(args ...any)                             {}
func (m *mockLogger) Panicf(format string, args ...any)             {}
func (m *mockLogger) With(fields ...logger.Field) logger.Logger     { return m }
func (m *mockLogger) WithContext(ctx context.Context) logger.Logger { return m }
func (m *mockLogger) Sync() error                                   { return nil }
func (m *mockLogger) Close() error                                  { return nil }
