package storage

import (
	"time"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 待处理
	TaskStatusRunning   TaskStatus = "running"   // 运行中
	TaskStatusCompleted TaskStatus = "completed" // 已完成
	TaskStatusFailed    TaskStatus = "failed"    // 失败
	TaskStatusPaused    TaskStatus = "paused"    // 暂停
	TaskStatusSkipped   TaskStatus = "skipped"   // 跳过
)

// ItemType 内容类型
type ItemType string

const (
	ItemTypeHTML  ItemType = "html"  // 网页
	ItemTypeImage ItemType = "image" // 图片
	ItemTypeFile  ItemType = "file"  // 文件
	ItemTypeData  ItemType = "data"  // 数据
)

// ItemStatus 内容状态
type ItemStatus string

const (
	ItemStatusPending ItemStatus = "pending" // 待处理
	ItemStatusSaved   ItemStatus = "saved"   // 已保存
	ItemStatusFailed  ItemStatus = "failed"  // 失败
	ItemStatusSkipped ItemStatus = "skipped" // 跳过
)

// DuplicateStrategy 去重策略
type DuplicateStrategy string

const (
	DuplicateStrategyURL         DuplicateStrategy = "url"          // URL 去重
	DuplicateStrategyURLHash     DuplicateStrategy = "url_hash"     // URL 哈希去重
	DuplicateStrategyContentHash DuplicateStrategy = "content_hash" // 内容哈希去重
	DuplicateStrategyNone        DuplicateStrategy = "none"         // 不去重
)

// Task 爬虫任务
type Task struct {
	ID          string         `json:"id" gorm:"primaryKey;size:255"`
	URL         string         `json:"url" gorm:"type:text;index:idx_url"`
	URLHash     string         `json:"url_hash" gorm:"size:32;index:idx_url_hash"` // URL 哈希（用于快速去重）
	Method      string         `json:"method" gorm:"size:10"`
	Priority    int            `json:"priority" gorm:"index:idx_priority"`
	Depth       int            `json:"depth"`
	Status      TaskStatus     `json:"status" gorm:"size:20;index:idx_status"`
	Retries     int            `json:"retries"`
	MaxRetries  int            `json:"max_retries"`
	Error       string         `json:"error,omitempty" gorm:"type:text"`
	Metadata    map[string]any `json:"metadata,omitempty" gorm:"serializer:json;type:text"`
	CreatedAt   time.Time      `json:"created_at" gorm:"index"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}

// Item 爬取内容
type Item struct {
	ID          string         `json:"id" gorm:"primaryKey;size:255"`
	TaskID      string         `json:"task_id" gorm:"size:255;index:idx_task_id"`
	URL         string         `json:"url" gorm:"type:text"`
	Type        ItemType       `json:"type" gorm:"size:20;index:idx_type"`
	Status      ItemStatus     `json:"status" gorm:"size:20;index:idx_status"`
	Title       string         `json:"title,omitempty" gorm:"type:text"`
	Content     string         `json:"content,omitempty" gorm:"type:text"`
	FilePath    string         `json:"file_path,omitempty" gorm:"type:text"`
	ContentHash string         `json:"content_hash,omitempty" gorm:"size:64;index:idx_content_hash"`
	Size        int64          `json:"size"`
	Error       string         `json:"error,omitempty" gorm:"type:text"`
	Metadata    map[string]any `json:"metadata,omitempty" gorm:"serializer:json;type:text"`
	CreatedAt   time.Time      `json:"created_at" gorm:"index"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// Progress 进度信息
type Progress struct {
	Total       int64      `json:"total"`
	Completed   int64      `json:"completed"`
	Failed      int64      `json:"failed"`
	Pending     int64      `json:"pending"`
	Running     int64      `json:"running"`
	StartTime   time.Time  `json:"start_time"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// TaskFilter 任务过滤条件
type TaskFilter struct {
	Status    []TaskStatus // 状态过滤
	Priority  *int         // 优先级过滤
	Limit     int          // 限制数量
	Offset    int          // 偏移量
	OrderBy   string       // 排序字段
	OrderDesc bool         // 是否降序
}

// ItemFilter 内容过滤条件
type ItemFilter struct {
	TaskID      string       // 任务 ID
	Type        []ItemType   // 类型过滤
	Status      []ItemStatus // 状态过滤
	ContentHash string       // 内容哈希
	Limit       int          // 限制数量
	Offset      int          // 偏移量
	OrderBy     string       // 排序字段
	OrderDesc   bool         // 是否降序
}

// Storage 存储接口
type Storage interface {
	// 任务管理
	SaveTask(task *Task) error                     // 保存任务
	GetTask(id string) (*Task, error)              // 获取任务
	GetTaskByURL(url string) (*Task, error)        // 根据 URL 获取任务（去重）
	GetTaskByURLHash(hash string) (*Task, error)   // 根据 URL 哈希获取任务（去重）
	UpdateTask(task *Task) error                   // 更新任务
	DeleteTask(id string) error                    // 删除任务
	ListTasks(filter *TaskFilter) ([]*Task, error) // 列出任务
	CountTasks(filter *TaskFilter) (int64, error)  // 统计任务数

	// 批量操作
	SaveTasks(tasks []*Task) error                       // 批量保存
	UpdateTaskStatus(id string, status TaskStatus) error // 更新状态

	// 内容管理
	SaveItem(item *Item) error                           // 保存内容
	GetItem(id string) (*Item, error)                    // 获取内容
	GetItemByContentHash(hash string) (*Item, error)     // 根据内容哈希获取（去重）
	UpdateItemStatus(id string, status ItemStatus) error // 更新内容状态
	ListItems(filter *ItemFilter) ([]*Item, error)       // 列出内容
	CountItems(filter *ItemFilter) (int64, error)        // 统计内容数
	DeleteItem(id string) error                          // 删除内容

	// 进度管理（通过统计 Task 表得出）
	GetProgress() (*Progress, error) // 获取进度

	// 清理
	Clear() error // 清空所有数据
	Close() error // 关闭连接
}
