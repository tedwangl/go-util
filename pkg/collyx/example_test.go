package collyx_test

import (
	"fmt"
	"log"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/tedwangl/go-util/pkg/collyx"
)

// Example_basic 基础用法
func Example_basic() {
	// 创建客户端
	cfg := collyx.DefaultConfig()
	cfg.MaxDepth = 2
	cfg.Parallelism = 5

	// 添加 HTML 处理器
	cfg.OnHTML = map[string]func(*colly.HTMLElement){
		"a[href]": func(e *colly.HTMLElement) {
			link := e.Attr("href")
			fmt.Println("找到链接:", link)
			e.Request.Visit(link)
		},
	}

	client, err := collyx.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// 访问 URL
	client.Visit("https://example.com")

	// 等待完成
	client.Wait()
}

// Example_withLogger 带日志
func Example_withLogger() {
	cfg := collyx.DefaultConfig()
	cfg.EnableLogger = true
	cfg.LogLevel = collyx.LogLevelInfo

	client, _ := collyx.NewClient(cfg)
	defer client.Close()

	client.Visit("https://example.com")
	client.Wait()

	// 查看统计
	if logger := client.Logger(); logger != nil {
		stats := logger.GetStats()
		fmt.Printf("总请求: %d, 成功: %d, 失败: %d\n",
			stats.Total, stats.Success, stats.Failed)
	}
}

// Example_withQueue 带队列
func Example_withQueue() {
	cfg := collyx.DefaultConfig()
	cfg.EnableQueue = true

	client, _ := collyx.NewClient(cfg)
	defer client.Close()

	// 添加请求到队列
	client.VisitWithPriority("https://example.com/page1", 1)
	client.VisitWithPriority("https://example.com/page2", 2)
	client.VisitWithPriority("https://example.com/important", 0) // 高优先级

	// 处理队列
	go client.ProcessQueue(true)

	client.Wait()
}

// Example_queuePersistence 队列持久化
func Example_queuePersistence() {
	cfg := collyx.DefaultConfig()
	cfg.EnableQueue = true

	client, _ := collyx.NewClient(cfg)
	defer client.Close()

	queue := client.Queue()

	// 添加请求
	queue.Add(&collyx.Request{
		URL:      "https://example.com/page1",
		Priority: 1,
	})
	queue.Add(&collyx.Request{
		URL:      "https://example.com/page2",
		Priority: 2,
	})

	// 保存队列
	queue.SaveToFile("queue.json")

	// 稍后加载
	queue.LoadFromFile("queue.json")

	// 处理队列
	go client.ProcessQueue(true)
	client.Wait()
}

// Example_customHandlers 自定义处理器
func Example_customHandlers() {
	cfg := collyx.DefaultConfig()

	// 请求前处理
	cfg.OnRequest = []func(*colly.Request){
		func(r *colly.Request) {
			fmt.Println("访问:", r.URL)
		},
	}

	// 响应后处理
	cfg.OnResponse = []func(*colly.Response){
		func(r *colly.Response) {
			fmt.Printf("状态码: %d, 大小: %d\n", r.StatusCode, len(r.Body))
		},
	}

	// HTML 处理
	cfg.OnHTML = map[string]func(*colly.HTMLElement){
		"title": func(e *colly.HTMLElement) {
			fmt.Println("标题:", e.Text)
		},
		"a[href]": func(e *colly.HTMLElement) {
			link := e.Attr("href")
			e.Request.Visit(link)
		},
	}

	// 错误处理
	cfg.OnError = []func(*colly.Response, error){
		func(r *colly.Response, err error) {
			fmt.Printf("错误: %s, URL: %s\n", err, r.Request.URL)
		},
	}

	client, _ := collyx.NewClient(cfg)
	defer client.Close()

	client.Visit("https://example.com")
	client.Wait()
}

// Example_advancedConfig 高级配置
func Example_advancedConfig() {
	cfg := collyx.DefaultConfig()

	// 限制域名
	cfg.AllowedDomains = []string{"example.com", "www.example.com"}

	// 自定义 User-Agent
	cfg.UserAgent = "MyBot/1.0"

	// 限流
	cfg.Parallelism = 5
	cfg.Delay = 1 * time.Second
	cfg.RandomDelay = 500 * time.Millisecond

	// 重试
	cfg.MaxRetries = 5
	cfg.RetryHTTPCodes = []int{500, 502, 503, 504, 403, 429}

	// 超时
	cfg.RequestTimeout = 30 * time.Second

	// 缓存
	cfg.CacheDir = "./cache"

	client, _ := collyx.NewClient(cfg)
	defer client.Close()

	client.Visit("https://example.com")
	client.Wait()
}

// Example_stopAndResume 停止和恢复
func Example_stopAndResume() {
	cfg := collyx.DefaultConfig()
	cfg.EnableQueue = true

	client, _ := collyx.NewClient(cfg)

	// 添加大量请求
	for i := 0; i < 100; i++ {
		client.VisitWithPriority(fmt.Sprintf("https://example.com/page%d", i), i)
	}

	// 处理一段时间后停止
	go func() {
		time.Sleep(5 * time.Second)
		client.Stop()
	}()

	// 保存未完成的队列
	defer func() {
		if queue := client.Queue(); queue != nil && queue.Size() > 0 {
			queue.SaveToFile("unfinished.json")
			fmt.Printf("保存了 %d 个未完成的请求\n", queue.Size())
		}
		client.Close()
	}()

	client.ProcessQueue(true)
	client.Wait()
}
