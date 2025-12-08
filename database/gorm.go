package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/Tsukikage7/microservice-kit/logger"
)

// BaseModel GORM 基础模型.
type BaseModel[T any] struct {
	ID          T              `gorm:"primaryKey"`
	CreatedTime time.Time      `gorm:"column:created_time;autoCreateTime"`
	UpdatedTime time.Time      `gorm:"column:updated_time;autoUpdateTime"`
	DeletedTime gorm.DeletedAt `gorm:"column:deleted_time;index"`
}

// gormDatabase GORM 数据库实现.
type gormDatabase struct {
	db     *gorm.DB
	config *Config
	logger logger.Logger
}

// newGORMDatabase 创建 GORM 数据库连接.
func newGORMDatabase(config *Config, log logger.Logger) (Database, error) {
	dialector, err := getDialector(config.Driver, config.DSN)
	if err != nil {
		return nil, err
	}

	gormConfig := &gorm.Config{
		Logger: newGORMLoggerAdapter(log, config.SlowThreshold, config.LogLevel),
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, err
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(config.Pool.MaxOpen)
	sqlDB.SetMaxIdleConns(config.Pool.MaxIdle)
	sqlDB.SetConnMaxLifetime(config.Pool.MaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.Pool.MaxIdleTime)

	return &gormDatabase{
		db:     db,
		config: config,
		logger: log,
	}, nil
}

// getDialector 根据驱动类型返回对应的 Dialector.
func getDialector(driver, dsn string) (gorm.Dialector, error) {
	switch driver {
	case DriverMySQL:
		return mysql.Open(dsn), nil
	case DriverPostgres, DriverPostgreSQL:
		return postgres.Open(dsn), nil
	case DriverSQLite, DriverSQLite3:
		return sqlite.Open(dsn), nil
	default:
		return nil, ErrUnsupportedDriver
	}
}

// DB 获取 GORM 数据库实例.
func (g *gormDatabase) DB() any {
	return g.db
}

// GORM 获取类型安全的 GORM 实例.
func (g *gormDatabase) GORM() *gorm.DB {
	return g.db
}

// AutoMigrate 自动迁移表结构.
func (g *gormDatabase) AutoMigrate(models ...any) error {
	if !g.config.AutoMigrate {
		g.logger.Debug("[Database] 自动迁移已禁用，跳过表结构创建")
		return nil
	}

	g.logger.Debug("[Database] 开始自动迁移表结构")
	if err := g.db.AutoMigrate(models...); err != nil {
		g.logger.Error("[Database] 自动迁移失败", "error", err)
		return err
	}
	g.logger.Debug("[Database] 表结构迁移完成")
	return nil
}

// Close 关闭数据库连接.
func (g *gormDatabase) Close() error {
	sqlDB, err := g.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GORMDatabase GORM 数据库扩展接口.
type GORMDatabase interface {
	Database
	GORM() *gorm.DB
}


// AsGORM 将 Database 转换为 GORMDatabase.
func AsGORM(db Database) *gorm.DB {
	if gdb, ok := db.(*gormDatabase); ok {
		return gdb.db
	}
	panic("database: 无法提取 *gorm.DB，请确保使用 GORM 类型的数据库")
}

// gormLoggerAdapter GORM 日志适配器.
type gormLoggerAdapter struct {
	logger        logger.Logger
	slowThreshold time.Duration
	logLevel      gormlogger.LogLevel
}

// newGORMLoggerAdapter 创建 GORM 日志适配器.
func newGORMLoggerAdapter(log logger.Logger, slowThreshold time.Duration, level string) gormlogger.Interface {
	logLevel := gormlogger.Info
	switch level {
	case "silent":
		logLevel = gormlogger.Silent
	case "error":
		logLevel = gormlogger.Error
	case "warn":
		logLevel = gormlogger.Warn
	case "info":
		logLevel = gormlogger.Info
	}

	return &gormLoggerAdapter{
		logger:        log,
		slowThreshold: slowThreshold,
		logLevel:      logLevel,
	}
}

// LogMode 设置日志模式.
func (l *gormLoggerAdapter) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

// Info 信息日志.
func (l *gormLoggerAdapter) Info(_ context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Info {
		l.logger.Infof(msg, data...)
	}
}

// Warn 警告日志.
func (l *gormLoggerAdapter) Warn(_ context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Warn {
		l.logger.Warnf(msg, data...)
	}
}

// Error 错误日志.
func (l *gormLoggerAdapter) Error(_ context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Error {
		l.logger.Errorf(msg, data...)
	}
}

// Trace SQL 跟踪日志.
func (l *gormLoggerAdapter) Trace(_ context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.logLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		l.logger.Error(fmt.Sprintf("[Database] SQL 执行失败 | 错误=%v | 耗时=%v | 行数=%d | SQL=%s",
			err, elapsed, rows, sql))
	case elapsed > l.slowThreshold && l.slowThreshold > 0:
		l.logger.Warn(fmt.Sprintf("[Database] 慢查询 | 耗时=%v | 阈值=%v | 行数=%d | SQL=%s",
			elapsed, l.slowThreshold, rows, sql))
	default:
		l.logger.Debug(fmt.Sprintf("[Database] SQL 执行成功 | 耗时=%v | 行数=%d | SQL=%s",
			elapsed, rows, sql))
	}
}
