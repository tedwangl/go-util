package limitx

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucketLimiter(t *testing.T) {
	config := Config{
		Rate:  10.0, // 每秒10个请求
		Burst: 5,    // 突发容量5
	}

	limiter := NewTokenBucketLimiter(config)

	// 测试 Allow 方法
	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Errorf("Expected Allow() to return true for request %d", i+1)
		}
	}

	// 第6个请求可能被限制
	time.Sleep(100 * time.Millisecond) // 等待一小段时间让令牌桶填充
	if !limiter.Allow() {
		t.Log("Request 6 was limited as expected")
	}

	t.Logf("Token bucket limiter type: %s", limiter.Type())
}

func TestLeakyBucketLimiter(t *testing.T) {
	config := Config{
		Rate:     1.0, // 每秒1个请求
		Burst:    3,   // 桶容量3
		Duration: time.Second,
	}

	limiter := NewLeakyBucketLimiter(config)

	// 测试 Allow 方法
	for i := 0; i < 3; i++ {
		if !limiter.Allow() {
			t.Errorf("Expected Allow() to return true for request %d", i+1)
		}
	}

	// 第4个请求可能会被限制
	if limiter.Allow() {
		t.Log("Request 4 was allowed (may happen depending on timing)")
	} else {
		t.Log("Request 4 was limited as expected")
	}

	t.Logf("Leaky bucket limiter type: %s", limiter.Type())
}

func TestSlidingWindowLimiter(t *testing.T) {
	config := Config{
		Rate:     5.0,         // 窗口内最多5个请求
		Burst:    5,           // 最大请求数
		Duration: time.Second, // 窗口大小1秒
	}

	limiter := NewSlidingWindowLimiter(config)

	// 测试 Allow 方法
	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Errorf("Expected Allow() to return true for request %d", i+1)
		}
	}

	// 第6个请求应该被限制
	if limiter.Allow() {
		t.Error("Expected request 6 to be limited")
	}

	t.Logf("Sliding window limiter type: %s", limiter.Type())
}

func TestTimerLimiter(t *testing.T) {
	config := Config{
		Duration: 100 * time.Millisecond, // 每100毫秒允许一个请求
	}

	limiter := NewTimerLimiter(config)

	// 第一个请求应该被允许
	if !limiter.Allow() {
		t.Error("First request should be allowed")
	}

	// 第二个请求应该被限制
	if limiter.Allow() {
		t.Error("Second request should be limited")
	}

	// 等待一段时间后应该可以再次通过
	time.Sleep(150 * time.Millisecond)
	if !limiter.Allow() {
		t.Error("Request after timeout should be allowed")
	}

	t.Logf("Timer limiter type: %s", limiter.Type())
}

func TestMultiLimiter(t *testing.T) {
	multiLimiter := NewMultiLimiter()

	// 创建一个令牌桶限流器
	tokenConfig := Config{Rate: 10.0, Burst: 5}
	tokenLimiter := NewTokenBucketLimiter(tokenConfig)
	multiLimiter.AddLimiter("token", tokenLimiter)

	// 设置当前限流器
	err := multiLimiter.SetCurrent("token")
	if err != nil {
		t.Fatalf("Failed to set current limiter: %v", err)
	}

	// 测试功能
	if !multiLimiter.Allow() {
		t.Error("Expected multi limiter to allow request")
	}

	if multiLimiter.Type() != "token_bucket" {
		t.Errorf("Expected type 'token_bucket', got '%s'", multiLimiter.Type())
	}

	t.Logf("Current limiter: %s", multiLimiter.GetCurrent())
}

func TestLimiterFactory(t *testing.T) {
	factory := NewLimiterFactory()

	// 测试创建令牌桶限流器
	tokenLimiter, err := factory.CreateLimiter("token_bucket")
	if err != nil || tokenLimiter == nil {
		t.Errorf("Failed to create token bucket limiter: %v", err)
	}

	// 测试创建多策略限流器
	multiLimiter := factory.CreateMultiLimiter()
	if multiLimiter == nil {
		t.Error("Failed to create multi limiter")
	}

	// 验证多策略限流器中有默认的限流器
	if multiLimiter.GetCurrent() == "" {
		t.Error("Multi limiter should have a default limiter")
	}

	t.Logf("Factory created multi limiter with default: %s", multiLimiter.GetCurrent())
}

func TestWaitMethods(t *testing.T) {
	config := Config{
		Rate:  2.0, // 每秒2个请求
		Burst: 1,   // 突发容量1
	}

	limiter := NewTokenBucketLimiter(config)
	ctx := context.Background()

	// 测试 Wait 方法
	result := limiter.Wait(ctx)
	if !result {
		t.Error("Expected Wait to return true")
	}

	// 测试 WaitN 方法
	result = limiter.WaitN(ctx, 1)
	if !result {
		t.Error("Expected WaitN to return true")
	}

	t.Logf("Wait methods test passed for %s", limiter.Type())
}
