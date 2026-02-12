package zapx

import (
	"io"

	"gopkg.in/natefinch/lumberjack.v2"
)

type RotateRule interface {
	CurrentFileName() string
	Gzip() bool
	MaxBackups() int
	MaxAge() int
	MaxSize() int
}

type sizeRotateRule struct {
	filename   string
	maxSize    int
	maxBackups int
	maxAge     int
	gzip       bool
}

func NewSizeRotateRule(filename string, maxSize, maxBackups, maxAge int, gzip bool) RotateRule {
	return &sizeRotateRule{
		filename:   filename,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxAge:     maxAge,
		gzip:       gzip,
	}
}

func (r *sizeRotateRule) CurrentFileName() string {
	return r.filename
}

func (r *sizeRotateRule) Gzip() bool {
	return r.gzip
}

func (r *sizeRotateRule) MaxBackups() int {
	return r.maxBackups
}

func (r *sizeRotateRule) MaxAge() int {
	return r.maxAge
}

func (r *sizeRotateRule) MaxSize() int {
	return r.maxSize
}

type dailyRotateRule struct {
	filename   string
	maxBackups int
	maxAge     int
	gzip       bool
}

func NewDailyRotateRule(filename string, maxBackups, maxAge int, gzip bool) RotateRule {
	return &dailyRotateRule{
		filename:   filename,
		maxBackups: maxBackups,
		maxAge:     maxAge,
		gzip:       gzip,
	}
}

func (r *dailyRotateRule) CurrentFileName() string {
	return r.filename
}

func (r *dailyRotateRule) Gzip() bool {
	return r.gzip
}

func (r *dailyRotateRule) MaxBackups() int {
	return r.maxBackups
}

func (r *dailyRotateRule) MaxAge() int {
	return r.maxAge
}

func (r *dailyRotateRule) MaxSize() int {
	return 0
}

type RotateLogger struct {
	*lumberjack.Logger
	rule RotateRule
}

func NewRotateLogger(filename string, rule RotateRule) *RotateLogger {
	return &RotateLogger{
		Logger: &lumberjack.Logger{
			Filename:   filename,
			MaxSize:    rule.MaxSize(),
			MaxBackups: rule.MaxBackups(),
			MaxAge:     rule.MaxAge(),
			Compress:   rule.Gzip(),
		},
		rule: rule,
	}
}

func (l *RotateLogger) Write(p []byte) (n int, err error) {
	return l.Logger.Write(p)
}

func (l *RotateLogger) Close() error {
	return l.Logger.Close()
}

func createRotateWriter(filename string, c LogConf) io.WriteCloser {
	var rule RotateRule

	if c.Rotation == "size" {
		rule = NewSizeRotateRule(filename, c.MaxSize, c.MaxBackups, c.KeepDays, c.Compress)
	} else {
		rule = NewDailyRotateRule(filename, c.MaxBackups, c.KeepDays, c.Compress)
	}

	return NewRotateLogger(filename, rule)
}
