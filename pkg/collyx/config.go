package collyx

import (
	"net/http"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/tedwangl/go-util/pkg/collyx/storage"
)

// Config 爬虫配置
type Config struct {
	// 基础配置
	MaxDepth          int           // 最大深度，默认 1
	UserAgent         string        // User-Agent，默认 Mozilla/5.0
	AllowedDomains    []string      // 允许的域名
	DisallowedDomains []string      // 禁止的域名
	AllowURLRevisit   bool          // 是否允许 URL 重访
	IgnoreRobotsTxt   bool          // 是否忽略 robots.txt，默认 true
	CacheDir          string        // 缓存目录
	RequestTimeout    time.Duration // 请求超时，默认 30s

	// 限流配置
	Parallelism int           // 并发数，默认 10
	Delay       time.Duration // 延迟，默认 500ms
	RandomDelay time.Duration // 随机延迟，默认 500ms

	// 重定向配置
	MaxRedirects int // 最大重定向次数，默认 3

	// 重试配置
	MaxRetries     int   // 最大重试次数，默认 3
	RetryHTTPCodes []int // 需要重试的 HTTP 状态码，默认 5xx 和 403
	RetryOnTimeout bool  // 超时是否重试，默认 true

	// 日志配置
	EnableLogger bool     // 是否启用日志，默认 false
	LogLevel     LogLevel // 日志级别，默认 INFO
	LogDir       string   // 日志目录，默认 log
	PrintHeaders bool     // 是否打印请求头和响应头
	PrintCookies bool     // 是否打印 Cookie

	// 队列配置
	EnableQueue bool // 是否启用队列，默认 false

	// 存储配置
	EnableStorage     bool                      // 是否启用存储，默认 false
	StorageType       string                    // 存储类型：sqlite/mysql，默认 sqlite
	StorageDir        string                    // 存储目录（sqlite），默认 ./data
	StorageDSN        string                    // 数据库连接（mysql）
	DuplicateStrategy storage.DuplicateStrategy // 去重策略，默认 url

	// 自定义处理器
	OnRequest  []func(*colly.Request)
	OnResponse []func(*colly.Response)
	OnHTML     map[string]func(*colly.HTMLElement) // CSS 选择器 -> 处理函数
	OnError    []func(*colly.Response, error)

	// 自定义重定向处理器
	RedirectHandler func(req *http.Request, via []*http.Request) error
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		MaxDepth:          1,
		UserAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		IgnoreRobotsTxt:   true,
		RequestTimeout:    30 * time.Second,
		Parallelism:       10,
		Delay:             500 * time.Millisecond,
		RandomDelay:       500 * time.Millisecond,
		MaxRedirects:      3,
		MaxRetries:        3,
		RetryHTTPCodes:    []int{500, 502, 503, 504, 403},
		RetryOnTimeout:    true,
		EnableLogger:      false,
		LogLevel:          LogLevelInfo,
		LogDir:            "log",
		EnableQueue:       false,
		EnableStorage:     false,
		StorageType:       "sqlite",
		StorageDir:        "./data",
		DuplicateStrategy: storage.DuplicateStrategyURL,
		OnHTML:            make(map[string]func(*colly.HTMLElement)),
	}
}
