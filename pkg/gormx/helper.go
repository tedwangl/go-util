package gormx

import (
	"gorm.io/gorm"
)

// PageResult 分页结果
type PageResult struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Data     any   `json:"data"`
}

// FindWithPage 分页查询
func FindWithPage(db *gorm.DB, page, pageSize int, dest any) (*PageResult, error) {
	var total int64

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	if err := db.Scopes(Paginate(page, pageSize)).Find(dest).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Data:     dest,
	}, nil
}

// Exists 检查记录是否存在
func Exists(db *gorm.DB) (bool, error) {
	var count int64
	err := db.Count(&count).Error
	return count > 0, err
}

// BatchCreate 批量创建（分批插入）
func BatchCreate(db *gorm.DB, data any, batchSize int) error {
	return db.CreateInBatches(data, batchSize).Error
}

// BatchUpdate 批量更新
func BatchUpdate(db *gorm.DB, ids []int64, updates map[string]any) error {
	if len(ids) == 0 {
		return nil
	}
	return db.Where("id IN ?", ids).Updates(updates).Error
}

// BatchDelete 批量删除
func BatchDelete(db *gorm.DB, ids []int64, model any) error {
	if len(ids) == 0 {
		return nil
	}
	return db.Where("id IN ?", ids).Delete(model).Error
}

// SoftDelete 软删除
func SoftDelete(db *gorm.DB, id int64, model any) error {
	return db.Where("id = ?", id).Delete(model).Error
}

// HardDelete 硬删除
func HardDelete(db *gorm.DB, id int64, model any) error {
	return db.Unscoped().Where("id = ?", id).Delete(model).Error
}

// Restore 恢复软删除记录
func Restore(db *gorm.DB, id int64, model any) error {
	return db.Unscoped().Model(model).Where("id = ?", id).Update("deleted_at", nil).Error
}

// FirstOrCreate 查找或创建
func FirstOrCreate(db *gorm.DB, where any, dest any) error {
	return db.Where(where).FirstOrCreate(dest).Error
}

// UpdateOrCreate 更新或创建
func UpdateOrCreate(db *gorm.DB, where any, updates any, dest any) error {
	return db.Where(where).Assign(updates).FirstOrCreate(dest).Error
}
