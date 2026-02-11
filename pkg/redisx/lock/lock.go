package lock

import (
	"context"
	"time"
)

// Lock 分布式锁接口
type Lock interface {
	// Acquire 获取锁
	Acquire(ctx context.Context) error
	
	// Release 释放锁
	Release(ctx context.Context) error
	
	// IsLocked 检查锁是否被持有
	IsLocked(ctx context.Context) (bool, error)
	
	// GetKey 获取锁的键
	GetKey() string
}

// LockOptions 锁选项
type LockOptions struct {
	// 过期时间
	Expiration time.Duration
	
	// 重试次数
	RetryCount int
	
	// 重试间隔
	RetryInterval time.Duration
	
	// 是否启用看门狗
	EnableWatchdog bool
	
	// 看门狗检查间隔
	WatchdogInterval time.Duration
}

// NewLockOptions 创建默认锁选项
func NewLockOptions() *LockOptions {
	return &LockOptions{
		Expiration:       time.Second * 10,
		RetryCount:       3,
		RetryInterval:    time.Millisecond * 100,
		EnableWatchdog:   true,
		WatchdogInterval: time.Second * 3,
	}
}
