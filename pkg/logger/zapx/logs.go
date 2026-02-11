package zap

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

func Alert(v string) {
	getWriter().Alert(v)
}

func Debug(v ...any) {
	if shallLog(DebugLevel) {
		getWriter().Debug(callerDepth, fmt.Sprint(v...))
	}
}

func Debugf(format string, v ...any) {
	if shallLog(DebugLevel) {
		getWriter().Debug(callerDepth, fmt.Sprintf(format, v...))
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
		getWriter().Debug(callerDepth, msg, fields...)
	}
}

func Error(v ...any) {
	if shallLog(ErrorLevel) {
		getWriter().Error(callerDepth, fmt.Sprint(v...))
	}
}

func Errorf(format string, v ...any) {
	if shallLog(ErrorLevel) {
		getWriter().Error(callerDepth, fmt.Errorf(format, v...).Error())
	}
}

func Errorfn(fn func() any) {
	if shallLog(ErrorLevel) {
		getWriter().Error(callerDepth, fn())
	}
}

func ErrorStack(v ...any) {
	if shallLog(ErrorLevel) {
		getWriter().Stack(callerDepth, fmt.Sprint(v...))
	}
}

func ErrorStackf(format string, v ...any) {
	if shallLog(ErrorLevel) {
		getWriter().Stack(callerDepth, fmt.Sprintf(format, v...))
	}
}

func Errorv(v any) {
	if shallLog(ErrorLevel) {
		getWriter().Error(callerDepth, v)
	}
}

func Errorw(msg string, fields ...LogField) {
	if shallLog(ErrorLevel) {
		getWriter().Error(callerDepth, msg, fields...)
	}
}

func Info(v ...any) {
	if shallLog(InfoLevel) {
		getWriter().Info(callerDepth, fmt.Sprint(v...))
	}
}

func Infof(format string, v ...any) {
	if shallLog(InfoLevel) {
		getWriter().Info(callerDepth, fmt.Sprintf(format, v...))
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
		getWriter().Info(callerDepth, msg, fields...)
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
		getWriter().Severe(callerDepth, fmt.Sprint(v...))
	}
}

func Severef(format string, v ...any) {
	if shallLog(SevereLevel) {
		getWriter().Severe(callerDepth, fmt.Sprintf(format, v...))
	}
}

func Slow(v ...any) {
	if shallLog(ErrorLevel) {
		getWriter().Slow(callerDepth, fmt.Sprint(v...))
	}
}

func Slowf(format string, v ...any) {
	if shallLog(ErrorLevel) {
		getWriter().Slow(callerDepth, fmt.Sprintf(format, v...))
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
		getWriter().Slow(callerDepth, msg, fields...)
	}
}

func Stat(v ...any) {
	if shallLog(InfoLevel) {
		getWriter().Stat(callerDepth, fmt.Sprint(v...))
	}
}

func Statf(format string, v ...any) {
	if shallLog(InfoLevel) {
		getWriter().Stat(callerDepth, fmt.Sprintf(format, v...))
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
