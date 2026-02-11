// Package internal 漏桶限流器内部实现
package internal

import (
	"context"
	"sync"
	"time"
)

// LeakyBucketLimiter 漏桶限流器
type LeakyBucketLimiter struct {
	capacity int           // 桶容量
	rate     time.Duration // 漏水速率（每单位时间漏出多少请求）
	queue    chan struct{} // 请求队列
	mutex    sync.Mutex
	config   InternalConfig
}

// NewLeakyBucketLimiter 创建漏桶限流器
func NewLeakyBucketLimiter(config InternalConfig) *LeakyBucketLimiter {
	capacity := int(config.Rate)
	if config.Burst > 0 {
		capacity = config.Burst
	}

	// 默认每秒漏一个请求
	rate := time.Second
	if config.Duration > 0 {
		rate = config.Duration
	}

	limiter := &LeakyBucketLimiter{
		capacity: capacity,
		rate:     rate,
		queue:    make(chan struct{}, capacity),
		config:   config,
	}

	// 启动漏水协程
	go limiter.startDraining()

	return limiter
}

// startDraining 开始漏水过程
func (l *LeakyBucketLimiter) startDraining() {
	ticker := time.NewTicker(l.rate)
	defer ticker.Stop()

	for range ticker.C {
		// 尝试从桶中取出一个请求（漏水）
		select {
		case <-l.queue:
		default:
			// 桶是空的，无需漏水
		}
	}
}

// Allow 检查是否允许请求通过
func (l *LeakyBucketLimiter) Allow() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	select {
	case l.queue <- struct{}{}: // 尝试将请求放入桶中
		return true
	default: // 桶满了，拒绝请求
		return false
	}
}

// AllowN 检查是否允许n个请求通过
func (l *LeakyBucketLimiter) AllowN(n int) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 检查是否有足够的空间容纳n个请求
	if len(l.queue)+n > l.capacity {
		return false
	}

	// 添加n个请求到桶中
	for i := 0; i < n; i++ {
		select {
		case l.queue <- struct{}{}:
		default:
			return false
		}
	}

	return true
}

// Wait 等待直到请求被允许（阻塞）
func (l *LeakyBucketLimiter) Wait(ctx context.Context) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	select {
	case l.queue <- struct{}{}: // 尝试将请求放入桶中
		return true
	case <-ctx.Done():
		return false
	}
}

// WaitN 等待直到n个请求被允许（阻塞）
func (l *LeakyBucketLimiter) WaitN(ctx context.Context, n int) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	for i := 0; i < n; i++ {
		select {
		case l.queue <- struct{}{}:
		case <-ctx.Done():
			return false
		}
	}

	return true
}

// Type 返回限流器类型
func (l *LeakyBucketLimiter) Type() string {
	return "leaky_bucket"
}
