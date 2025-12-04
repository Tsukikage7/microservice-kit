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
