package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/tedwangl/go-util/pkg/scheduler"
	genid "github.com/tedwangl/go-util/pkg/utils/snowflake"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type (
	// TaskStatus 任务状态
	TaskStatus string

	// Task 任务（通用）
	Task struct {
		ID          int64      `gorm:"primarykey" json:"id"`             // 雪花ID
		Name        string     `gorm:"uniqueIndex;not null" json:"name"` // 任务名称
		Command     string     `gorm:"not null" json:"command"`          // 执行命令
		Schedule    string     `gorm:"default:''" json:"schedule"`       // cron 表达式或特殊标记（@once, @delay:5m）
		Enabled     bool       `gorm:"default:true" json:"enabled"`      // 是否启用
		Completed   bool       `gorm:"default:false" json:"completed"`   // 是否已完成（once/delay 任务用）
		RunAt       *time.Time `json:"run_at,omitempty"`                 // 指定执行时间（用于延迟任务）
		CompletedAt *time.Time `json:"completed_at,omitempty"`           // 完成时间
		CreatedAt   time.Time  `json:"created_at"`
		UpdatedAt   time.Time  `json:"updated_at"`
	}

	// TaskLog 任务执行日志（只记录状态）
	TaskLog struct {
		ID        int64      `gorm:"primarykey" json:"id"`          // 雪花ID
		TaskID    int64      `gorm:"index;not null" json:"task_id"` // 任务ID
		TaskName  string     `gorm:"index" json:"task_name"`        // 任务名称
		PID       int        `gorm:"default:0" json:"pid"`          // 进程ID（运行中时有效）
		StartTime time.Time  `json:"start_time"`                    // 开始时间
		EndTime   *time.Time `json:"end_time"`                      // 结束时间
		Status    string     `json:"status"`                        // success, failed, running, killed
	}

	// Daemon 任务守护进程
	Daemon struct {
		DB        *gorm.DB // 暴露给外部访问
		scheduler *scheduler.Scheduler
		dbPath    string
		idGen     *genid.SnowflakeID
		started   bool // 标记 scheduler 是否已启动
	}
)

const (
	TaskStatusSuccess = "success"
	TaskStatusFailed  = "failed"
	TaskStatusRunning = "running"
)

// NewDaemon 创建守护进程
func NewDaemon(dbPath string) (*Daemon, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	// 打开数据库
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&Task{}, &TaskLog{}); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 创建雪花ID生成器（节点ID=1）
	idGen, err := genid.NewSnowflakeID(1)
	if err != nil {
		return nil, fmt.Errorf("创建ID生成器失败: %w", err)
	}

	return &Daemon{
		DB:        db,
		scheduler: scheduler.NewScheduler(scheduler.WithSeconds()),
		dbPath:    dbPath,
		idGen:     idGen,
	}, nil
}

// Start 启动守护进程（只启动有调度的任务）
func (d *Daemon) Start() error {
	if err := d.loadTasks(); err != nil {
		return err
	}

	// 启动调度器
	d.scheduler.Start()
	d.started = true
	return nil
}

// loadTasks 加载所有任务到调度器（只加载未完成的调度任务）
func (d *Daemon) loadTasks() error {
	// 加载所有启用的、未完成的调度任务
	var tasks []Task
	if err := d.DB.Where("enabled = ? AND completed = ? AND schedule != ''", true, false).Find(&tasks).Error; err != nil {
		return fmt.Errorf("加载任务失败: %w", err)
	}

	// 注册任务到调度器
	for i := range tasks {
		task := &tasks[i]
		taskID := task.ID // 捕获 ID，避免闭包问题
		taskName := task.Name

		if err := d.scheduler.AddFunc(task.Schedule, task.Name, func() error {
			// 每次执行时从数据库加载最新任务配置
			var currentTask Task
			if err := d.DB.Where("id = ?", taskID).First(&currentTask).Error; err != nil {
				fmt.Printf("加载任务 %s 失败: %v\n", taskName, err)
				return err
			}
			d.executeTask(&currentTask)
			return nil
		}); err != nil {
			return fmt.Errorf("注册任务 %s 失败: %w", task.Name, err)
		}
	}

	return nil
}

// Reload 重新加载任务（删除所有旧任务，重新加载）
func (d *Daemon) Reload() error {
	// 获取当前所有任务
	jobs := d.scheduler.ListJobs()
	for _, job := range jobs {
		d.scheduler.RemoveJob(job.Name)
	}

	// 重新加载
	return d.loadTasks()
}

// RemoveJobFromScheduler 从调度器中移除任务（不影响正在执行的任务）
func (d *Daemon) RemoveJobFromScheduler(name string) error {
	return d.scheduler.RemoveJob(name)
}

// AddJobToScheduler 添加任务到调度器
func (d *Daemon) AddJobToScheduler(task *Task) error {
	// 重新从数据库加载任务，确保数据最新
	var t Task
	if err := d.DB.Where("id = ?", task.ID).First(&t).Error; err != nil {
		return fmt.Errorf("加载任务失败: %w", err)
	}

	return d.scheduler.AddFunc(t.Schedule, t.Name, func() error {
		// 每次执行时重新加载任务，确保使用最新配置
		var currentTask Task
		if err := d.DB.Where("id = ?", t.ID).First(&currentTask).Error; err != nil {
			fmt.Printf("加载任务 %s 失败: %v\n", t.Name, err)
			return err
		}
		d.executeTask(&currentTask)
		return nil
	})
}

// Stop 停止守护进程
func (d *Daemon) Stop() {
	if d.started {
		d.scheduler.Stop()
		d.started = false
	}
}

// executeTask 执行任务（只记录状态）
func (d *Daemon) executeTask(task *Task) {
	// 创建执行日志
	log := &TaskLog{
		ID:        d.idGen.NextID(),
		TaskID:    task.ID,
		TaskName:  task.Name,
		StartTime: time.Now(),
		Status:    TaskStatusRunning,
	}
	d.DB.Create(log)

	// 执行命令
	cmd := exec.Command("sh", "-c", task.Command)
	err := cmd.Run()

	// 更新日志状态
	now := time.Now()
	log.EndTime = &now
	if err != nil {
		log.Status = TaskStatusFailed
	} else {
		log.Status = TaskStatusSuccess
	}

	d.DB.Save(log)
}

// ExecuteOnceTask 执行一次性/延迟任务（公开方法，供外部调用）
func (d *Daemon) ExecuteOnceTask(task *Task) {
	d.executeOnceTask(task)
}

// executeOnceTask 执行一次性/延迟任务（执行后标记为完成）
func (d *Daemon) executeOnceTask(task *Task) {
	// 如果是延迟任务，等待到指定时间
	if task.RunAt != nil {
		waitDuration := time.Until(*task.RunAt)
		if waitDuration > 0 {
			fmt.Printf("任务 %s 将在 %s 后执行\n", task.Name, waitDuration.Round(time.Second))
			time.Sleep(waitDuration)
		}
	}

	fmt.Printf("开始执行一次性任务: %s\n", task.Name)

	// 执行任务
	d.executeTask(task)

	// 执行完成后标记为已完成
	now := time.Now()
	if err := d.DB.Model(task).Updates(map[string]any{
		"completed":    true,
		"enabled":      false,
		"completed_at": now,
	}).Error; err != nil {
		fmt.Printf("标记任务完成失败: %v\n", err)
	} else {
		fmt.Printf("一次性任务 %s 执行完成\n", task.Name)
	}
}

// AddTask 添加任务（必须有调度）
func (d *Daemon) AddTask(name, command, schedule string) error {
	return d.AddTaskWithRunAt(name, command, schedule, nil)
}

// AddTaskWithRunAt 添加任务（支持指定执行时间）
func (d *Daemon) AddTaskWithRunAt(name, command, schedule string, runAt *time.Time) error {
	if schedule == "" {
		return fmt.Errorf("调度表达式不能为空")
	}

	task := &Task{
		ID:       d.idGen.NextID(),
		Name:     name,
		Command:  command,
		Schedule: schedule,
		Enabled:  true,
		RunAt:    runAt,
	}
	return d.DB.Create(task).Error
}

// RemoveTask 删除任务
func (d *Daemon) RemoveTask(name string) error {
	return d.DB.Where("name = ?", name).Delete(&Task{}).Error
}

// EnableTask 启用任务
func (d *Daemon) EnableTask(name string) error {
	return d.DB.Model(&Task{}).Where("name = ?", name).Update("enabled", true).Error
}

// DisableTask 禁用任务
func (d *Daemon) DisableTask(name string) error {
	return d.DB.Model(&Task{}).Where("name = ?", name).Update("enabled", false).Error
}

// ListTasks 列出所有任务
func (d *Daemon) ListTasks() ([]Task, error) {
	var tasks []Task
	err := d.DB.Find(&tasks).Error
	return tasks, err
}

// GetTask 获取任务
func (d *Daemon) GetTask(name string) (*Task, error) {
	var task Task
	err := d.DB.Where("name = ?", name).First(&task).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("任务不存在: %s", name)
	}
	return &task, err
}

// GetTaskByID 根据ID获取任务
func (d *Daemon) GetTaskByID(id int64) (*Task, error) {
	var task Task
	err := d.DB.Where("id = ?", id).First(&task).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("任务不存在: %d", id)
	}
	return &task, err
}

// ListLogs 列出任务日志
func (d *Daemon) ListLogs(taskName string, limit int) ([]TaskLog, error) {
	query := d.DB.Order("start_time DESC")
	if taskName != "" {
		query = query.Where("task_name = ?", taskName)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	var logs []TaskLog
	err := query.Find(&logs).Error
	return logs, err
}

// Close 关闭
func (d *Daemon) Close() error {
	d.Stop()
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetScheduler 获取调度器（用于信号处理）
func (d *Daemon) GetScheduler() *scheduler.Scheduler {
	return d.scheduler
}
