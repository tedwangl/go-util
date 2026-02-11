package zap

import (
	"errors"
	"sync/atomic"
)

const (
	DebugLevel uint32 = iota
	InfoLevel
	ErrorLevel
	SevereLevel
	disableLevel = 0xff
)

const (
	accessFilename = "access.log"
	errorFilename  = "error.log"
	severeFilename = "severe.log"
	slowFilename   = "slow.log"
	statFilename   = "stat.log"
)

const (
	levelDebug  = "debug"
	levelInfo   = "info"
	levelError  = "error"
	levelSevere = "severe"
	levelFatal  = "fatal"
	levelSlow   = "slow"
	levelStat   = "stat"
	levelAlert  = "alert"
)

const (
	defaultCallerKey    = "caller"
	defaultContentKey   = "content"
	defaultDurationKey  = "duration"
	defaultLevelKey     = "level"
	defaultSpanKey      = "span"
	defaultTimestampKey = "@timestamp"
	defaultTraceKey     = "trace"
	defaultTruncatedKey = "truncated"
)

var (
	ErrLogPathNotSet      = errors.New("log path must be set")
	ErrLogServiceNameNotSet = errors.New("log service name must be set")
)

var (
	callerKey    = defaultCallerKey
	contentKey   = defaultContentKey
	durationKey  = defaultDurationKey
	levelKey     = defaultLevelKey
	spanKey      = defaultSpanKey
	timestampKey = defaultTimestampKey
	traceKey     = defaultTraceKey
	truncatedKey = defaultTruncatedKey
)

var (
	logLevel    uint32
	maxContentLength uint32
)

func setLogLevel(level string) {
	switch level {
	case "debug":
		atomic.StoreUint32(&logLevel, DebugLevel)
	case "info":
		atomic.StoreUint32(&logLevel, InfoLevel)
	case "error":
		atomic.StoreUint32(&logLevel, ErrorLevel)
	case "severe":
		atomic.StoreUint32(&logLevel, SevereLevel)
	}
}

func shallLog(level uint32) bool {
	return atomic.LoadUint32(&logLevel) <= level
}

func SetLevel(level uint32) {
	atomic.StoreUint32(&logLevel, level)
}
