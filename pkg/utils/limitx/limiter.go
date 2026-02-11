// Package limitx 提供多种限流算法的实现
package limitx

import (
	"context"
	internal2 "github.com/tedwangl/go-util/pkg/utils/limitx/internal"
	"time"
)

// Limiter 限流器接口
type Limiter interface {
	// Allow 检查是否允许请求通过
	Allow() bool

	// AllowN 检查是否允许n个请求通过
	AllowN(n int) bool

	// Wait 等待直到请求被允许（阻塞）
	Wait(ctx context.Context) bool

	// WaitN 等待直到n个请求被允许（阻塞）
	WaitN(ctx context.Context, n int) bool

	// Type 返回限流器类型
	Type() string
}

// Config 限流器通用配置
type Config struct {
	LimitType string        // 限流类型: token_bucket, leaky_bucket, sliding_window, timerx
	Rate      float64       // 速率
	Burst     int           // 突发容量
	Duration  time.Duration // 时间窗口
}

// 将内部实现包装成公共接口
type tokenBucketLimiter struct {
	*internal2.TokenBucketLimiter
}

type leakyBucketLimiter struct {
	*internal2.LeakyBucketLimiter
}

type slidingWindowLimiter struct {
	*internal2.SlidingWindowLimiter
}

type timerLimiter struct {
	*internal2.TimerLimiter
}

// NewTokenBucketLimiter 创建令牌桶限流器
func NewTokenBucketLimiter(config Config) Limiter {
	internalConfig := internal2.InternalConfig{
		LimitType: config.LimitType,
		Rate:      config.Rate,
		Burst:     config.Burst,
		Duration:  config.Duration,
	}

	return &tokenBucketLimiter{
		TokenBucketLimiter: internal2.NewTokenBucketLimiter(internalConfig),
	}
}

// NewLeakyBucketLimiter 创建漏桶限流器
func NewLeakyBucketLimiter(config Config) Limiter {
	internalConfig := internal2.InternalConfig{
		LimitType: config.LimitType,
		Rate:      config.Rate,
		Burst:     config.Burst,
		Duration:  config.Duration,
	}

	return &leakyBucketLimiter{
		LeakyBucketLimiter: internal2.NewLeakyBucketLimiter(internalConfig),
	}
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
func NewSlidingWindowLimiter(config Config) Limiter {
	internalConfig := internal2.InternalConfig{
		LimitType: config.LimitType,
		Rate:      config.Rate,
		Burst:     config.Burst,
		Duration:  config.Duration,
	}

	return &slidingWindowLimiter{
		SlidingWindowLimiter: internal2.NewSlidingWindowLimiter(internalConfig),
	}
}

// NewTimerLimiter 创建计时器限流器
func NewTimerLimiter(config Config) Limiter {
	internalConfig := internal2.InternalConfig{
		LimitType: config.LimitType,
		Rate:      config.Rate,
		Burst:     config.Burst,
		Duration:  config.Duration,
	}

	return &timerLimiter{
		TimerLimiter: internal2.NewTimerLimiter(internalConfig),
	}
}
