package utils

import (
	"fmt"
	"time"

	"github.com/tedwangl/go-util/pkg/utils/timex"
)

// ElapsedTimer 用于跟踪经过时间的计时器
type ElapsedTimer struct {
	start time.Duration
}

// NewElapsedTimer 创建并返回一个 ElapsedTimer
// 用法示例：
//
//	timer := NewElapsedTimer()
//	time.Sleep(100 * time.Millisecond)
//	fmt.Println(timer.Elapsed()) // 输出: 100ms
func NewElapsedTimer() *ElapsedTimer {
	return &ElapsedTimer{
		start: timex.Now(),
	}
}

// Duration 返回经过的时间
// 用法示例：
//
//	timer := NewElapsedTimer()
//	time.Sleep(100 * time.Millisecond)
//	duration := timer.Duration()
//	fmt.Println(duration) // 输出: 100ms
func (et *ElapsedTimer) Duration() time.Duration {
	return timex.Since(et.start)
}

// Elapsed 返回经过时间的字符串表示
// 用法示例：
//
//	timer := NewElapsedTimer()
//	time.Sleep(50 * time.Millisecond)
//	fmt.Println(timer.Elapsed()) // 输出: 50ms
func (et *ElapsedTimer) Elapsed() string {
	return timex.Since(et.start).String()
}

// ElapsedMs 返回经过时间的毫秒数字符串表示
// 用法示例：
//
//	timer := NewElapsedTimer()
//	time.Sleep(123 * time.Millisecond)
//	fmt.Println(timer.ElapsedMs()) // 输出: 123.0ms
func (et *ElapsedTimer) ElapsedMs() string {
	return fmt.Sprintf("%.1fms", float32(timex.Since(et.start))/float32(time.Millisecond))
}

// CurrentMicros 返回当前的微秒数
// 用法示例：
//
//	micros := CurrentMicros()
//	fmt.Println(micros) // 输出: 1234567890
func CurrentMicros() int64 {
	return time.Now().UnixNano() / int64(time.Microsecond)
}

// CurrentMillis 返回当前的毫秒数
// 用法示例：
//
//	millis := CurrentMillis()
//	fmt.Println(millis) // 输出: 1234567890
func CurrentMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
