package lock

import (
	"context"
	"fmt"
	"github.com/tedwangl/go-util/pkg/redisx/client"
	"time"
)

// RedLock 红锁（分布式锁）
type RedLock struct {
	locks   []*SingleLock
	quorum  int
	options *LockOptions
}

// NewRedLock 创建红锁
func NewRedLock(clients []client.Client, key string, options *LockOptions) *RedLock {
	if options == nil {
		options = NewLockOptions()
	}

	// 创建多个单锁
	locks := make([]*SingleLock, len(clients))
	for i, c := range clients {
		locks[i] = NewSingleLock(c, key, options)
	}

	// 计算法定数量
	quorum := len(clients)/2 + 1

	return &RedLock{
		locks:   locks,
		quorum:  quorum,
		options: options,
	}
}

// Acquire 获取锁
func (l *RedLock) Acquire(ctx context.Context) error {
	// 记录开始时间
	startTime := time.Now()

	// 成功获取的锁数量
	successCount := 0

	// 尝试在所有Redis实例上获取锁
	for _, lock := range l.locks {
		err := lock.tryAcquire(ctx)
		if err == nil {
			successCount++
		}
	}

	// 检查是否达到法定数量
	if successCount < l.quorum {
		// 获取锁失败，释放已获取的锁
		for _, lock := range l.locks {
			lock.Release(ctx)
		}

		return fmt.Errorf("failed to acquire redlock: only %d out of %d locks acquired", successCount, len(l.locks))
	}

	// 检查获取锁的时间是否超过过期时间的一半
	elapsed := time.Since(startTime)
	if elapsed > l.options.Expiration/2 {
		// 获取锁时间过长，释放所有锁
		for _, lock := range l.locks {
			lock.Release(ctx)
		}

		return fmt.Errorf("failed to acquire redlock: elapsed time %v exceeds half of expiration %v", elapsed, l.options.Expiration)
	}

	// 获取锁成功，启动所有锁的看门狗
	for _, lock := range l.locks {
		if lock.options.EnableWatchdog {
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
