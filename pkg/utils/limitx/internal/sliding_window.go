// Package internal 滑动窗口限流器内部实现
package internal

import (
	"context"
	"sync"
	"time"
)

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	windowSize time.Duration // 窗口大小
	maxCount   int           // 窗口内最大请求数
	requests   []time.Time   // 记录请求时间
	mutex      sync.RWMutex
	config     InternalConfig
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
func NewSlidingWindowLimiter(config InternalConfig) *SlidingWindowLimiter {
	windowSize := config.Duration
	if windowSize == 0 {
		windowSize = time.Second // 默认1秒窗口
	}

	maxCount := int(config.Rate)
	if config.Burst > 0 {
		maxCount = config.Burst
	}

	return &SlidingWindowLimiter{
		windowSize: windowSize,
		maxCount:   maxCount,
		requests:   make([]time.Time, 0),
		config:     config,
	}
}

// Allow 检查是否允许请求通过
func (s *SlidingWindowLimiter) Allow() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()

	// 清理窗口外的请求记录
	s.cleanup(now)

	// 检查是否超过限制
	if len(s.requests) >= s.maxCount {
		return false
	}

	// 添加当前请求到记录中
	s.requests = append(s.requests, now)
	return true
}

// AllowN 检查是否允许n个请求通过
func (s *SlidingWindowLimiter) AllowN(n int) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()

	// 清理窗口外的请求记录
	s.cleanup(now)

	// 检查是否超过限制
	if len(s.requests)+n > s.maxCount {
		return false
	}

	// 添加n个请求到记录中
	for i := 0; i < n; i++ {
		s.requests = append(s.requests, now)
	}

	return true
}

// Wait 等待直到请求被允许（阻塞）
func (s *SlidingWindowLimiter) Wait(ctx context.Context) bool {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s.Allow() {
				return true
			}
		case <-ctx.Done():
			return false
		}
	}
}

// WaitN 等待直到n个请求被允许（阻塞）
func (s *SlidingWindowLimiter) WaitN(ctx context.Context, n int) bool {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s.AllowN(n) {
				return true
			}
		case <-ctx.Done():
			return false
		}
	}
}

// cleanup 清理窗口外的请求记录
func (s *SlidingWindowLimiter) cleanup(now time.Time) {
	cutoff := now.Add(-s.windowSize)

	// 找到第一个在窗口内的请求
	startIdx := 0
	for i, reqTime := range s.requests {
		if reqTime.After(cutoff) {
			startIdx = i
			break
		}
	}

	// 保留窗口内的请求
	if startIdx > 0 {
		s.requests = s.requests[startIdx:]
	}
}

// Type 返回限流器类型
func (s *SlidingWindowLimiter) Type() string {
	return "sliding_window"
}
