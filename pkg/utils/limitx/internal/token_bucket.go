// Package internal 限流器内部实现
package internal

import (
	"context"
	"time"

	"golang.org/x/time/rate"
)

// InternalConfig 内部限流器配置
type InternalConfig struct {
	LimitType string        // 限流类型: token_bucket, leaky_bucket, sliding_window, timerx
	Rate      float64       // 速率
	Burst     int           // 突发容量
	Duration  time.Duration // 时间窗口
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	limiter *rate.Limiter
	config  InternalConfig
}

// NewTokenBucketLimiter 创建令牌桶限流器
func NewTokenBucketLimiter(config InternalConfig) *TokenBucketLimiter {
	// rate.Limit 表示每秒允许的请求数
	// burst 表示突发容量
	r := rate.Limit(config.Rate)
	burst := config.Burst
	if burst == 0 {
		burst = 1 // 默认突发容量为1
	}

	limiter := rate.NewLimiter(r, burst)
	return &TokenBucketLimiter{
		limiter: limiter,
		config:  config,
	}
}

// Allow 检查是否允许请求通过
func (t *TokenBucketLimiter) Allow() bool {
	return t.limiter.Allow()
}

// AllowN 检查是否允许n个请求通过
func (t *TokenBucketLimiter) AllowN(n int) bool {
	return t.limiter.AllowN(time.Now(), n)
}

// Wait 等待直到请求被允许（阻塞）
func (t *TokenBucketLimiter) Wait(ctx context.Context) bool {
	return t.limiter.Wait(ctx) == nil
}

// WaitN 等待直到n个请求被允许（阻塞）
func (t *TokenBucketLimiter) WaitN(ctx context.Context, n int) bool {
	return t.limiter.WaitN(ctx, n) == nil
}

// Type 返回限流器类型
func (t *TokenBucketLimiter) Type() string {
	return "token_bucket"
}
