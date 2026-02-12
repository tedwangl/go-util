package zapx

import (
	"sync/atomic"
	"time"
)

type limitedExecutor struct {
	threshold time.Duration
	lastTime  atomic.Value
	discarded atomic.Uint32
}

// NewLimitedExecutor 创建一个限流执行器
func NewLimitedExecutor(milliseconds int) *limitedExecutor {
	return newLimitedExecutor(milliseconds)
}

// LogOrDiscard 执行函数或丢弃
func (le *limitedExecutor) LogOrDiscard(execute func()) {
	le.logOrDiscard(execute)
}

func newLimitedExecutor(milliseconds int) *limitedExecutor {
	le := &limitedExecutor{
		threshold: time.Duration(milliseconds) * time.Millisecond,
		lastTime:  atomic.Value{},
	}
	le.lastTime.Store(time.Now().Add(-24 * time.Hour))
	return le
}

func (le *limitedExecutor) logOrDiscard(execute func()) {
	if le == nil || le.threshold <= 0 {
		execute()
		return
	}

	now := time.Now()
	lastTimeValue := le.lastTime.Load()
	var lastTime time.Time
	if lastTimeValue != nil {
		lastTime, _ = lastTimeValue.(time.Time)
	}

	if !lastTime.IsZero() && now.Sub(lastTime) <= le.threshold {
		le.discarded.Add(1)
	} else {
		le.lastTime.Store(now)
		discarded := le.discarded.Swap(0)
		if discarded > 0 {
			Errorf("Discarded %d error messages", discarded)
		}

		execute()
	}
}
