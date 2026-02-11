// Package limitx 策略切换机制
package limitx

import (
	"context"
	"errors"
	"sync"
)

// MultiLimiter 多策略限流器
type MultiLimiter struct {
	limiterMap map[string]Limiter
	current    string
	mutex      sync.RWMutex
}

// NewMultiLimiter 创建多策略限流器
func NewMultiLimiter() *MultiLimiter {
	return &MultiLimiter{
		limiterMap: make(map[string]Limiter),
	}
}

// AddLimiter 添加限流器
func (m *MultiLimiter) AddLimiter(name string, limiter Limiter) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.limiterMap[name] = limiter
	if m.current == "" {
		m.current = name // 设置默认限流器
	}
}

// SetCurrent 设置当前使用的限流器
func (m *MultiLimiter) SetCurrent(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.limiterMap[name]; !exists {
		return errors.New("limiter not found: " + name)
	}

	m.current = name
	return nil
}

// GetCurrent 获取当前使用的限流器名称
func (m *MultiLimiter) GetCurrent() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.current
}

// Allow 检查是否允许请求通过
func (m *MultiLimiter) Allow() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	limiter, exists := m.limiterMap[m.current]
	if !exists {
		return false // 如果当前限流器不存在，拒绝所有请求
	}

	return limiter.Allow()
}

// AllowN 检查是否允许n个请求通过
func (m *MultiLimiter) AllowN(n int) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	limiter, exists := m.limiterMap[m.current]
	if !exists {
		return false
	}

	return limiter.AllowN(n)
}

// Wait 等待直到请求被允许（阻塞）
func (m *MultiLimiter) Wait(ctx context.Context) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	limiter, exists := m.limiterMap[m.current]
	if !exists {
		return false
	}

	return limiter.Wait(ctx)
}

// WaitN 等待直到n个请求被允许（阻塞）
func (m *MultiLimiter) WaitN(ctx context.Context, n int) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	limiter, exists := m.limiterMap[m.current]
	if !exists {
		return false
	}

	return limiter.WaitN(ctx, n)
}

// Type 返回当前限流器类型
func (m *MultiLimiter) Type() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	limiter, exists := m.limiterMap[m.current]
	if !exists {
		return ""
	}

	return limiter.Type()
}

// GetLimiter 获取指定名称的限流器
func (m *MultiLimiter) GetLimiter(name string) (Limiter, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	limiter, exists := m.limiterMap[name]
	return limiter, exists
}
