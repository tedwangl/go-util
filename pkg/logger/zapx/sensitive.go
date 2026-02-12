package zapx

import (
	"go.uber.org/zap/zapcore"
)

// Sensitive 接口用于标记敏感信息，实现该接口的对象在日志中会被脱敏处理
type Sensitive interface {
	// MaskSensitive 返回脱敏后的对象
	MaskSensitive() any
}

// sensitiveMarshaler 是一个适配器，用于将 Sensitive 接口转换为 zapcore.ObjectMarshaler 接口
type sensitiveMarshaler struct {
	sensitive Sensitive
}

// MarshalLogObject 实现了 zapcore.ObjectMarshaler 接口
func (sm *sensitiveMarshaler) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	// 调用 MaskSensitive 方法获取脱敏后的对象
	masked := sm.sensitive.MaskSensitive()

	// 根据脱敏后的对象类型，调用不同的编码方法
	switch v := masked.(type) {
	case map[string]any:
		// 如果是 map[string]any 类型，遍历所有键值对并编码
		for k, val := range v {
			enc.AddReflected(k, val)
		}
	case zapcore.ObjectMarshaler:
		// 如果是 ObjectMarshaler 类型，直接调用其 MarshalLogObject 方法
		return v.MarshalLogObject(enc)
	default:
		// 其他类型，使用默认的编码方式
		enc.AddReflected("value", v)
	}

	return nil
}

// ToObjectMarshaler 将 Sensitive 接口转换为 zapcore.ObjectMarshaler 接口
func ToObjectMarshaler(s Sensitive) zapcore.ObjectMarshaler {
	return &sensitiveMarshaler{sensitive: s}
}
