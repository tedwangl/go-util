// Package limitx 配置管理
package limitx

import (
	"encoding/json"
	"io/ioutil"
	"time"
)

// LimitConfig 限流配置
type LimitConfig struct {
	TokenBucket    TokenBucketConfig   `json:"token_bucket"`
	LeakyBucket    LeakyBucketConfig   `json:"leaky_bucket"`
	SlidingWindow  SlidingWindowConfig `json:"sliding_window"`
	Timer          TimerConfig         `json:"timerx"`
	DefaultLimiter string              `json:"default_limiter"`
}

// TokenBucketConfig 令牌桶配置
type TokenBucketConfig struct {
	Enabled bool    `json:"enabled"`
	Rate    float64 `json:"rate"`
	Burst   int     `json:"burst"`
}

// LeakyBucketConfig 漏桶配置
type LeakyBucketConfig struct {
	Enabled  bool          `json:"enabled"`
	Rate     float64       `json:"rate"`
	Capacity int           `json:"capacity"`
	Duration time.Duration `json:"duration"`
}

// SlidingWindowConfig 滑动窗口配置
type SlidingWindowConfig struct {
	Enabled    bool          `json:"enabled"`
	MaxCount   int           `json:"max_count"`
	WindowSize time.Duration `json:"window_size"`
}

// TimerConfig 计时器配置
type TimerConfig struct {
	Enabled  bool          `json:"enabled"`
	Interval time.Duration `json:"interval"`
}

// ConfigManager 配置管理器
type ConfigManager struct {
	config LimitConfig
}

// NewConfigManager 创建配置管理器
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config: getDefaultConfig(),
	}
}

// LoadConfig 从文件加载配置
func (cm *ConfigManager) LoadConfig(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &cm.config)
	if err != nil {
		return err
	}

	return nil
}

// SaveConfig 保存配置到文件
func (cm *ConfigManager) SaveConfig(filePath string) error {
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, data, 0644)
}

// GetTokenBucketConfig 获取令牌桶配置
func (cm *ConfigManager) GetTokenBucketConfig() TokenBucketConfig {
	return cm.config.TokenBucket
}

// GetLeakyBucketConfig 获取漏桶配置
func (cm *ConfigManager) GetLeakyBucketConfig() LeakyBucketConfig {
	return cm.config.LeakyBucket
}

// GetSlidingWindowConfig 获取滑动窗口配置
func (cm *ConfigManager) GetSlidingWindowConfig() SlidingWindowConfig {
	return cm.config.SlidingWindow
}

// GetTimerConfig 获取计时器配置
func (cm *ConfigManager) GetTimerConfig() TimerConfig {
	return cm.config.Timer
}

// GetDefaultLimiter 获取默认限流器
func (cm *ConfigManager) GetDefaultLimiter() string {
	return cm.config.DefaultLimiter
}

// SetTokenBucketConfig 设置令牌桶配置
func (cm *ConfigManager) SetTokenBucketConfig(cfg TokenBucketConfig) {
	cm.config.TokenBucket = cfg
}

// SetLeakyBucketConfig 设置漏桶配置
func (cm *ConfigManager) SetLeakyBucketConfig(cfg LeakyBucketConfig) {
	cm.config.LeakyBucket = cfg
}

// SetSlidingWindowConfig 设置滑动窗口配置
func (cm *ConfigManager) SetSlidingWindowConfig(cfg SlidingWindowConfig) {
	cm.config.SlidingWindow = cfg
}

// SetTimerConfig 设置计时器配置
func (cm *ConfigManager) SetTimerConfig(cfg TimerConfig) {
	cm.config.Timer = cfg
}

// SetDefaultLimiter 设置默认限流器
func (cm *ConfigManager) SetDefaultLimiter(name string) {
	cm.config.DefaultLimiter = name
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() LimitConfig {
	return LimitConfig{
		TokenBucket: TokenBucketConfig{
			Enabled: true,
			Rate:    100.0, // 每秒100个请求
			Burst:   50,    // 突发容量50
		},
		LeakyBucket: LeakyBucketConfig{
			Enabled:  true,
			Rate:     1.0, // 每秒漏1个请求
			Capacity: 100, // 桶容量100
			Duration: time.Second,
		},
		SlidingWindow: SlidingWindowConfig{
			Enabled:    true,
			MaxCount:   100,         // 每窗口最多100个请求
			WindowSize: time.Second, // 窗口大小1秒
		},
		Timer: TimerConfig{
			Enabled:  false,       // 默认禁用计时器限流
			Interval: time.Second, // 间隔1秒
		},
		DefaultLimiter: "token_bucket", // 默认使用令牌桶
	}
}
