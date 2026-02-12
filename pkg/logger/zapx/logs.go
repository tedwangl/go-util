package zapx

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"runtime/debug"
	"sync/atomic"
	"time"
)

const callerDepth = 4

func processSensitiveArgs(v ...any) []any {
	processedArgs := make([]any, len(v))
	for i, arg := range v {
		if s, ok := arg.(Sensitive); ok {
			processedArgs[i] = s.MaskSensitive()
		} else {
			processedArgs[i] = arg
		}
	}
	return processedArgs
}

func processSensitiveFields(fields ...LogField) []LogField {
	processedFields := make([]LogField, len(fields))
	for i, field := range fields {
		if s, ok := field.Value.(Sensitive); ok {
			processedFields[i] = LogField{
				Key:   field.Key,
				Value: s.MaskSensitive(),
			}
		} else {
			processedFields[i] = field
		}
	}
	return processedFields
}

func logWithSensitiveHandling(
	level string,
	writerFunc func(skip int, v any, fields ...LogField),
	callerSkip int,
	v ...any,
) {
	if len(v) == 0 {
		writerFunc(callerSkip, "")
		return
	}

	// 如果第一个参数是字符串，并且其余参数都是 LogField 类型，则按特定方式处理
	if msg, ok := v[0].(string); ok && len(v) > 1 {
		var fields []LogField
		isAllFields := true

		for i := 1; i < len(v); i++ {
			if field, ok := v[i].(LogField); ok {
				fields = append(fields, field)
			} else if fieldSlice, ok := v[i].([]LogField); ok {
				fields = append(fields, fieldSlice...)
			} else {
				isAllFields = false
				break
			}
		}

		if isAllFields {
			writerFunc(callerSkip, msg, fields...)
			return
		}
	}

	// 默认行为：将所有参数格式化为字符串
	formatted := fmt.Sprint(v...)

	// 检查是否只有一个参数且该参数是 Sensitive 类型
	if len(v) == 1 {
		if s, ok := v[0].(Sensitive); ok {
			// 如果是敏感信息，直接传递给 writer，让 writer 处理
			writerFunc(callerSkip, s, Field("formatted", formatted))
			return
		}
	}

	writerFunc(callerSkip, formatted)
}

func logWithSensitiveHandlingSimple(
	level string,
	writerFunc func(skip int, v any),
	callerSkip int,
	v ...any,
) {
	if len(v) == 0 {
		writerFunc(callerSkip, "")
		return
	}

	// 检查是否有字段参数（即包含LogField的情况）
	hasLogField := false
	for _, arg := range v {
		if _, ok := arg.(LogField); ok {
			hasLogField = true
			break
		}
	}

	if hasLogField {
		// 如果有LogField参数，提取并处理其中的敏感信息
		var message string
		var fields []LogField

		for _, arg := range v {
			if field, ok := arg.(LogField); ok {
				// 处理字段中的敏感信息
				if s, ok := field.Value.(Sensitive); ok {
					fields = append(fields, LogField{
						Key:   field.Key,
						Value: s.MaskSensitive(),
					})
				} else {
					fields = append(fields, field)
				}
			} else {
				// 非字段参数作为消息部分
				if msg, ok := arg.(string); ok {
					message += msg
				} else {
					message += fmt.Sprint(arg)
				}
			}
		}

		// 将消息和字段组合起来
		if len(fields) > 0 {
			// 创建一个包含所有字段的映射，并将其格式化为字符串
			fieldMap := make(map[string]any)
			for _, field := range fields {
				fieldMap[field.Key] = field.Value
			}

			if message != "" {
				writerFunc(callerSkip, message+fmt.Sprintf("%v", fieldMap))
			} else {
				writerFunc(callerSkip, fieldMap)
			}
		} else {
			writerFunc(callerSkip, message)
		}
	} else {
		// 处理敏感信息
		processedArgs := processSensitiveArgs(v...)

		// 检查是否只有一个参数且该参数是 Sensitive 类型
		if len(processedArgs) == 1 {
			if s, ok := processedArgs[0].(Sensitive); ok {
				// 如果是敏感信息，直接传递给 writer，让 writer 处理
				writerFunc(callerSkip, s)
				return
			}
		}

		// 默认行为：将所有参数格式化为字符串
		formatted := fmt.Sprint(processedArgs...)

		writerFunc(callerSkip, formatted)
	}
}

func Alert(v string) {
	getWriter().Alert(v)
}

func Debug(v ...any) {
	if shallLog(DebugLevel) {
		logWithSensitiveHandling("debug", getWriter().Debug, callerDepth, v...)
	}
}

func Debugf(format string, v ...any) {
	if shallLog(DebugLevel) {
		processedArgs := processSensitiveArgs(v...)
		getWriter().Debug(callerDepth, fmt.Sprintf(format, processedArgs...))
	}
}

func Debugfn(fn func() any) {
	if shallLog(DebugLevel) {
		getWriter().Debug(callerDepth, fn())
	}
}

func Debugv(v any) {
	if shallLog(DebugLevel) {
		getWriter().Debug(callerDepth, v)
	}
}

func Debugw(msg string, fields ...LogField) {
	if shallLog(DebugLevel) {
		processedFields := processSensitiveFields(fields...)
		getWriter().Debug(callerDepth, msg, processedFields...)
	}
}

func Error(v ...any) {
	if shallLog(ErrorLevel) {
		logWithSensitiveHandling("error", getWriter().Error, callerDepth, v...)
	}
}

func Errorf(format string, v ...any) {
	if shallLog(ErrorLevel) {
		processedArgs := processSensitiveArgs(v...)
		getWriter().Error(callerDepth, fmt.Sprintf(format, processedArgs...))
	}
}

func Errorfn(fn func() any) {
	if shallLog(ErrorLevel) {
		getWriter().Error(callerDepth, fn())
	}
}

func ErrorStack(v ...any) {
	if shallLog(ErrorLevel) {
		logWithSensitiveHandlingSimple("error_stack", getWriter().Stack, callerDepth, v...)
	}
}

func ErrorStackf(format string, v ...any) {
	if shallLog(ErrorLevel) {
		processedArgs := processSensitiveArgs(v...)
		getWriter().Stack(callerDepth, fmt.Sprintf(format, processedArgs...))
	}
}

func Errorv(v any) {
	if shallLog(ErrorLevel) {
		getWriter().Error(callerDepth, v)
	}
}

func Errorw(msg string, fields ...LogField) {
	if shallLog(ErrorLevel) {
		processedFields := processSensitiveFields(fields...)
		getWriter().Error(callerDepth, msg, processedFields...)
	}
}

func Info(v ...any) {
	if shallLog(InfoLevel) {
		logWithSensitiveHandling("info", getWriter().Info, callerDepth, v...)
	}
}

func Infof(format string, v ...any) {
	if shallLog(InfoLevel) {
		processedArgs := processSensitiveArgs(v...)
		getWriter().Info(callerDepth, fmt.Sprintf(format, processedArgs...))
	}
}

func Infofn(fn func() any) {
	if shallLog(InfoLevel) {
		getWriter().Info(callerDepth, fn())
	}
}

func Infov(v any) {
	if shallLog(InfoLevel) {
		getWriter().Info(callerDepth, v)
	}
}

func Infow(msg string, fields ...LogField) {
	if shallLog(InfoLevel) {
		processedFields := processSensitiveFields(fields...)
		getWriter().Info(callerDepth, msg, processedFields...)
	}
}

func Must(err error) {
	if err == nil {
		return
	}

	msg := fmt.Sprintf("%+v\n\n%s", err.Error(), debug.Stack())
	log.Print(msg)
	getWriter().Alert(msg)

	if ExitOnFatal {
		os.Exit(1)
	} else {
		panic(msg)
	}
}

func MustSetup(c LogConf) {
	Must(SetUp(c))
}

func Severe(v ...any) {
	if shallLog(SevereLevel) {
		logWithSensitiveHandlingSimple("severe", getWriter().Severe, callerDepth, v...)
	}
}

func Severef(format string, v ...any) {
	if shallLog(SevereLevel) {
		processedArgs := processSensitiveArgs(v...)
		getWriter().Severe(callerDepth, fmt.Sprintf(format, processedArgs...))
	}
}

func Slow(v ...any) {
	if shallLog(ErrorLevel) {
		logWithSensitiveHandling("slow", getWriter().Slow, callerDepth, v...)
	}
}

func Slowfn(fn func() any) {
	if shallLog(ErrorLevel) {
		getWriter().Slow(callerDepth, fn())
	}
}

func Slowv(v any) {
	if shallLog(ErrorLevel) {
		getWriter().Slow(callerDepth, v)
	}
}

func Sloww(msg string, fields ...LogField) {
	if shallLog(ErrorLevel) {
		processedFields := processSensitiveFields(fields...)
		getWriter().Slow(callerDepth, msg, processedFields...)
	}
}

func Statw(msg string, fields ...LogField) {
	if shallLog(InfoLevel) {
		processedFields := processSensitiveFields(fields...)
		getWriter().Stat(callerDepth, msg, processedFields...)
	}
}

func Severew(msg string, fields ...LogField) {
	if shallLog(SevereLevel) {
		// 由于底层Severe方法不接受字段参数，我们只记录消息
		// 如果需要记录字段，可以将它们转换为JSON字符串附加到消息中
		if len(fields) > 0 {
			// 处理字段中的敏感信息
			_ = processSensitiveFields(fields...)
		}
		getWriter().Severe(callerDepth, msg)
	}
}

func Stat(v ...any) {
	if shallLog(InfoLevel) {
		logWithSensitiveHandling("stat", getWriter().Stat, callerDepth, v...)
	}
}

func Statf(format string, v ...any) {
	if shallLog(InfoLevel) {
		processedArgs := processSensitiveArgs(v...)
		getWriter().Stat(callerDepth, fmt.Sprintf(format, processedArgs...))
	}
}

func SetUp(c LogConf) error {
	var err error
	setupOnce.Do(func() {
		setLogLevel(c.Level)
		setupFieldKeys(c.FieldKeys)

		if len(c.TimeFormat) > 0 {
			timeFormat = c.TimeFormat
		}

		if c.MaxContentLength > 0 {
			atomic.StoreUint32(&maxContentLength, c.MaxContentLength)
		}

		switch c.Mode {
		case "file":
			err = setupWithFiles(c)
		case "volume":
			err = setupWithVolume(c)
		case "multi":
			err = setupWithMulti(c)
		default:
			setupWithConsole(c)
		}

		// 重定向系统日志
		if c.CollectSysLog {
			CollectSysLog(c.SysLogLevel)
		}
	})

	return err
}

func WithCaller(skip int) Logger {
	return newLogger(getWriter()).WithCallerSkip(skip)
}

func WithContext(ctx context.Context) Logger {
	return newLogger(getWriter()).WithContext(ctx)
}

func WithDuration(d time.Duration) Logger {
	return newLogger(getWriter()).WithDuration(d)
}

func WithFields(fields ...LogField) Logger {
	return newLogger(getWriter()).WithFields(fields...)
}

func setupFieldKeys(c fieldKeyConf) {
	if len(c.CallerKey) > 0 {
		callerKey = c.CallerKey
	}
	if len(c.ContentKey) > 0 {
		contentKey = c.ContentKey
	}
	if len(c.DurationKey) > 0 {
		durationKey = c.DurationKey
	}
	if len(c.LevelKey) > 0 {
		levelKey = c.LevelKey
	}
	if len(c.SpanKey) > 0 {
		spanKey = c.SpanKey
	}
	if len(c.TimestampKey) > 0 {
		timestampKey = c.TimestampKey
	}
	if len(c.TraceKey) > 0 {
		traceKey = c.TraceKey
	}
	if len(c.TruncatedKey) > 0 {
		truncatedKey = c.TruncatedKey
	}
}

func setupWithConsole(c LogConf) {
	SetWriter(newConsoleWriter(c))
}

func setupWithFiles(c LogConf) error {
	w, err := newFileWriter(c)
	if err != nil {
		return err
	}

	SetWriter(w)
	return nil
}

func setupWithVolume(c LogConf) error {
	if len(c.ServiceName) == 0 {
		return ErrLogServiceNameNotSet
	}

	c.Path = path.Join(c.Path, c.ServiceName)
	return setupWithFiles(c)
}

func setupWithMulti(c LogConf) error {
	// 创建文件写入器
	fileWriter, err := newFileWriter(c)
	if err != nil {
		return err
	}

	// 创建控制台写入器
	consoleWriter := newConsoleWriter(c)

	// 创建多路写入器
	multiWriter := NewMultiWriter(fileWriter, consoleWriter)

	// 设置多路写入器
	SetWriter(multiWriter)

	return nil
}

var ExitOnFatal = true
