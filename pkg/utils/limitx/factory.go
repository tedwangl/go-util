// Package limitx 工厂模式创建限流器
package limitx

import (
	"fmt"
)

// LimiterFactory 限流器工厂
type LimiterFactory struct {
	configManager *ConfigManager
}

// NewLimiterFactory 创建限流器工厂
func NewLimiterFactory() *LimiterFactory {
	return &LimiterFactory{
		configManager: NewConfigManager(),
	}
}

// CreateLimiter 根据类型创建限流器
func (lf *LimiterFactory) CreateLimiter(limiterType string) (Limiter, error) {
	switch limiterType {
	case "token_bucket":
		cfg := lf.configManager.GetTokenBucketConfig()
		if !cfg.Enabled {
			return nil, fmt.Errorf("token bucket limiter is disabled")
		}
		config := Config{
			LimitType: "token_bucket",
			Rate:      cfg.Rate,
			Burst:     cfg.Burst,
		}
		return NewTokenBucketLimiter(config), nil

	case "leaky_bucket":
		cfg := lf.configManager.GetLeakyBucketConfig()
		if !cfg.Enabled {
			return nil, fmt.Errorf("leaky bucket limiter is disabled")
		}
		config := Config{
			LimitType: "leaky_bucket",
			Rate:      cfg.Rate,
			Burst:     cfg.Capacity,
			Duration:  cfg.Duration,
		}
		return NewLeakyBucketLimiter(config), nil

	case "sliding_window":
		cfg := lf.configManager.GetSlidingWindowConfig()
		if !cfg.Enabled {
			return nil, fmt.Errorf("sliding window limiter is disabled")
		}
		config := Config{
			LimitType: "sliding_window",
			Rate:      float64(cfg.MaxCount),
			Burst:     cfg.MaxCount,
			Duration:  cfg.WindowSize,
		}
		return NewSlidingWindowLimiter(config), nil

	case "timerx":
		cfg := lf.configManager.GetTimerConfig()
		if !cfg.Enabled {
			return nil, fmt.Errorf("timerx limiter is disabled")
		}
		config := Config{
			LimitType: "timerx",
			Duration:  cfg.Interval,
		}
		return NewTimerLimiter(config), nil

	default:
		return nil, fmt.Errorf("unknown limiter type: %s", limiterType)
	}
}

// CreateMultiLimiter 创建多策略限流器
func (lf *LimiterFactory) CreateMultiLimiter() *MultiLimiter {
	multiLimiter := NewMultiLimiter()

	// 添加所有启用的限流器
	if tokenBucket, err := lf.CreateLimiter("token_bucket"); err == nil && tokenBucket != nil {
		multiLimiter.AddLimiter("token_bucket", tokenBucket)
	}

	if leakyBucket, err := lf.CreateLimiter("leaky_bucket"); err == nil && leakyBucket != nil {
		multiLimiter.AddLimiter("leaky_bucket", leakyBucket)
	}

	if slidingWindow, err := lf.CreateLimiter("sliding_window"); err == nil && slidingWindow != nil {
		multiLimiter.AddLimiter("sliding_window", slidingWindow)
	}

	if timer, err := lf.CreateLimiter("timerx"); err == nil && timer != nil {
		multiLimiter.AddLimiter("timerx", timer)
	}

	// 设置默认限流器
	defaultLimiter := lf.configManager.GetDefaultLimiter()
	if defaultLimiter != "" {
		multiLimiter.SetCurrent(defaultLimiter)
	}

	return multiLimiter
}
