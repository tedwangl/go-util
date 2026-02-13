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
		ID        int64     `gorm:"primarykey" json:"id"`             // 雪花ID
		Name      string    `gorm:"uniqueIndex;not null" json:"name"` // 任务名称
		Command   string    `gorm:"not null" json:"command"`          // 执行命令
		Schedule  string    `gorm:"default:''" json:"schedule"`       // cron 表达式（空表示不调度）
		Enabled   bool      `gorm:"default:true" json:"enabled"`      // 是否启用（仅调度任务有效）
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
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
		db        *gorm.DB
		scheduler *scheduler.Scheduler
		dbPath    string
		idGen     *genid.SnowflakeID
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
		db:        db,
		scheduler: scheduler.NewScheduler(),
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
	return nil
}

// loadTasks 加载所有任务到调度器
func (d *Daemon) loadTasks() error {
	// 加载所有启用的调度任务
	var tasks []Task
	if err := d.db.Where("enabled = ? AND schedule != ''", true).Find(&tasks).Error; err != nil {
		return fmt.Errorf("加载任务失败: %w", err)
	}

	// 注册任务到调度器
	for _, task := range tasks {
		t := task // 避免闭包问题
		if err := d.scheduler.AddFunc(t.Schedule, t.Name, func() error {
			d.executeTask(&t)
			return nil
		}); err != nil {
			return fmt.Errorf("注册任务 %s 失败: %w", t.Name, err)
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
	t := *task // 复制值，避免闭包问题
	return d.scheduler.AddFunc(t.Schedule, t.Name, func() error {
		d.executeTask(&t)
		return nil
	})
}

// Stop 停止守护进程
func (d *Daemon) Stop() {
	d.scheduler.Stop()
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
	d.db.Create(log)

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

	d.db.Save(log)
}

// AddTask 添加任务（必须有调度）
func (d *Daemon) AddTask(name, command, schedule string) error {
	if schedule == "" {
		return fmt.Errorf("调度表达式不能为空")
	}

	task := &Task{
		ID:       d.idGen.NextID(),
		Name:     name,
		Command:  command,
		Schedule: schedule,
		Enabled:  true,
	}
	return d.db.Create(task).Error
}

// RemoveTask 删除任务
func (d *Daemon) RemoveTask(name string) error {
	return d.db.Where("name = ?", name).Delete(&Task{}).Error
}

// EnableTask 启用任务
func (d *Daemon) EnableTask(name string) error {
	return d.db.Model(&Task{}).Where("name = ?", name).Update("enabled", true).Error
}

// DisableTask 禁用任务
func (d *Daemon) DisableTask(name string) error {
	return d.db.Model(&Task{}).Where("name = ?", name).Update("enabled", false).Error
}

// ListTasks 列出所有任务
func (d *Daemon) ListTasks() ([]Task, error) {
	var tasks []Task
	err := d.db.Find(&tasks).Error
	return tasks, err
}

// GetTask 获取任务
func (d *Daemon) GetTask(name string) (*Task, error) {
	var task Task
	err := d.db.Where("name = ?", name).First(&task).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("任务不存在: %s", name)
	}
	return &task, err
}

// GetTaskByID 根据ID获取任务
func (d *Daemon) GetTaskByID(id int64) (*Task, error) {
	var task Task
	err := d.db.Where("id = ?", id).First(&task).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("任务不存在: %d", id)
	}
	return &task, err
}

// ListLogs 列出任务日志
func (d *Daemon) ListLogs(taskName string, limit int) ([]TaskLog, error) {
	query := d.db.Order("start_time DESC")
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
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
