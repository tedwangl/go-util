package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/tedwangl/go-util/pkg/redisx/client"
)

// RedLock 红锁（分布式锁）
type RedLock struct {
	locks        []*SingleLock
	quorum       int
	options      *LockOptions
	clockDriftMs int64 // 时钟漂移容忍时间（毫秒）
}

// NewRedLock 创建红锁
func NewRedLock(clients []client.Client, key string, options *LockOptions) *RedLock {
	if options == nil {
		options = NewLockOptions()
	}

	// 创建多个单锁（禁用看门狗，由红锁统一管理）
	locks := make([]*SingleLock, len(clients))
	for i, c := range clients {
		lockOpts := *options
		lockOpts.EnableWatchdog = false // 红锁自己管理看门狗
		locks[i] = NewSingleLock(c, key, &lockOpts)
	}

	// 计算法定数量（N/2 + 1）
	quorum := len(clients)/2 + 1

	return &RedLock{
		locks:        locks,
		quorum:       quorum,
		options:      options,
		clockDriftMs: 2, // 默认 2ms 时钟漂移容忍
	}
}

// Acquire 获取锁
func (l *RedLock) Acquire(ctx context.Context) error {
	// 记录开始时间
	startTime := time.Now()

	// 成功获取的锁数量
	successCount := 0

	// 尝试在所有Redis实例上获取锁（快速失败，不重试）
	for _, lock := range l.locks {
		err := lock.TryAcquire(ctx)
		if err == nil {
			successCount++
		}
	}

	// 计算有效时间（减去获取时间和时钟漂移）
	elapsed := time.Since(startTime)
	validityTime := l.options.Expiration - elapsed - time.Duration(l.clockDriftMs)*time.Millisecond

	// 检查是否达到法定数量且有效时间充足
	if successCount < l.quorum || validityTime <= 0 {
		// 获取锁失败，释放已获取的锁
		for _, lock := range l.locks {
			lock.Release(ctx)
		}

		if successCount < l.quorum {
			return fmt.Errorf("failed to acquire redlock: only %d out of %d locks acquired (quorum: %d)",
				successCount, len(l.locks), l.quorum)
		}
		return fmt.Errorf("failed to acquire redlock: validity time %v is not sufficient", validityTime)
	}

	// 获取锁成功，启动看门狗（如果启用）
	if l.options.EnableWatchdog {
		for _, lock := range l.locks {
			lock.startWatchdog()
		}
	}

	return nil
}

// Release 释放锁
func (l *RedLock) Release(ctx context.Context) error {
	// 释放所有锁
	for _, lock := range l.locks {
		lock.Release(ctx)
	}

	return nil
}

// IsLocked 检查锁是否被持有
func (l *RedLock) IsLocked(ctx context.Context) (bool, error) {
	// 检查所有锁
	for _, lock := range l.locks {
		locked, err := lock.IsLocked(ctx)
		if err != nil {
			return false, err
		}

		if locked {
			return true, nil
		}
	}

	return false, nil
}

// GetKey 获取锁的键
func (l *RedLock) GetKey() string {
	if len(l.locks) > 0 {
		return l.locks[0].GetKey()
	}

	return ""
}

// SetClockDrift 设置时钟漂移容忍时间（毫秒）
func (l *RedLock) SetClockDrift(ms int64) {
	l.clockDriftMs = ms
}
