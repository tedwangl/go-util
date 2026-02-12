package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type (
	// Scheduler 定时任务调度器
	Scheduler struct {
		cron   *cron.Cron
		jobs   map[string]cron.EntryID
		mu     sync.RWMutex
		logger Logger
	}

	// Job 任务函数（带 context，用于需要取消的场景）
	Job func(ctx context.Context) error

	// SimpleJob 简单任务函数（无需 context）
	SimpleJob func() error

	// Logger 日志接口
	Logger interface {
		Info(msg string, fields ...any)
		Error(msg string, err error, fields ...any)
	}

	// defaultLogger 默认日志实现
	defaultLogger struct{}

	// Option 配置选项
	Option func(*Scheduler)

	// JobInfo 任务信息
	JobInfo struct {
		Name     string    // 任务名称
		Next     time.Time // 下次执行时间
		Prev     time.Time // 上次执行时间
		Schedule string    // Cron 表达式
	}
)

func (l *defaultLogger) Info(msg string, fields ...any) {
	fmt.Printf("[INFO] %s %v\n", msg, fields)
}

func (l *defaultLogger) Error(msg string, err error, fields ...any) {
	fmt.Printf("[ERROR] %s: %v %v\n", msg, err, fields)
}

// WithLogger 设置日志器
func WithLogger(logger Logger) Option {
	return func(s *Scheduler) {
		s.logger = logger
	}
}

// WithSeconds 支持秒级精度（默认分钟级）
// 注意：启用后所有 cron 表达式必须是 6 字段格式（秒 分 时 日 月 周）
func WithSeconds() Option {
	return func(s *Scheduler) {
		s.cron = cron.New(cron.WithSeconds())
	}
}

// NewScheduler 创建调度器
func NewScheduler(opts ...Option) *Scheduler {
	s := &Scheduler{
		cron:   cron.New(),
		jobs:   make(map[string]cron.EntryID),
		logger: &defaultLogger{},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// AddFunc 添加简单任务（推荐，无需处理 context）
func (s *Scheduler) AddFunc(spec, name string, job SimpleJob) error {
	// 包装为 Job 类型
	wrappedJob := func(ctx context.Context) error {
		return job()
	}
	return s.AddJob(spec, name, wrappedJob)
}

// AddJob 添加定时任务（需要 context 的场景）
// spec: cron 表达式
//   - 标准格式（默认）: "分 时 日 月 周" (5 个字段)
//     例如: "0 2 * * *" (每天凌晨 2 点)
//   - 秒级格式（需要 WithSeconds）: "秒 分 时 日 月 周" (6 个字段)
//     例如: "*/5 * * * * *" (每 5 秒)
//   - 预定义（通用）: @every 1h, @daily, @hourly, @weekly, @monthly, @yearly
//     例如: "@every 10s" (每 10 秒)
//
// name: 任务名称（唯一标识）
// job: 任务函数
//
// 注意：
// - Start() 后仍可动态添加任务
// - 预定义表达式（@every 等）在任何模式下都有效
// - 标准 cron 和秒级 cron 不能混用，由创建时的 WithSeconds 决定
func (s *Scheduler) AddJob(spec, name string, job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查任务是否已存在
	if _, exists := s.jobs[name]; exists {
		return fmt.Errorf("job %s already exists", name)
	}

	// 包装任务函数
	wrappedJob := s.wrapJob(name, job)

	// 添加到 cron
	entryID, err := s.cron.AddFunc(spec, wrappedJob)
	if err != nil {
		return fmt.Errorf("failed to add job %s: %w", name, err)
	}

	s.jobs[name] = entryID
	s.logger.Info("job added", "name", name, "spec", spec)

	return nil
}

// RemoveJob 移除任务
func (s *Scheduler) RemoveJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, exists := s.jobs[name]
	if !exists {
		return fmt.Errorf("job %s not found", name)
	}

	s.cron.Remove(entryID)
	delete(s.jobs, name)
	s.logger.Info("job removed", "name", name)

	return nil
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.cron.Start()
	s.logger.Info("scheduler started")
}

// Stop 停止调度器（等待运行中的任务完成）
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("scheduler stopped")
}

// ListJobs 列出所有任务
func (s *Scheduler) ListJobs() []JobInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]JobInfo, 0, len(s.jobs))
	for name, entryID := range s.jobs {
		entry := s.cron.Entry(entryID)
		jobs = append(jobs, JobInfo{
			Name:     name,
			Next:     entry.Next,
			Prev:     entry.Prev,
			Schedule: fmt.Sprintf("%v", entry.Schedule),
		})
	}

	return jobs
}

// wrapJob 包装任务函数，添加日志和错误处理
func (s *Scheduler) wrapJob(name string, job Job) func() {
	return func() {
		ctx := context.Background()
		start := time.Now()

		s.logger.Info("job started", "name", name)

		// 执行任务
		if err := job(ctx); err != nil {
			s.logger.Error("job failed", err, "name", name, "duration", time.Since(start))
		} else {
			s.logger.Info("job completed", "name", name, "duration", time.Since(start))
		}
	}
}

// RunOnce 立即执行一次任务（不影响定时调度）
func (s *Scheduler) RunOnce(name string) error {
	s.mu.RLock()
	entryID, exists := s.jobs[name]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job %s not found", name)
	}

	entry := s.cron.Entry(entryID)
	go entry.Job.Run()

	return nil
}
