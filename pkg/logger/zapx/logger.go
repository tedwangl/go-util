package zap

import (
	"context"
	"fmt"
	"time"
)

type Logger interface {
	Debug(...any)
	Debugf(string, ...any)
	Debugfn(func() any)
	Debugv(any)
	Debugw(string, ...LogField)
	Error(...any)
	Errorf(string, ...any)
	Errorfn(func() any)
	Errorv(any)
	Errorw(string, ...LogField)
	Info(...any)
	Infof(string, ...any)
	Infofn(func() any)
	Infov(any)
	Infow(string, ...LogField)
	Slow(...any)
	Slowf(string, ...any)
	Slowfn(func() any)
	Slowv(any)
	Sloww(string, ...LogField)
	WithCallerSkip(int) Logger
	WithContext(context.Context) Logger
	WithDuration(time.Duration) Logger
	WithFields(...LogField) Logger
}

type baseLogger struct {
	writer Writer
	ctx    context.Context
	fields []LogField
	skip   int
}

func newLogger(writer Writer) Logger {
	return &baseLogger{
		writer: writer,
		skip:   2,
	}
}

func (l *baseLogger) Debug(v ...any) {
	if shallLog(DebugLevel) {
		l.writer.Debug(l.skip, fmt.Sprint(v...), mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Debugf(format string, v ...any) {
	if shallLog(DebugLevel) {
		l.writer.Debug(l.skip, fmt.Sprintf(format, v...), mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Debugfn(fn func() any) {
	if shallLog(DebugLevel) {
		l.writer.Debug(l.skip, fn(), mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Debugv(v any) {
	if shallLog(DebugLevel) {
		l.writer.Debug(l.skip, v, mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Debugw(msg string, fields ...LogField) {
	if shallLog(DebugLevel) {
		allFields := mergeFields(l.ctx, l.fields...)
		allFields = append(allFields, Field(contentKey, msg))
		allFields = append(allFields, fields...)
		l.writer.Debug(l.skip, "", allFields...)
	}
}

func (l *baseLogger) Error(v ...any) {
	if shallLog(ErrorLevel) {
		l.writer.Error(l.skip, fmt.Sprint(v...), mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Errorf(format string, v ...any) {
	if shallLog(ErrorLevel) {
		l.writer.Error(l.skip, fmt.Errorf(format, v...).Error(), mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Errorfn(fn func() any) {
	if shallLog(ErrorLevel) {
		l.writer.Error(l.skip, fn(), mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Errorv(v any) {
	if shallLog(ErrorLevel) {
		l.writer.Error(l.skip, v, mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Errorw(msg string, fields ...LogField) {
	if shallLog(ErrorLevel) {
		allFields := mergeFields(l.ctx, l.fields...)
		allFields = append(allFields, Field(contentKey, msg))
		allFields = append(allFields, fields...)
		l.writer.Error(l.skip, "", allFields...)
	}
}

func (l *baseLogger) Info(v ...any) {
	if shallLog(InfoLevel) {
		l.writer.Info(l.skip, fmt.Sprint(v...), mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Infof(format string, v ...any) {
	if shallLog(InfoLevel) {
		l.writer.Info(l.skip, fmt.Sprintf(format, v...), mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Infofn(fn func() any) {
	if shallLog(InfoLevel) {
		l.writer.Info(l.skip, fn(), mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Infov(v any) {
	if shallLog(InfoLevel) {
		l.writer.Info(l.skip, v, mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Infow(msg string, fields ...LogField) {
	if shallLog(InfoLevel) {
		allFields := mergeFields(l.ctx, l.fields...)
		allFields = append(allFields, Field(contentKey, msg))
		allFields = append(allFields, fields...)
		l.writer.Info(l.skip, "", allFields...)
	}
}

func (l *baseLogger) Slow(v ...any) {
	if shallLog(ErrorLevel) {
		l.writer.Slow(l.skip, fmt.Sprint(v...), mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Slowf(format string, v ...any) {
	if shallLog(ErrorLevel) {
		l.writer.Slow(l.skip, fmt.Sprintf(format, v...), mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Slowfn(fn func() any) {
	if shallLog(ErrorLevel) {
		l.writer.Slow(l.skip, fn(), mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Slowv(v any) {
	if shallLog(ErrorLevel) {
		l.writer.Slow(l.skip, v, mergeFields(l.ctx, l.fields...)...)
	}
}

func (l *baseLogger) Sloww(msg string, fields ...LogField) {
	if shallLog(ErrorLevel) {
		allFields := mergeFields(l.ctx, l.fields...)
		allFields = append(allFields, Field(contentKey, msg))
		allFields = append(allFields, fields...)
		l.writer.Slow(l.skip, "", allFields...)
	}
}

func (l *baseLogger) WithCallerSkip(skip int) Logger {
	return &baseLogger{
		writer: l.writer,
		ctx:    l.ctx,
		fields: l.fields,
		skip:   l.skip + skip,
	}
}

func (l *baseLogger) WithContext(ctx context.Context) Logger {
	return &baseLogger{
		writer: l.writer,
		ctx:    ctx,
		fields: l.fields,
		skip:   l.skip,
	}
}

func (l *baseLogger) WithDuration(d time.Duration) Logger {
	newFields := make([]LogField, len(l.fields), len(l.fields)+1)
	copy(newFields, l.fields)
	newFields = append(newFields, Field(durationKey, d))
	return &baseLogger{
		writer: l.writer,
		ctx:    l.ctx,
		fields: newFields,
		skip:   l.skip,
	}
}

func (l *baseLogger) WithFields(fields ...LogField) Logger {
	newFields := make([]LogField, len(l.fields), len(l.fields)+len(fields))
	copy(newFields, l.fields)
	newFields = append(newFields, fields...)
	return &baseLogger{
		writer: l.writer,
		ctx:    l.ctx,
		fields: newFields,
		skip:   l.skip,
	}
}
