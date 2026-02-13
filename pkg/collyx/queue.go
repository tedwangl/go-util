package collyx

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

// Request 请求
type Request struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Priority  int               `json:"priority"`  // 优先级，越小越高
	Depth     int               `json:"depth"`     // 深度
	Timestamp time.Time         `json:"timestamp"` // 时间戳
	Headers   *http.Header      `json:"headers,omitempty"`
	Ctx       map[string]string `json:"ctx,omitempty"` // 上下文
}

// Queue 请求队列
type Queue struct {
	requests []*Request
	mu       sync.Mutex
	enabled  bool
}

// NewQueue 创建队列
func NewQueue() *Queue {
	return &Queue{
		requests: make([]*Request, 0),
		enabled:  true,
	}
}

// Enable 启用队列
func (q *Queue) Enable() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.enabled = true
}

// Disable 禁用队列
func (q *Queue) Disable() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.enabled = false
}

// IsEnabled 是否启用
func (q *Queue) IsEnabled() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.enabled
}

// Add 添加请求
func (q *Queue) Add(req *Request) {
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}
	if req.Ctx == nil {
		req.Ctx = make(map[string]string)
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	q.requests = append(q.requests, req)

	// 排序：优先级 → 时间戳
	sort.Slice(q.requests, func(i, j int) bool {
		if q.requests[i].Priority != q.requests[j].Priority {
			return q.requests[i].Priority < q.requests[j].Priority
		}
		return q.requests[i].Timestamp.Before(q.requests[j].Timestamp)
	})
}

// AddBatch 批量添加请求
func (q *Queue) AddBatch(reqs []*Request) {
	for _, req := range reqs {
		q.Add(req)
	}
}

// Pop 弹出请求
func (q *Queue) Pop() *Request {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.requests) == 0 {
		return nil
	}

	req := q.requests[0]
	q.requests = q.requests[1:]
	return req
}

// Size 队列大小
func (q *Queue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.requests)
}

// Clear 清空队列
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.requests = make([]*Request, 0)
	log.Println("[队列清空] 已清空所有请求")
}

// SaveToFile 保存到文件
func (q *Queue) SaveToFile(filePath string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(q.requests); err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	log.Printf("[队列保存成功] 已保存 %d 个请求到 %s", len(q.requests), filePath)
	return nil
}

// LoadFromFile 从文件加载
func (q *Queue) LoadFromFile(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	var reqs []*Request
	if err := decoder.Decode(&reqs); err != nil {
		return fmt.Errorf("反序列化失败: %w", err)
	}

	q.AddBatch(reqs)

	log.Printf("[队列加载成功] 已从 %s 加载 %d 个请求", filePath, len(reqs))
	return nil
}
