package collyx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel 日志级别
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelError LogLevel = "ERROR"
)

// Stats 统计信息
type Stats struct {
	Total     int64
	Success   int64
	Failed    int64
	Remaining int64
	StartTime time.Time
}

// Logger 日志器
type Logger struct {
	logger       *zap.Logger
	level        LogLevel
	printHeaders bool
	printCookies bool
	stats        *Stats
	statsMu      sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewLogger 创建日志器
func NewLogger(level LogLevel, logDir string) *Logger {
	if logDir == "" {
		logDir = "log"
	}

	// 确保目录存在
	os.MkdirAll(logDir, 0755)

	// 日志文件
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", strings.ToLower(string(level))))

	// 日志轮转
	writer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    512, // MB
		MaxBackups: 10,
		MaxAge:     28, // days
		Compress:   false,
	})

	// 编码器配置
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// 日志级别
	var levelEnabler zapcore.LevelEnabler
	switch level {
	case LogLevelDebug:
		levelEnabler = zap.DebugLevel
	case LogLevelInfo:
		levelEnabler = zap.InfoLevel
	case LogLevelError:
		levelEnabler = zap.ErrorLevel
	default:
		levelEnabler = zap.InfoLevel
	}

	// 创建 logger
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writer,
		levelEnabler,
	)
	logger := zap.New(core)

	ctx, cancel := context.WithCancel(context.Background())

	l := &Logger{
		logger: logger,
		level:  level,
		stats: &Stats{
			StartTime: time.Now(),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// 启动统计服务
	go l.serveStats()

	return l
}

// SetPrintHeaders 设置是否打印请求头和响应头
func (l *Logger) SetPrintHeaders(print bool) {
	l.printHeaders = print
}

// SetPrintCookies 设置是否打印 Cookie
func (l *Logger) SetPrintCookies(print bool) {
	l.printCookies = print
}

// HandleRequest 处理请求
func (l *Logger) HandleRequest(r *colly.Request) {
	l.statsMu.Lock()
	l.stats.Total++
	l.stats.Remaining++
	l.statsMu.Unlock()

	r.Ctx.Put("startTime", time.Now())

	switch l.level {
	case LogLevelDebug:
		l.logger.Debug("请求开始",
			zap.String("url", r.URL.String()),
			zap.String("method", r.Method),
			zap.Int("depth", r.Depth),
		)
		if l.printHeaders && r.Headers != nil {
			for k, v := range *r.Headers {
				l.logger.Debug("请求头", zap.String("key", k), zap.Strings("value", v))
			}
		}
	case LogLevelInfo:
		l.logger.Info("请求开始",
			zap.String("url", r.URL.String()),
			zap.String("method", r.Method),
		)
	}
}

// HandleResponse 处理响应
func (l *Logger) HandleResponse(r *colly.Response) {
	l.statsMu.Lock()
	l.stats.Success++
	if l.stats.Remaining > 0 {
		l.stats.Remaining--
	}
	l.statsMu.Unlock()

	duration := time.Duration(0)
	if startTime, ok := r.Request.Ctx.GetAny("startTime").(time.Time); ok {
		duration = time.Since(startTime)
	}

	switch l.level {
	case LogLevelDebug:
		l.logger.Debug("请求成功",
			zap.String("url", r.Request.URL.String()),
			zap.Int("status", r.StatusCode),
			zap.Duration("duration", duration),
		)
		if l.printHeaders {
			for k, v := range *r.Headers {
				l.logger.Debug("响应头", zap.String("key", k), zap.Strings("value", v))
			}
		}
	case LogLevelInfo:
		l.logger.Info("请求成功",
			zap.String("url", r.Request.URL.String()),
			zap.Int("status", r.StatusCode),
			zap.Duration("duration", duration),
		)
	}
}

// HandleError 处理错误
func (l *Logger) HandleError(r *colly.Response, err error) {
	l.statsMu.Lock()
	l.stats.Failed++
	if l.stats.Remaining > 0 {
		l.stats.Remaining--
	}
	l.statsMu.Unlock()

	duration := time.Duration(0)
	if startTime, ok := r.Request.Ctx.GetAny("startTime").(time.Time); ok {
		duration = time.Since(startTime)
	}

	l.logger.Error("请求失败",
		zap.String("url", r.Request.URL.String()),
		zap.Int("status", r.StatusCode),
		zap.Duration("duration", duration),
		zap.Error(err),
	)
}

// GetStats 获取统计信息
func (l *Logger) GetStats() Stats {
	l.statsMu.Lock()
	defer l.statsMu.Unlock()
	return *l.stats
}

// serveStats 定期输出统计信息
func (l *Logger) serveStats() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if l.level == LogLevelInfo || l.level == LogLevelDebug {
				stats := l.GetStats()
				l.logger.Info("爬取进度",
					zap.Int64("总数", stats.Total),
					zap.Int64("成功", stats.Success),
					zap.Int64("失败", stats.Failed),
					zap.Int64("剩余", stats.Remaining),
					zap.Duration("耗时", time.Since(stats.StartTime)),
				)
			}
		case <-l.ctx.Done():
			return
		}
	}
}

// Close 关闭日志器
func (l *Logger) Close() error {
	if l.cancel != nil {
		l.cancel()
	}
	return l.logger.Sync()
}
