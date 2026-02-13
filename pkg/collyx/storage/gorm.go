package storage

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormStorage GORM 存储（支持 SQLite 和 MySQL）
type GormStorage struct {
	db *gorm.DB
}

// NewSQLiteStorage 创建 SQLite 存储
func NewSQLiteStorage(dbPath string) (*GormStorage, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	s := &GormStorage{db: db}
	if err := s.initTables(); err != nil {
		return nil, err
	}

	return s, nil
}

// NewMySQLStorage 创建 MySQL 存储
func NewMySQLStorage(dsn string) (*GormStorage, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	s := &GormStorage{db: db}
	if err := s.initTables(); err != nil {
		return nil, err
	}

	return s, nil
}

// initTables 初始化表
func (s *GormStorage) initTables() error {
	return s.db.AutoMigrate(&Task{}, &Item{})
}

// SaveTask 保存任务
func (s *GormStorage) SaveTask(task *Task) error {
	task.UpdatedAt = time.Now()
	return s.db.Save(task).Error
}

// GetTask 获取任务
func (s *GormStorage) GetTask(id string) (*Task, error) {
	var task Task
	err := s.db.Where("id = ?", id).First(&task).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("任务不存在: %s", id)
	}
	return &task, err
}

// GetTaskByURL 根据 URL 获取任务
func (s *GormStorage) GetTaskByURL(url string) (*Task, error) {
	var task Task
	err := s.db.Where("url = ?", url).First(&task).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("任务不存在: %s", url)
	}
	return &task, err
}

// GetTaskByURLHash 根据 URL 哈希获取任务
func (s *GormStorage) GetTaskByURLHash(hash string) (*Task, error) {
	var task Task
	err := s.db.Where("url_hash = ?", hash).First(&task).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("任务不存在: %s", hash)
	}
	return &task, err
}

// UpdateTask 更新任务
func (s *GormStorage) UpdateTask(task *Task) error {
	return s.SaveTask(task)
}

// DeleteTask 删除任务
func (s *GormStorage) DeleteTask(id string) error {
	return s.db.Where("id = ?", id).Delete(&Task{}).Error
}

// applyTaskFilter 应用任务过滤条件（GORM Scope）
func applyTaskFilter(filter *TaskFilter) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(filter.Status) > 0 {
			db = db.Where("status IN ?", filter.Status)
		}
		if filter.Priority != nil {
			db = db.Where("priority = ?", *filter.Priority)
		}
		return db
	}
}

// ListTasks 列出任务
func (s *GormStorage) ListTasks(filter *TaskFilter) ([]*Task, error) {
	query := s.db.Model(&Task{}).Scopes(applyTaskFilter(filter))

	// 排序
	if filter.OrderBy != "" {
		order := filter.OrderBy
		if filter.OrderDesc {
			order += " DESC"
		}
		query = query.Order(order)
	}

	// 分页
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	var tasks []*Task
	err := query.Find(&tasks).Error
	return tasks, err
}

// CountTasks 统计任务数
func (s *GormStorage) CountTasks(filter *TaskFilter) (int64, error) {
	var count int64
	err := s.db.Model(&Task{}).Scopes(applyTaskFilter(filter)).Count(&count).Error
	return count, err
}

// SaveTasks 批量保存
func (s *GormStorage) SaveTasks(tasks []*Task) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, task := range tasks {
			task.UpdatedAt = time.Now()
			if err := tx.Save(task).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateTaskStatus 更新任务状态
func (s *GormStorage) UpdateTaskStatus(id string, status TaskStatus) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}

	// 完成或失败时记录完成时间
	if status == TaskStatusCompleted || status == TaskStatusFailed || status == TaskStatusSkipped {
		now := time.Now()
		updates["completed_at"] = &now
	}

	return s.db.Model(&Task{}).Where("id = ?", id).Updates(updates).Error
}

// GetProgress 获取进度
func (s *GormStorage) GetProgress() (*Progress, error) {
	progress := &Progress{
		UpdatedAt: time.Now(),
	}

	// 统计各状态任务数
	type Result struct {
		Total     int64
		Completed int64
		Failed    int64
		Pending   int64
		Running   int64
	}

	var result Result
	err := s.db.Model(&Task{}).Select(`
		COUNT(*) as total,
		SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as completed,
		SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as failed,
		SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as pending,
		SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as running
	`, TaskStatusCompleted, TaskStatusFailed, TaskStatusPending, TaskStatusRunning).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	progress.Total = result.Total
	progress.Completed = result.Completed
	progress.Failed = result.Failed
	progress.Pending = result.Pending
	progress.Running = result.Running

	// 获取最早的任务创建时间作为开始时间
	var firstTask Task
	if err := s.db.Order("created_at ASC").First(&firstTask).Error; err == nil {
		progress.StartTime = firstTask.CreatedAt
	} else {
		progress.StartTime = time.Now()
	}

	return progress, nil
}

// applyItemFilter 应用内容过滤条件（GORM Scope）
func applyItemFilter(filter *ItemFilter) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if filter.TaskID != "" {
			db = db.Where("task_id = ?", filter.TaskID)
		}
		if len(filter.Type) > 0 {
			db = db.Where("type IN ?", filter.Type)
		}
		if len(filter.Status) > 0 {
			db = db.Where("status IN ?", filter.Status)
		}
		if filter.ContentHash != "" {
			db = db.Where("content_hash = ?", filter.ContentHash)
		}
		return db
	}
}

// SaveItem 保存内容
func (s *GormStorage) SaveItem(item *Item) error {
	item.UpdatedAt = time.Now()
	return s.db.Save(item).Error
}

// GetItem 获取内容
func (s *GormStorage) GetItem(id string) (*Item, error) {
	var item Item
	err := s.db.Where("id = ?", id).First(&item).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("内容不存在: %s", id)
	}
	return &item, err
}

// GetItemByContentHash 根据内容哈希获取
func (s *GormStorage) GetItemByContentHash(hash string) (*Item, error) {
	var item Item
	err := s.db.Where("content_hash = ?", hash).First(&item).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("内容不存在: %s", hash)
	}
	return &item, err
}

// ListItems 列出内容
func (s *GormStorage) ListItems(filter *ItemFilter) ([]*Item, error) {
	query := s.db.Model(&Item{}).Scopes(applyItemFilter(filter))

	// 排序
	if filter.OrderBy != "" {
		order := filter.OrderBy
		if filter.OrderDesc {
			order += " DESC"
		}
		query = query.Order(order)
	}

	// 分页
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	var items []*Item
	err := query.Find(&items).Error
	return items, err
}

// CountItems 统计内容数
func (s *GormStorage) CountItems(filter *ItemFilter) (int64, error) {
	var count int64
	err := s.db.Model(&Item{}).Scopes(applyItemFilter(filter)).Count(&count).Error
	return count, err
}

// UpdateItemStatus 更新内容状态
func (s *GormStorage) UpdateItemStatus(id string, status ItemStatus) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}
	return s.db.Model(&Item{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteItem 删除内容
func (s *GormStorage) DeleteItem(id string) error {
	return s.db.Where("id = ?", id).Delete(&Item{}).Error
}

// Clear 清空所有数据
func (s *GormStorage) Clear() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&Task{}).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&Item{}).Error; err != nil {
			return err
		}
		return nil
	})
}

// Close 关闭
func (s *GormStorage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
