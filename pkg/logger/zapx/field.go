package zap

import (
	"context"
	"sync"
	"sync/atomic"
)

type (
	LogField struct {
		Key   string
		Value any
	}

	fieldsKey struct{}
)

var (
	globalFields     atomic.Value
	globalFieldsLock sync.Mutex
)

func Field(key string, value any) LogField {
	return LogField{
		Key:   key,
		Value: value,
	}
}

func AddGlobalFields(fields ...LogField) {
	globalFieldsLock.Lock()
	defer globalFieldsLock.Unlock()

	old := globalFields.Load()
	if old == nil {
		globalFields.Store(append([]LogField(nil), fields...))
	} else {
		globalFields.Store(append(old.([]LogField), fields...))
	}
}

func ContextWithFields(ctx context.Context, fields ...LogField) context.Context {
	if val := ctx.Value(fieldsKey{}); val != nil {
		if arr, ok := val.([]LogField); ok {
			allFields := make([]LogField, 0, len(arr)+len(fields))
			allFields = append(allFields, arr...)
			allFields = append(allFields, fields...)
			return context.WithValue(ctx, fieldsKey{}, allFields)
		}
	}

	return context.WithValue(ctx, fieldsKey{}, fields)
}

func getGlobalFields() []LogField {
	globals := globalFields.Load()
	if globals == nil {
		return nil
	}
	return globals.([]LogField)
}

func getContextFields(ctx context.Context) []LogField {
	if ctx == nil {
		return nil
	}
	if val := ctx.Value(fieldsKey{}); val != nil {
		return val.([]LogField)
	}
	return nil
}

func mergeFields(ctx context.Context, fields ...LogField) []LogField {
	globals := getGlobalFields()
	contextFields := getContextFields(ctx)

	totalLen := len(fields)
	if globals != nil {
		totalLen += len(globals)
	}
	if contextFields != nil {
		totalLen += len(contextFields)
	}

	if totalLen == 0 {
		return fields
	}

	result := make([]LogField, 0, totalLen)
	if globals != nil {
		result = append(result, globals...)
	}
	if contextFields != nil {
		result = append(result, contextFields...)
	}
	result = append(result, fields...)

	return result
}
