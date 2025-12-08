// Package logger 提供结构化日志记录功能.
package logger

import (
	"math"
	"strconv"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var bufferPool = buffer.NewPool()

// consoleEncoder 自定义 console 编码器.
// 将额外字段以 key=value 格式输出，而非 JSON.
type consoleEncoder struct {
	zapcore.Encoder
	config zapcore.EncoderConfig
}

// newConsoleEncoder 创建自定义 console 编码器.
func newConsoleEncoder(config zapcore.EncoderConfig) zapcore.Encoder {
	return &consoleEncoder{
		Encoder: zapcore.NewConsoleEncoder(config),
		config:  config,
	}
}

// Clone 克隆编码器.
func (c *consoleEncoder) Clone() zapcore.Encoder {
	return &consoleEncoder{
		Encoder: c.Encoder.Clone(),
		config:  c.config,
	}
}

// EncodeEntry 编码日志条目.
func (c *consoleEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf := bufferPool.Get()
	enc := &fieldEncoder{buf: buf}

	// 编码时间
	if c.config.TimeKey != "" && c.config.EncodeTime != nil {
		c.config.EncodeTime(entry.Time, enc)
		buf.AppendString(c.config.ConsoleSeparator)
	}

	// 编码级别
	if c.config.LevelKey != "" && c.config.EncodeLevel != nil {
		c.config.EncodeLevel(entry.Level, enc)
		buf.AppendString(c.config.ConsoleSeparator)
	}

	// 编码调用者
	if entry.Caller.Defined && c.config.CallerKey != "" && c.config.EncodeCaller != nil {
		c.config.EncodeCaller(entry.Caller, enc)
		buf.AppendString(c.config.ConsoleSeparator)
	}

	// 编码消息
	buf.AppendString(entry.Message)

	// 编码额外字段为 key=value 格式
	if len(fields) > 0 {
		buf.AppendString(c.config.ConsoleSeparator)
		for i, field := range fields {
			if i > 0 {
				buf.AppendByte(' ')
			}
			c.encodeField(buf, field)
		}
	}

	buf.AppendByte('\n')
	return buf, nil
}

// encodeField 将字段编码为 key=value 格式.
func (c *consoleEncoder) encodeField(buf *buffer.Buffer, field zapcore.Field) {
	buf.AppendString(field.Key)
	buf.AppendByte('=')

	switch field.Type {
	case zapcore.StringType:
		buf.AppendString(field.String)

	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		buf.AppendString(strconv.FormatInt(field.Integer, 10))

	case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type, zapcore.UintptrType:
		buf.AppendString(strconv.FormatUint(uint64(field.Integer), 10))

	case zapcore.Float64Type:
		buf.AppendString(strconv.FormatFloat(math.Float64frombits(uint64(field.Integer)), 'g', -1, 64))

	case zapcore.Float32Type:
		buf.AppendString(strconv.FormatFloat(float64(math.Float32frombits(uint32(field.Integer))), 'g', -1, 32))

	case zapcore.BoolType:
		buf.AppendString(strconv.FormatBool(field.Integer == 1))

	case zapcore.DurationType:
		buf.AppendString(time.Duration(field.Integer).String())

	case zapcore.TimeType:
		t := c.decodeTime(field)
		buf.AppendString(t.Format("2006-01-02 15:04:05"))

	case zapcore.TimeFullType:
		if t, ok := field.Interface.(time.Time); ok {
			buf.AppendString(t.Format("2006-01-02 15:04:05"))
		}

	case zapcore.ErrorType:
		if err, ok := field.Interface.(error); ok && err != nil {
			buf.AppendString(err.Error())
		}

	case zapcore.StringerType:
		if s, ok := field.Interface.(interface{ String() string }); ok {
			buf.AppendString(s.String())
		}

	default:
		c.encodeReflected(buf, field)
	}
}

// decodeTime 解码时间字段.
func (c *consoleEncoder) decodeTime(field zapcore.Field) time.Time {
	if field.Interface != nil {
		if loc, ok := field.Interface.(*time.Location); ok {
			return time.Unix(0, field.Integer).In(loc)
		}
	}
	return time.Unix(0, field.Integer)
}

// encodeReflected 编码反射类型.
func (c *consoleEncoder) encodeReflected(buf *buffer.Buffer, field zapcore.Field) {
	if field.Interface != nil {
		if s, ok := field.Interface.(interface{ String() string }); ok {
			buf.AppendString(s.String())
			return
		}
		// 使用简单的类型断言处理常见类型
		switch v := field.Interface.(type) {
		case string:
			buf.AppendString(v)
		case int:
			buf.AppendString(strconv.Itoa(v))
		case int64:
			buf.AppendString(strconv.FormatInt(v, 10))
		case float64:
			buf.AppendString(strconv.FormatFloat(v, 'g', -1, 64))
		case bool:
			buf.AppendString(strconv.FormatBool(v))
		default:
			buf.AppendString("<complex>")
		}
	} else if field.String != "" {
		buf.AppendString(field.String)
	}
}

// fieldEncoder 用于编码单个值到 buffer.
type fieldEncoder struct {
	buf *buffer.Buffer
}

// AppendString 追加字符串.
func (e *fieldEncoder) AppendString(v string) {
	e.buf.AppendString(v)
}

// AppendBool 追加布尔值.
func (e *fieldEncoder) AppendBool(v bool) {
	e.buf.AppendString(strconv.FormatBool(v))
}

// AppendByteString 追加字节字符串.
func (e *fieldEncoder) AppendByteString(v []byte) {
	e.buf.AppendString(string(v))
}

// AppendInt 追加整数.
func (e *fieldEncoder) AppendInt(v int) {
	e.buf.AppendString(strconv.Itoa(v))
}

// AppendInt64 追加 int64.
func (e *fieldEncoder) AppendInt64(v int64) {
	e.buf.AppendString(strconv.FormatInt(v, 10))
}

// AppendInt32 追加 int32.
func (e *fieldEncoder) AppendInt32(v int32) {
	e.buf.AppendString(strconv.FormatInt(int64(v), 10))
}

// AppendInt16 追加 int16.
func (e *fieldEncoder) AppendInt16(v int16) {
	e.buf.AppendString(strconv.FormatInt(int64(v), 10))
}

// AppendInt8 追加 int8.
func (e *fieldEncoder) AppendInt8(v int8) {
	e.buf.AppendString(strconv.FormatInt(int64(v), 10))
}

// AppendUint 追加无符号整数.
func (e *fieldEncoder) AppendUint(v uint) {
	e.buf.AppendString(strconv.FormatUint(uint64(v), 10))
}

// AppendUint64 追加 uint64.
func (e *fieldEncoder) AppendUint64(v uint64) {
	e.buf.AppendString(strconv.FormatUint(v, 10))
}

// AppendUint32 追加 uint32.
func (e *fieldEncoder) AppendUint32(v uint32) {
	e.buf.AppendString(strconv.FormatUint(uint64(v), 10))
}

// AppendUint16 追加 uint16.
func (e *fieldEncoder) AppendUint16(v uint16) {
	e.buf.AppendString(strconv.FormatUint(uint64(v), 10))
}

// AppendUint8 追加 uint8.
func (e *fieldEncoder) AppendUint8(v uint8) {
	e.buf.AppendString(strconv.FormatUint(uint64(v), 10))
}

// AppendUintptr 追加 uintptr.
func (e *fieldEncoder) AppendUintptr(v uintptr) {
	e.buf.AppendString(strconv.FormatUint(uint64(v), 10))
}

// AppendFloat64 追加 float64.
func (e *fieldEncoder) AppendFloat64(v float64) {
	e.buf.AppendString(strconv.FormatFloat(v, 'g', -1, 64))
}

// AppendFloat32 追加 float32.
func (e *fieldEncoder) AppendFloat32(v float32) {
	e.buf.AppendString(strconv.FormatFloat(float64(v), 'g', -1, 32))
}

// AppendComplex128 追加 complex128（忽略）.
func (e *fieldEncoder) AppendComplex128(_ complex128) {}

// AppendComplex64 追加 complex64（忽略）.
func (e *fieldEncoder) AppendComplex64(_ complex64) {}

// AppendDuration 追加时间间隔.
func (e *fieldEncoder) AppendDuration(v time.Duration) {
	e.buf.AppendString(v.String())
}

// AppendTime 追加时间.
func (e *fieldEncoder) AppendTime(v time.Time) {
	e.buf.AppendString(v.Format("2006-01-02 15:04:05"))
}

// AppendArray 追加数组（忽略）.
func (e *fieldEncoder) AppendArray(_ zapcore.ArrayMarshaler) error {
	return nil
}

// AppendObject 追加对象（忽略）.
func (e *fieldEncoder) AppendObject(_ zapcore.ObjectMarshaler) error {
	return nil
}

// AppendReflected 追加反射值.
func (e *fieldEncoder) AppendReflected(v interface{}) error {
	if s, ok := v.(interface{ String() string }); ok {
		e.buf.AppendString(s.String())
	} else {
		e.buf.AppendString("<reflected>")
	}
	return nil
}
