package collyx

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/tedwangl/go-util/pkg/collyx/storage"
)

// Client 爬虫客户端
type Client struct {
	collector *colly.Collector
	config    *Config
	logger    *Logger
	queue     *Queue
	storage   storage.Storage
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewClient 创建爬虫客户端
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 创建 collector
	c := colly.NewCollector(
		colly.MaxDepth(cfg.MaxDepth),
		colly.UserAgent(cfg.UserAgent),
	)

	// 设置允许的域名
	if len(cfg.AllowedDomains) > 0 {
		c.AllowedDomains = cfg.AllowedDomains
	}

	// 设置禁止的域名
	if len(cfg.DisallowedDomains) > 0 {
		c.DisallowedDomains = cfg.DisallowedDomains
	}

	// 设置 URL 重访
	c.AllowURLRevisit = cfg.AllowURLRevisit

	// 设置 robots.txt
	c.IgnoreRobotsTxt = cfg.IgnoreRobotsTxt

	// 设置缓存目录
	if cfg.CacheDir != "" {
		c.CacheDir = cfg.CacheDir
	}

	// 设置请求超时
	if cfg.RequestTimeout > 0 {
		c.SetRequestTimeout(cfg.RequestTimeout)
	}

	// 设置限流
	if err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: cfg.Parallelism,
		Delay:       cfg.Delay,
		RandomDelay: cfg.RandomDelay,
	}); err != nil {
		return nil, fmt.Errorf("设置限流失败: %w", err)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		collector: c,
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
	}

	// 设置重定向处理器
	client.setupRedirectHandler()

	// 设置日志
	if cfg.EnableLogger {
		client.logger = NewLogger(cfg.LogLevel, cfg.LogDir)
		client.logger.SetPrintHeaders(cfg.PrintHeaders)
		client.logger.SetPrintCookies(cfg.PrintCookies)
		client.setupLoggerHandlers()
	}

	// 设置重试
	client.setupRetryHandler()

	// 设置用户自定义处理器
	client.setupUserHandlers()

	// 设置队列
	if cfg.EnableQueue {
		client.queue = NewQueue()
	}

	// 设置存储
	if cfg.EnableStorage {
		var err error
		switch cfg.StorageType {
		case "sqlite":
			dbPath := cfg.StorageDir + "/crawler.db"
			client.storage, err = storage.NewSQLiteStorage(dbPath)
		case "mysql":
			client.storage, err = storage.NewMySQLStorage(cfg.StorageDSN)
		default:
			return nil, fmt.Errorf("不支持的存储类型: %s", cfg.StorageType)
		}
		if err != nil {
			return nil, fmt.Errorf("初始化存储失败: %w", err)
		}
	}

	return client, nil
}

// setupRedirectHandler 设置重定向处理器
func (c *Client) setupRedirectHandler() {
	if c.config.RedirectHandler != nil {
		c.collector.SetRedirectHandler(c.config.RedirectHandler)
		return
	}

	// 默认重定向处理器
	c.collector.SetRedirectHandler(func(req *http.Request, via []*http.Request) error {
		if len(via) > c.config.MaxRedirects {
			return fmt.Errorf("超过最大重定向次数: %d", c.config.MaxRedirects)
		}
		return nil
	})
}

// setupLoggerHandlers 设置日志处理器
func (c *Client) setupLoggerHandlers() {
	c.collector.OnRequest(c.logger.HandleRequest)
	c.collector.OnResponse(c.logger.HandleResponse)
	c.collector.OnError(c.logger.HandleError)
}

// setupRetryHandler 设置重试处理器
func (c *Client) setupRetryHandler() {
	c.collector.OnError(func(r *colly.Response, err error) {
		// 获取重试次数
		retryCount := 0
		if val := r.Request.Ctx.GetAny("retryCount"); val != nil {
			if count, ok := val.(int); ok {
				retryCount = count
			}
		}

		// 判断是否需要重试
		shouldRetry := false

		// 检查 HTTP 状态码
		for _, code := range c.config.RetryHTTPCodes {
			if r.StatusCode == code {
				shouldRetry = true
				break
			}
		}

		// 检查超时错误
		if c.config.RetryOnTimeout && (strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "context deadline exceeded")) {
			shouldRetry = true
		}

		// 404 不重试
		if r.StatusCode == 404 {
			shouldRetry = false
		}

		// 如果需要重试且未超过最大重试次数
		if shouldRetry && retryCount < c.config.MaxRetries {
			// 指数退避：1s → 2s → 4s
			delay := time.Duration(math.Pow(2, float64(retryCount))) * time.Second
			log.Printf("[准备重试] URL: %s, 第 %d 次重试，延迟: %v",
				r.Request.URL.String(), retryCount+1, delay)

			// 创建新的上下文
			newCtx := r.Request.Ctx
			newCtx.Put("retryCount", retryCount+1)

			// 延迟后重试
			go func() {
				select {
				case <-c.ctx.Done():
					log.Printf("[请求取消] URL: %s, 爬虫已停止", r.Request.URL.String())
					return
				case <-time.After(delay):
				}

				if c.ctx.Err() != nil {
					log.Printf("[请求取消] URL: %s, 爬虫已停止", r.Request.URL.String())
					return
				}

				// 重试请求
				if err := c.collector.Request(r.Request.Method, r.Request.URL.String(),
					r.Request.Body, newCtx, nil); err != nil {
					log.Printf("[重试失败] URL: %s, 错误: %v", r.Request.URL.String(), err)
				}
			}()
		} else if retryCount >= c.config.MaxRetries {
			log.Printf("[达到最大重试次数] URL: %s, 已尝试 %d 次",
				r.Request.URL.String(), c.config.MaxRetries)
		}
	})
}

// setupUserHandlers 设置用户自定义处理器
func (c *Client) setupUserHandlers() {
	// OnRequest
	for _, handler := range c.config.OnRequest {
		c.collector.OnRequest(handler)
	}

	// OnResponse
	for _, handler := range c.config.OnResponse {
		c.collector.OnResponse(handler)
	}

	// OnHTML
	for selector, handler := range c.config.OnHTML {
		c.collector.OnHTML(selector, handler)
	}

	// OnError
	for _, handler := range c.config.OnError {
		c.collector.OnError(handler)
	}
}

// Visit 访问 URL
func (c *Client) Visit(url string) error {
	if c.ctx.Err() != nil {
		return fmt.Errorf("爬虫已停止: %w", c.ctx.Err())
	}

	// 去重检查
	if c.storage != nil {
		skip, task, err := storage.ShouldSkipTask(c.storage, url, c.config.DuplicateStrategy)
		if err == nil && skip {
			log.Printf("[跳过任务] URL: %s, 原因: 已存在（状态: %s）", url, task.Status)
			return nil
		}
	}

	// 如果启用队列，添加到队列
	if c.queue != nil && c.queue.IsEnabled() {
		c.queue.Add(&Request{
			URL:       url,
			Method:    "GET",
			Priority:  0,
			Timestamp: time.Now(),
		})
		return nil
	}

	// 直接访问
	return c.collector.Visit(url)
}

// VisitWithPriority 带优先级访问 URL（需要启用队列）
func (c *Client) VisitWithPriority(url string, priority int) error {
	if c.queue == nil {
		return fmt.Errorf("队列未启用")
	}

	c.queue.Add(&Request{
		URL:       url,
		Method:    "GET",
		Priority:  priority,
		Timestamp: time.Now(),
	})
	return nil
}

// ProcessQueue 处理队列（需要启用队列）
func (c *Client) ProcessQueue(stopWhenEmpty bool) error {
	if c.queue == nil {
		return fmt.Errorf("队列未启用")
	}

	for {
		if c.ctx.Err() != nil {
			log.Println("[队列处理停止] 上下文已取消")
			break
		}

		req := c.queue.Pop()
		if req == nil {
			if stopWhenEmpty {
				log.Println("[队列处理完成] 队列为空")
				break
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}

		// 执行请求
		if err := c.executeRequest(req); err != nil {
			log.Printf("[请求执行失败] URL: %s, 错误: %v", req.URL, err)
		}
	}

	return nil
}

// executeRequest 执行请求
func (c *Client) executeRequest(req *Request) error {
	if c.ctx.Err() != nil {
		return fmt.Errorf("爬虫已停止")
	}

	// 创建 colly 上下文
	ctx := colly.NewContext()
	ctx.Put("priority", req.Priority)
	ctx.Put("depth", req.Depth)
	ctx.Put("timestamp", req.Timestamp.Format(time.RFC3339))

	// 添加自定义上下文
	for k, v := range req.Ctx {
		ctx.Put(k, v)
	}

	// 执行请求
	var headers http.Header
	if req.Headers != nil {
		headers = *req.Headers
	}
	return c.collector.Request(req.Method, req.URL, nil, ctx, headers)
}

// Wait 等待所有请求完成
func (c *Client) Wait() {
	waitDone := make(chan struct{})

	go func() {
		c.collector.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
	case <-c.ctx.Done():
		log.Println("[Wait 提前返回] 上下文已取消")
	}
}

// Stop 停止爬虫
func (c *Client) Stop() {
	if c.cancel != nil {
		log.Println("[爬虫停止] 正在取消所有请求...")
		c.cancel()
		time.Sleep(100 * time.Millisecond)
		log.Println("[爬虫停止] 已取消所有请求")
	}
}

// Close 关闭爬虫
func (c *Client) Close() error {
	c.Stop()

	if c.logger != nil {
		if err := c.logger.Close(); err != nil {
			return err
		}
	}

	if c.storage != nil {
		if err := c.storage.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Collector 返回底层的 colly.Collector（高级用法）
func (c *Client) Collector() *colly.Collector {
	return c.collector
}

// Queue 返回队列（如果启用）
func (c *Client) Queue() *Queue {
	return c.queue
}

// Logger 返回日志器（如果启用）
func (c *Client) Logger() *Logger {
	return c.logger
}

// Storage 返回存储（如果启用）
func (c *Client) Storage() storage.Storage {
	return c.storage
}
