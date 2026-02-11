// Package internal 计时器限流器内部实现
package internal

import (
	"context"
	"sync"
	"time"
)

// TimerLimiter 计时器限流器
type TimerLimiter struct {
	interval time.Duration // 间隔时间
	lastCall time.Time     // 上次调用时间
	mutex    sync.RWMutex
	config   InternalConfig
}

// NewTimerLimiter 创建计时器限流器
func NewTimerLimiter(config InternalConfig) *TimerLimiter {
	interval := config.Duration
	if interval == 0 {
		interval = time.Second // 默认1秒
	}

	return &TimerLimiter{
		interval: interval,
		config:   config,
	}
}

// Allow 检查是否允许请求通过
func (t *TimerLimiter) Allow() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	now := time.Now()

	// 检查距离上次调用是否超过了间隔时间
	if now.Sub(t.lastCall) >= t.interval {
		t.lastCall = now
		return true
	}

	return false
}

// AllowN 检查是否允许n个请求通过
func (t *TimerLimiter) AllowN(n int) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	now := time.Now()

	// 计时器限流器一般不允许批量请求通过
	if now.Sub(t.lastCall) >= t.interval {
		t.lastCall = now
		return n == 1 // 只允许单个请求通过
	}

	return false
}

// Wait 等待直到请求被允许（阻塞）
func (t *TimerLimiter) Wait(ctx context.Context) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	now := time.Now()

	// 如果需要等待，计算还需等待的时间
	if now.Sub(t.lastCall) < t.interval {
		waitDuration := t.interval - now.Sub(t.lastCall)

		// 释放锁，以便其他goroutine可以检查
		t.mutex.Unlock()

		// 等待或直到上下文取消
		select {
		case <-time.After(waitDuration):
			t.mutex.Lock() // 重新获取锁
			t.lastCall = time.Now()
			return true
		case <-ctx.Done():
			t.mutex.Lock() // 重新获取锁
			return false
		}
	} else {
		t.lastCall = now
		return true
	}
}

// WaitN 等待直到n个请求被允许（阻塞）
func (t *TimerLimiter) WaitN(ctx context.Context, n int) bool {
	if n != 1 {
		// 计时器限流器通常只允许一个请求通过
		return false
	}

	return t.Wait(ctx)
}

// Type 返回限流器类型
func (t *TimerLimiter) Type() string {
	return "timerx"
}
