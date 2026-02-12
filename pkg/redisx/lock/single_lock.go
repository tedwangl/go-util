package lock

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/tedwangl/go-util/pkg/redisx/client"
)

// SingleLock 单节点分布式锁
type SingleLock struct {
	client  client.Client
	key     string
	value   string
	options *LockOptions

	// 看门狗相关
	watchdogRunning bool
	watchdogMutex   sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewSingleLock 创建单锁
func NewSingleLock(client client.Client, key string, options *LockOptions) *SingleLock {
	if options == nil {
		options = NewLockOptions()
	}

	// 生成随机值，用于释放锁时的验证
	value := fmt.Sprintf("%d:%d", time.Now().UnixNano(), rand.Intn(10000))

	// 创建上下文用于看门狗
	ctx, cancel := context.WithCancel(context.Background())

	return &SingleLock{
		client:  client,
		key:     key,
		value:   value,
		options: options,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Acquire 获取锁
func (l *SingleLock) Acquire(ctx context.Context) error {
	// 尝试获取锁
	for i := 0; i <= l.options.RetryCount; i++ {
		if i > 0 {
			// 重试间隔
			time.Sleep(l.options.RetryInterval)
		}

		err := l.tryAcquire(ctx)
		if err == nil {
			// 获取锁成功，启动看门狗
			if l.options.EnableWatchdog {
				l.startWatchdog()
			}
			return nil
		}
	}

	return fmt.Errorf("failed to acquire lock after %d retries", l.options.RetryCount)
}

// tryAcquire 尝试获取锁（内部方法）
func (l *SingleLock) tryAcquire(ctx context.Context) error {
	// 使用SET NX命令获取锁
	cmd := l.client.SetNX(ctx, l.key, l.value, l.options.Expiration)
	success, err := cmd.Result()
	if err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("lock already held")
	}

	return nil
}

// TryAcquire 尝试获取锁（不重试，用于红锁）
func (l *SingleLock) TryAcquire(ctx context.Context) error {
	return l.tryAcquire(ctx)
}

// Release 释放锁
func (l *SingleLock) Release(ctx context.Context) error {
	// 停止看门狗
	l.stopWatchdog()

	// 使用Lua脚本原子释放锁
	luaScript := `
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	else
		return 0
	end
	`

	cmd := l.client.Eval(ctx, luaScript, []string{l.key}, l.value)
	_, err := cmd.Result()
	if err != nil {
		return err
	}

	return nil
}

// IsLocked 检查锁是否被持有
func (l *SingleLock) IsLocked(ctx context.Context) (bool, error) {
	cmd := l.client.Exists(ctx, l.key)
	count, err := cmd.Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetKey 获取锁的键
func (l *SingleLock) GetKey() string {
	return l.key
}

// startWatchdog 启动看门狗
func (l *SingleLock) startWatchdog() {
	l.watchdogMutex.Lock()
	defer l.watchdogMutex.Unlock()

	if l.watchdogRunning {
		return
	}

	l.watchdogRunning = true

	go func() {
		ticker := time.NewTicker(l.options.WatchdogInterval)
		defer ticker.Stop()

		for {
			select {
			case <-l.ctx.Done():
				l.watchdogMutex.Lock()
				l.watchdogRunning = false
				l.watchdogMutex.Unlock()
				return
			case <-ticker.C:
				// 续期锁
				l.renewLock()
			}
		}
	}()
}

// stopWatchdog 停止看门狗
func (l *SingleLock) stopWatchdog() {
	l.watchdogMutex.Lock()
	defer l.watchdogMutex.Unlock()

	if !l.watchdogRunning {
		return
	}

	l.cancel()
	l.watchdogRunning = false
}

// renewLock 续期锁
func (l *SingleLock) renewLock() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	// 使用Lua脚本原子续期
	luaScript := `
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("expire", KEYS[1], ARGV[2])
	else
		return 0
	end
	`

	cmd := l.client.Eval(ctx, luaScript, []string{l.key}, l.value, int(l.options.Expiration.Seconds()))
	_, err := cmd.Result()
	if err != nil {
		// 续期失败，停止看门狗
		l.stopWatchdog()
	}
}
