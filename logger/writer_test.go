package logger

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// RotateWriterTestSuite 轮转写入器测试套件.
type RotateWriterTestSuite struct {
	suite.Suite
	tmpDir string
}

func TestRotateWriterSuite(t *testing.T) {
	suite.Run(t, new(RotateWriterTestSuite))
}

func (s *RotateWriterTestSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
}

func (s *RotateWriterTestSuite) TestNewRotateWriter() {
	writer := NewRotateWriter(s.tmpDir, "test")
	s.NotNil(writer)
	defer writer.Close()
}

func (s *RotateWriterTestSuite) TestNewRotateWriter_WithOptions() {
	writer := NewRotateWriter(
		s.tmpDir,
		"test",
		WithMaxAge(30),
		WithCompress(true),
		WithRotationMode(RotationHourly),
	)
	s.NotNil(writer)
	defer writer.Close()

	rw := writer.(*rotateWriter)
	s.Equal(30*24*time.Hour, rw.maxAge)
	s.True(rw.compress)
	s.Equal(RotationHourly, rw.rotationMode)
}

func (s *RotateWriterTestSuite) TestWrite() {
	writer := NewRotateWriter(s.tmpDir, "test")
	defer writer.Close()

	data := []byte("test log message\n")
	n, err := writer.Write(data)

	s.NoError(err)
	s.Equal(len(data), n)

	// 验证文件创建
	logDir := filepath.Join(s.tmpDir, "test")
	files, err := os.ReadDir(logDir)
	s.NoError(err)
	s.NotEmpty(files)

	// 验证文件内容
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".log") {
			content, err := os.ReadFile(filepath.Join(logDir, file.Name()))
			s.NoError(err)
			s.Equal(string(data), string(content))
		}
	}
}

func (s *RotateWriterTestSuite) TestMultipleWrites() {
	writer := NewRotateWriter(s.tmpDir, "test")
	defer writer.Close()

	for i := 0; i < 100; i++ {
		data := []byte("test log message\n")
		_, err := writer.Write(data)
		s.NoError(err)
	}

	logDir := filepath.Join(s.tmpDir, "test")
	files, err := os.ReadDir(logDir)
	s.NoError(err)
	s.NotEmpty(files)
}

func (s *RotateWriterTestSuite) TestSync() {
	writer := NewRotateWriter(s.tmpDir, "test")
	defer writer.Close()

	_, err := writer.Write([]byte("test\n"))
	s.NoError(err)

	err = writer.Sync()
	s.NoError(err)
}

func (s *RotateWriterTestSuite) TestSyncWithoutFile() {
	writer := NewRotateWriter(s.tmpDir, "test")
	defer writer.Close()

	// Sync without any writes
	err := writer.Sync()
	s.NoError(err)
}

func (s *RotateWriterTestSuite) TestClose() {
	writer := NewRotateWriter(s.tmpDir, "test")

	_, err := writer.Write([]byte("test\n"))
	s.NoError(err)

	err = writer.Close()
	s.NoError(err)

	// 再次 Close 应该没问题
	err = writer.Close()
	s.NoError(err)
}

func (s *RotateWriterTestSuite) TestFileNaming_Daily() {
	writer := NewRotateWriter(s.tmpDir, "app", WithRotationMode(RotationDaily))
	defer writer.Close()

	_, err := writer.Write([]byte("test\n"))
	s.NoError(err)

	logDir := filepath.Join(s.tmpDir, "app")
	files, _ := os.ReadDir(logDir)

	today := time.Now().Format("2006-01-02")
	expectedName := "app_" + today + ".log"

	found := false
	for _, file := range files {
		if file.Name() == expectedName {
			found = true
			break
		}
	}
	s.True(found, "expected file %v not found", expectedName)
}

func (s *RotateWriterTestSuite) TestFileNaming_Hourly() {
	writer := NewRotateWriter(s.tmpDir, "app", WithRotationMode(RotationHourly))
	defer writer.Close()

	_, err := writer.Write([]byte("test\n"))
	s.NoError(err)

	logDir := filepath.Join(s.tmpDir, "app")
	files, _ := os.ReadDir(logDir)

	now := time.Now()
	expectedPrefix := "app_" + now.Format("2006-01-02") + "_"

	found := false
	for _, file := range files {
		if strings.HasPrefix(file.Name(), expectedPrefix) && strings.HasSuffix(file.Name(), ".log") {
			found = true
			break
		}
	}
	s.True(found, "expected file with prefix %v not found", expectedPrefix)
}

func (s *RotateWriterTestSuite) TestConcurrentWrites() {
	writer := NewRotateWriter(s.tmpDir, "concurrent")
	defer writer.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = writer.Write([]byte("goroutine write\n"))
			}
		}()
	}
	wg.Wait()

	logDir := filepath.Join(s.tmpDir, "concurrent")
	files, err := os.ReadDir(logDir)
	s.NoError(err)
	s.NotEmpty(files)
}

func (s *RotateWriterTestSuite) TestIsLogFile() {
	rw := &rotateWriter{prefix: "app"}

	testCases := []struct {
		filename string
		want     bool
	}{
		{"app_2024-01-01.log", true},
		{"app_2024-01-01_12.log", true},
		{"app_2024-01-01.log.gz", true},
		{"other_2024-01-01.log", false},
		{"app.log", false},
		{"app_", false},
	}

	for _, tc := range testCases {
		s.Equal(tc.want, rw.isLogFile(tc.filename), "filename: %s", tc.filename)
	}
}

func (s *RotateWriterTestSuite) TestRotate() {
	writer := NewRotateWriter(s.tmpDir, "rotate-test", WithMaxAge(1))
	defer writer.Close()

	// 写入数据创建文件
	_, err := writer.Write([]byte("test\n"))
	s.NoError(err)

	// 获取内部 rotateWriter
	rw := writer.(*rotateWriter)

	// 修改 currentDay 触发轮转
	rw.mu.Lock()
	rw.currentDay = "2020-01-01"
	rw.mu.Unlock()

	// 再次写入触发轮转
	_, err = writer.Write([]byte("after rotate\n"))
	s.NoError(err)

	// 验证新文件创建
	logDir := filepath.Join(s.tmpDir, "rotate-test")
	files, err := os.ReadDir(logDir)
	s.NoError(err)
	s.NotEmpty(files)
}

func (s *RotateWriterTestSuite) TestCleanupOldLogs() {
	writer := NewRotateWriter(s.tmpDir, "cleanup-test", WithMaxAge(1))
	defer writer.Close()

	rw := writer.(*rotateWriter)

	// 创建旧日志文件
	logDir := filepath.Join(s.tmpDir, "cleanup-test")
	err := os.MkdirAll(logDir, 0o755)
	s.Require().NoError(err)

	// 创建一个"旧"日志文件
	oldLogFile := filepath.Join(logDir, "cleanup-test_2020-01-01.log")
	err = os.WriteFile(oldLogFile, []byte("old log"), 0o644)
	s.Require().NoError(err)

	// 修改文件时间为很久以前
	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	err = os.Chtimes(oldLogFile, oldTime, oldTime)
	s.Require().NoError(err)

	// 执行清理
	rw.cleanupOldLogs()

	// 验证旧文件被删除
	_, err = os.Stat(oldLogFile)
	s.True(os.IsNotExist(err), "old log file should be deleted")
}

func (s *RotateWriterTestSuite) TestCleanupOldLogs_WithCompress() {
	writer := NewRotateWriter(s.tmpDir, "compress-test", WithMaxAge(1), WithCompress(true))
	defer writer.Close()

	rw := writer.(*rotateWriter)

	// 创建日志目录
	logDir := filepath.Join(s.tmpDir, "compress-test")
	err := os.MkdirAll(logDir, 0o755)
	s.Require().NoError(err)

	// 创建一个"旧"日志文件
	oldLogFile := filepath.Join(logDir, "compress-test_2020-01-01.log")
	err = os.WriteFile(oldLogFile, []byte("old log content"), 0o644)
	s.Require().NoError(err)

	// 修改文件时间为超过 maxAge
	oldTime := time.Now().Add(-2 * 24 * time.Hour)
	err = os.Chtimes(oldLogFile, oldTime, oldTime)
	s.Require().NoError(err)

	// 执行清理（应该压缩文件）
	rw.cleanupOldLogs()

	// 验证压缩文件被创建
	_, err = os.Stat(oldLogFile + ".gz")
	s.NoError(err, "compressed file should exist")

	// 验证原文件被删除
	_, err = os.Stat(oldLogFile)
	s.True(os.IsNotExist(err), "original file should be deleted after compression")
}

func (s *RotateWriterTestSuite) TestCleanupOldLogs_SkipDirectories() {
	writer := NewRotateWriter(s.tmpDir, "skipdir-test", WithMaxAge(1))
	defer writer.Close()

	rw := writer.(*rotateWriter)

	// 创建日志目录
	logDir := filepath.Join(s.tmpDir, "skipdir-test")
	err := os.MkdirAll(logDir, 0o755)
	s.Require().NoError(err)

	// 创建子目录（应该被跳过）
	subDir := filepath.Join(logDir, "skipdir-test_2020-01-01.log")
	err = os.MkdirAll(subDir, 0o755)
	s.Require().NoError(err)

	// 执行清理（不应该报错）
	s.NotPanics(func() {
		rw.cleanupOldLogs()
	})

	// 子目录应该仍然存在
	_, err = os.Stat(subDir)
	s.NoError(err, "subdirectory should still exist")
}

func (s *RotateWriterTestSuite) TestCleanupOldLogs_NoMaxAge() {
	writer := NewRotateWriter(s.tmpDir, "nomaxage-test")
	defer writer.Close()

	rw := writer.(*rotateWriter)

	// 不设置 maxAge，cleanupOldLogs 应该直接返回
	s.NotPanics(func() {
		rw.cleanupOldLogs()
	})
}

func (s *RotateWriterTestSuite) TestCleanupOldLogs_NonExistentDir() {
	rw := &rotateWriter{
		baseDir: "/nonexistent/path",
		prefix:  "test",
		maxAge:  24 * time.Hour,
	}

	// 不应该 panic
	s.NotPanics(func() {
		rw.cleanupOldLogs()
	})
}

func (s *RotateWriterTestSuite) TestCompressFile() {
	writer := NewRotateWriter(s.tmpDir, "compress-func-test")
	defer writer.Close()

	rw := writer.(*rotateWriter)

	// 创建测试文件
	testFile := filepath.Join(s.tmpDir, "test-compress.log")
	testContent := "test content for compression"
	err := os.WriteFile(testFile, []byte(testContent), 0o644)
	s.Require().NoError(err)

	// 压缩文件
	rw.compressFile(testFile)

	// 验证压缩文件存在
	_, err = os.Stat(testFile + ".gz")
	s.NoError(err, "compressed file should exist")

	// 验证原文件被删除
	_, err = os.Stat(testFile)
	s.True(os.IsNotExist(err), "original file should be deleted")
}

func (s *RotateWriterTestSuite) TestCompressFile_NonExistent() {
	rw := &rotateWriter{}

	// 压缩不存在的文件不应该 panic
	s.NotPanics(func() {
		rw.compressFile("/nonexistent/file.log")
	})
}

func (s *RotateWriterTestSuite) TestShouldRotate_Hourly() {
	writer := NewRotateWriter(s.tmpDir, "hourly-rotate", WithRotationMode(RotationHourly))
	defer writer.Close()

	rw := writer.(*rotateWriter)

	// 写入数据创建文件
	_, err := writer.Write([]byte("test\n"))
	s.NoError(err)

	// 不应该立即需要轮转
	rw.mu.Lock()
	shouldRotate := rw.shouldRotate()
	rw.mu.Unlock()
	s.False(shouldRotate)
}

func (s *RotateWriterTestSuite) TestShouldRotate_DayChange() {
	writer := NewRotateWriter(s.tmpDir, "day-rotate")
	defer writer.Close()

	rw := writer.(*rotateWriter)

	// 修改 currentDay 为昨天
	rw.mu.Lock()
	rw.currentDay = time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	shouldRotate := rw.shouldRotate()
	rw.mu.Unlock()

	s.True(shouldRotate)
}

func (s *RotateWriterTestSuite) TestBuildFilename() {
	rw := &rotateWriter{
		baseDir:      s.tmpDir,
		prefix:       "test",
		currentDay:   "2024-01-15",
		rotationMode: RotationDaily,
	}

	filename := rw.buildFilename()
	s.Contains(filename, "test_2024-01-15.log")

	rw.rotationMode = RotationHourly
	filename = rw.buildFilename()
	s.Contains(filename, "test_2024-01-15_")
	s.Contains(filename, ".log")
}

// SyncWriterTestSuite 同步写入器测试套件.
type SyncWriterTestSuite struct {
	suite.Suite
	tmpDir string
}

func TestSyncWriterSuite(t *testing.T) {
	suite.Run(t, new(SyncWriterTestSuite))
}

func (s *SyncWriterTestSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
}

func (s *SyncWriterTestSuite) TestSyncWriter() {
	file, err := os.CreateTemp(s.tmpDir, "test*.log")
	s.Require().NoError(err)
	defer os.Remove(file.Name())
	defer file.Close()

	sw := newSyncWriter(file)

	data := []byte("test message\n")
	n, err := sw.Write(data)
	s.NoError(err)
	s.Equal(len(data), n)

	err = sw.Sync()
	s.NoError(err)

	err = sw.Close()
	s.NoError(err)
}

func (s *SyncWriterTestSuite) TestSyncWriter_NonSyncable() {
	sw := newSyncWriter(&nonSyncableWriter{})

	_, err := sw.Write([]byte("test"))
	s.NoError(err)

	// Sync 应该返回 nil（writer 不支持 Sync）
	err = sw.Sync()
	s.NoError(err)

	// Close 应该返回 nil（writer 不支持 Close）
	err = sw.Close()
	s.NoError(err)
}

// nonSyncableWriter 用于测试的不支持 Sync 的 writer.
type nonSyncableWriter struct {
	data []byte
}

func (w *nonSyncableWriter) Write(p []byte) (n int, err error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

// HelperFunctionTestSuite 辅助函数测试套件.
type HelperFunctionTestSuite struct {
	suite.Suite
}

func TestHelperFunctionSuite(t *testing.T) {
	suite.Run(t, new(HelperFunctionTestSuite))
}

func (s *HelperFunctionTestSuite) TestIsCompressedFile() {
	testCases := []struct {
		filename string
		want     bool
	}{
		{"app.log", false},
		{"app.log.gz", true},
		{"app.gz", true},
		{"app.tar.gz", true},
		{"app", false},
		{".gz", true},
	}

	for _, tc := range testCases {
		s.Equal(tc.want, isCompressedFile(tc.filename), "filename: %s", tc.filename)
	}
}
