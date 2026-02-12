package gormx

import (
	"time"

	"gorm.io/gorm"
)

// Paginate 分页 Scope
func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 10
		}
		if pageSize > 100 {
			pageSize = 100 // 限制最大分页大小
		}

		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

// WithoutDeleted 排除软删除记录
func WithoutDeleted() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("deleted_at IS NULL")
	}
}

// OnlyDeleted 只查询软删除记录
func OnlyDeleted() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Unscoped().Where("deleted_at IS NOT NULL")
	}
}

// OrderByID 按 ID 排序
func OrderByID(desc bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if desc {
			return db.Order("id DESC")
		}
		return db.Order("id ASC")
	}
}

// OrderByCreatedAt 按创建时间排序
func OrderByCreatedAt(desc bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if desc {
			return db.Order("created_at DESC")
		}
		return db.Order("created_at ASC")
	}
}

// OrderByUpdatedAt 按更新时间排序
func OrderByUpdatedAt(desc bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if desc {
			return db.Order("updated_at DESC")
		}
		return db.Order("updated_at ASC")
	}
}

// TimeRange 时间范围查询
func TimeRange(field string, start, end time.Time) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if !start.IsZero() {
			db = db.Where(field+" >= ?", start)
		}
		if !end.IsZero() {
			db = db.Where(field+" <= ?", end)
		}
		return db
	}
}

// InIDs 按 ID 列表查询
func InIDs(ids []int64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(ids) == 0 {
			return db.Where("1 = 0") // 空列表返回空结果
		}
		return db.Where("id IN ?", ids)
	}
}

// Search 模糊搜索
func Search(field, keyword string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if keyword == "" {
			return db
		}
		return db.Where(field+" LIKE ?", "%"+keyword+"%")
	}
}

// Status 状态过滤
func Status(status string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if status == "" {
			return db
		}
		return db.Where("status = ?", status)
	}
}

// Active 只查询激活状态
func Active() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("is_active = ?", true)
	}
}

// SelectFields 选择指定字段
func SelectFields(fields ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(fields) == 0 {
			return db
		}
		return db.Select(fields)
	}
}

// OmitFields 排除指定字段
func OmitFields(fields ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(fields) == 0 {
			return db
		}
		return db.Omit(fields...)
	}
}
