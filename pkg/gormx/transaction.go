package gormx

import (
	"context"

	"gorm.io/gorm"
)

// Transaction 事务辅助函数
func (c *Client) Transaction(fn func(tx *gorm.DB) error) error {
	return c.DB.Transaction(fn)
}

// TransactionWithContext 带上下文的事务
func (c *Client) TransactionWithContext(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return c.DB.WithContext(ctx).Transaction(fn)
}

// BeginTx 手动开启事务
func (c *Client) BeginTx(ctx context.Context) *gorm.DB {
	return c.DB.WithContext(ctx).Begin()
}

// Commit 提交事务
func Commit(tx *gorm.DB) error {
	return tx.Commit().Error
}

// Rollback 回滚事务
func Rollback(tx *gorm.DB) error {
	return tx.Rollback().Error
}
