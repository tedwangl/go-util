package zap

import (
	"fmt"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Writer interface {
	Close() error
	Debug(skip int, v any, fields ...LogField)
	Error(skip int, v any, fields ...LogField)
	Info(skip int, v any, fields ...LogField)
	Slow(skip int, v any, fields ...LogField)
	Severe(skip int, v any)
	Stack(skip int, v any)
	Stat(skip int, v any, fields ...LogField)
	Alert(v any)
}

type zapWriter struct {
	infoLogger   *zap.Logger
	errorLogger  *zap.Logger
	severeLogger *zap.Logger
	slowLogger   *zap.Logger
	statLogger   *zap.Logger
	stackLogger  *zap.Logger
	alertLogger  *zap.Logger
	sugarInfo    *zap.SugaredLogger
	sugarError   *zap.SugaredLogger
	sugarSevere  *zap.SugaredLogger
	sugarSlow    *zap.SugaredLogger
	sugarStat    *zap.SugaredLogger
	sugarStack   *zap.SugaredLogger
	sugarAlert   *zap.SugaredLogger
	config       LogConf
	stackLimiter *limitedExecutor
}

type atomicWriter struct {
	writer Writer
	lock   sync.RWMutex
}

type multiWriter struct {
	writers []Writer
}

func NewMultiWriter(writers ...Writer) Writer {
	return &multiWriter{
		writers: writers,
	}
}

func (w *multiWriter) Close() error {
	var errs []error
	for _, writer := range w.writers {
		if err := writer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

func (w *multiWriter) Debug(skip int, v any, fields ...LogField) {
	for _, writer := range w.writers {
		writer.Debug(skip, v, fields...)
	}
}

func (w *multiWriter) Error(skip int, v any, fields ...LogField) {
	for _, writer := range w.writers {
		writer.Error(skip, v, fields...)
	}
}

func (w *multiWriter) Info(skip int, v any, fields ...LogField) {
	for _, writer := range w.writers {
		writer.Info(skip, v, fields...)
	}
}

func (w *multiWriter) Slow(skip int, v any, fields ...LogField) {
	for _, writer := range w.writers {
		writer.Slow(skip, v, fields...)
	}
}

func (w *multiWriter) Severe(skip int, v any) {
	for _, writer := range w.writers {
		writer.Severe(skip, v)
	}
}

func (w *multiWriter) Stack(skip int, v any) {
	for _, writer := range w.writers {
		writer.Stack(skip, v)
	}
}

func (w *multiWriter) Stat(skip int, v any, fields ...LogField) {
	for _, writer := range w.writers {
		writer.Stat(skip, v, fields...)
	}
}

func (w *multiWriter) Alert(v any) {
	for _, writer := range w.writers {
		writer.Alert(v)
	}
}

var (
	awriter    = &atomicWriter{}
	setupOnce  sync.Once
	timeFormat = "2006-01-02T15:04:05.000Z07:00"
)

func (w *atomicWriter) Load() Writer {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w.writer
}

func (w *atomicWriter) Store(v Writer) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.writer = v
}

func (w *atomicWriter) Swap(v Writer) Writer {
	w.lock.Lock()
	defer w.lock.Unlock()
	old := w.writer
	w.writer = v
	return old
}

func getWriter() Writer {
	w := awriter.Load()
	if w == nil {
		w = newConsoleWriter(LogConf{})
		awriter.Store(w)
	}
	return w
}

func SetWriter(w Writer) {
	if atomic.LoadUint32(&logLevel) != disableLevel {
		awriter.Store(w)
	}
}

func Reset() Writer {
	return awriter.Swap(nil)
}

func Close() error {
	if w := awriter.Swap(nil); w != nil {
		return w.Close()
	}
	return nil
}

func newConsoleWriter(c LogConf) Writer {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        timestampKey,
		LevelKey:       levelKey,
		NameKey:        "logger",
		CallerKey:      callerKey,
		MessageKey:     contentKey,
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zapcore.DebugLevel,
	)

	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))

	var stackLimiter *limitedExecutor
	if c.StackCooldownMillis > 0 {
		stackLimiter = NewLimitedExecutor(c.StackCooldownMillis)
	}

	return &zapWriter{
		infoLogger:   zapLogger,
		errorLogger:  zapLogger,
		severeLogger: zapLogger,
		slowLogger:   zapLogger,
		statLogger:   zapLogger,
		stackLogger:  zapLogger,
		alertLogger:  zapLogger,
		sugarInfo:    zapLogger.Sugar(),
		sugarError:   zapLogger.Sugar(),
		sugarSevere:  zapLogger.Sugar(),
		sugarSlow:    zapLogger.Sugar(),
		sugarStat:    zapLogger.Sugar(),
		sugarStack:   zapLogger.Sugar(),
		sugarAlert:   zapLogger.Sugar(),
		stackLimiter: stackLimiter,
	}
}

func newFileWriter(c LogConf) (Writer, error) {
	if len(c.Path) == 0 {
		return nil, ErrLogPathNotSet
	}

	// 自定义时间编码器，使用 getTimestamp 函数
	customTimeEncoder := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(getTimestamp())
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        timestampKey,
		LevelKey:       levelKey,
		NameKey:        "logger",
		CallerKey:      callerKey,
		MessageKey:     contentKey,
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if c.Encoding == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	accessFile := path.Join(c.Path, accessFilename)
	errorFile := path.Join(c.Path, errorFilename)
	severeFile := path.Join(c.Path, severeFilename)
	slowFile := path.Join(c.Path, slowFilename)
	statFile := path.Join(c.Path, statFilename)

	accessWriter := createRotateWriter(accessFile, c)
	errorWriter := createRotateWriter(errorFile, c)
	severeWriter := createRotateWriter(severeFile, c)
	slowWriter := createRotateWriter(slowFile, c)
	statWriter := createRotateWriter(statFile, c)

	infoCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(accessWriter),
		zapcore.DebugLevel,
	)

	errorCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(errorWriter),
		zapcore.DebugLevel,
	)

	severeCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(severeWriter),
		zapcore.DebugLevel,
	)

	slowCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(slowWriter),
		zapcore.DebugLevel,
	)

	statCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(statWriter),
		zapcore.DebugLevel,
	)

	infoLogger := zap.New(infoCore, zap.AddCaller(), zap.AddCallerSkip(c.CallerSkip))
	errorLogger := zap.New(errorCore, zap.AddCaller(), zap.AddCallerSkip(c.CallerSkip))
	severeLogger := zap.New(severeCore, zap.AddCaller(), zap.AddCallerSkip(c.CallerSkip))
	slowLogger := zap.New(slowCore, zap.AddCaller(), zap.AddCallerSkip(c.CallerSkip))
	statLogger := zap.New(statCore, zap.AddCaller(), zap.AddCallerSkip(c.CallerSkip))

	var stackLogger *zap.Logger
	var stackLimiter *limitedExecutor
	if c.StackCooldownMillis > 0 {
		stackLimiter = NewLimitedExecutor(c.StackCooldownMillis)
	}
	stackLogger = errorLogger.WithOptions(zap.AddStacktrace(zapcore.ErrorLevel))

	alertLogger := errorLogger

	return &zapWriter{
		infoLogger:   infoLogger,
		errorLogger:  errorLogger,
		severeLogger: severeLogger,
		slowLogger:   slowLogger,
		statLogger:   statLogger,
		stackLogger:  stackLogger,
		alertLogger:  alertLogger,
		sugarInfo:    infoLogger.Sugar(),
		sugarError:   errorLogger.Sugar(),
		sugarSevere:  severeLogger.Sugar(),
		sugarSlow:    slowLogger.Sugar(),
		sugarStat:    statLogger.Sugar(),
		sugarStack:   stackLogger.Sugar(),
		sugarAlert:   alertLogger.Sugar(),
		config:       c,
		stackLimiter: stackLimiter,
	}, nil
}

func (w *zapWriter) Close() error {
	var errs []error
	if err := w.infoLogger.Sync(); err != nil {
		errs = append(errs, err)
	}
	if err := w.errorLogger.Sync(); err != nil {
		errs = append(errs, err)
	}
	if err := w.severeLogger.Sync(); err != nil {
		errs = append(errs, err)
	}
	if err := w.slowLogger.Sync(); err != nil {
		errs = append(errs, err)
	}
	if err := w.statLogger.Sync(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

func (w *zapWriter) Debug(skip int, v any, fields ...LogField) {
	// 添加调用者信息
	caller := getCaller(skip)
	if caller != "" {
		fields = append(fields, LogField{Key: "caller", Value: caller})
	}

	// 处理敏感信息
	if s, ok := v.(Sensitive); ok {
		v = ToObjectMarshaler(s)
	}

	zapFields := toInterfaceSlice(fields...)
	if str, ok := v.(string); ok {
		w.sugarInfo.Debugw(str, zapFields...)
	} else {
		w.sugarInfo.Debugw("", append(zapFields, "value", v)...)
	}
}

func (w *zapWriter) Error(skip int, v any, fields ...LogField) {
	// 添加调用者信息
	caller := getCaller(skip)
	if caller != "" {
		fields = append(fields, LogField{Key: "caller", Value: caller})
	}

	// 处理敏感信息
	if s, ok := v.(Sensitive); ok {
		v = ToObjectMarshaler(s)
	}

	zapFields := toInterfaceSlice(fields...)
	if str, ok := v.(string); ok {
		w.sugarError.Errorw(str, zapFields...)
	} else {
		w.sugarError.Errorw("", append(zapFields, "value", v)...)
	}
}

func (w *zapWriter) Info(skip int, v any, fields ...LogField) {
	// 添加调用者信息
	caller := getCaller(skip)
	if caller != "" {
		fields = append(fields, LogField{Key: "caller", Value: caller})
	}

	// 处理敏感信息
	if s, ok := v.(Sensitive); ok {
		v = ToObjectMarshaler(s)
	}

	zapFields := toInterfaceSlice(fields...)
	if str, ok := v.(string); ok {
		w.sugarInfo.Infow(str, zapFields...)
	} else {
		w.sugarInfo.Infow("", append(zapFields, "value", v)...)
	}
}

func (w *zapWriter) Slow(skip int, v any, fields ...LogField) {
	// 添加调用者信息
	caller := getCaller(skip)
	if caller != "" {
		fields = append(fields, LogField{Key: "caller", Value: caller})
	}

	// 处理敏感信息
	if s, ok := v.(Sensitive); ok {
		v = ToObjectMarshaler(s)
	}

	zapFields := toInterfaceSlice(fields...)
	if str, ok := v.(string); ok {
		w.sugarSlow.Warnw(str, zapFields...)
	} else {
		w.sugarSlow.Warnw("", append(zapFields, "value", v)...)
	}
}

func (w *zapWriter) Severe(skip int, v any) {
	// 添加调用者信息
	caller := getCaller(skip)

	// 处理敏感信息
	if s, ok := v.(Sensitive); ok {
		v = ToObjectMarshaler(s)
	}

	if str, ok := v.(string); ok {
		if caller != "" {
			w.sugarSevere.Errorw(str, "caller", caller)
		} else {
			w.sugarSevere.Errorw(str)
		}
	} else {
		if caller != "" {
			w.sugarSevere.Errorw("", "caller", caller, "value", v)
		} else {
			w.sugarSevere.Errorw("", "value", v)
		}
	}
}

func (w *zapWriter) Stack(skip int, v any) {
	// 添加调用者信息
	caller := getCaller(skip)

	// 处理敏感信息
	if s, ok := v.(Sensitive); ok {
		v = ToObjectMarshaler(s)
	}

	logFunc := func() {
		if str, ok := v.(string); ok {
			if caller != "" {
				w.stackLogger.Error(str, zap.String("caller", caller))
			} else {
				w.stackLogger.Error(str)
			}
		} else {
			if caller != "" {
				w.stackLogger.Error("", zap.String("caller", caller), zap.Any("value", v))
			} else {
				w.stackLogger.Error("", zap.Any("value", v))
			}
		}
	}
	if w.stackLimiter != nil {
		w.stackLimiter.logOrDiscard(logFunc)
	} else {
		logFunc()
	}
}

func (w *zapWriter) Stat(skip int, v any, fields ...LogField) {
	// 添加调用者信息
	caller := getCaller(skip)
	if caller != "" {
		fields = append(fields, LogField{Key: "caller", Value: caller})
	}

	// 处理敏感信息
	if s, ok := v.(Sensitive); ok {
		v = ToObjectMarshaler(s)
	}

	zapFields := toInterfaceSlice(fields...)
	if str, ok := v.(string); ok {
		w.sugarStat.Infow(str, zapFields...)
	} else {
		w.sugarStat.Infow("", append(zapFields, "value", v)...)
	}
}

func (w *zapWriter) Alert(v any) {
	// 处理敏感信息
	if s, ok := v.(Sensitive); ok {
		v = ToObjectMarshaler(s)
	}

	if str, ok := v.(string); ok {
		w.sugarAlert.Errorw(str)
	} else {
		w.sugarAlert.Errorw("", "value", v)
	}
}

func toInterfaceSlice(fields ...LogField) []interface{} {
	result := make([]interface{}, 0, len(fields)*2)
	for _, f := range fields {
		// 处理敏感信息
		if s, ok := f.Value.(Sensitive); ok {
			// 如果是 Sensitive 类型，使用 ToObjectMarshaler 函数来包装它
			result = append(result, f.Key, ToObjectMarshaler(s))
		} else {
			// 其他类型，直接使用
			result = append(result, f.Key, f.Value)
		}
	}
	return result
}

type nopWriter struct{}

func (n nopWriter) Close() error {
	return nil
}

func (n nopWriter) Debug(_ int, _ any, _ ...LogField) {}

func (n nopWriter) Error(_ int, _ any, _ ...LogField) {}

func (n nopWriter) Info(_ int, _ any, _ ...LogField) {}

func (n nopWriter) Slow(_ int, _ any, _ ...LogField) {}

func (n nopWriter) Severe(_ int, _ any) {}

func (n nopWriter) Stack(_ int, _ any) {}

func (n nopWriter) Stat(_ int, _ any, _ ...LogField) {}

func (n nopWriter) Alert(_ any) {}

func Disable() {
	atomic.StoreUint32(&logLevel, disableLevel)
	awriter.Store(nopWriter{})
}
