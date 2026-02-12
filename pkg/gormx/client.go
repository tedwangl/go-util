package gormx

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Client GORM 客户端封装
type Client struct {
	*gorm.DB
	config *Config
}

// NewClient 创建 GORM 客户端
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 配置日志
	logConfig := logger.Config{
		SlowThreshold:             cfg.SlowThreshold,
		LogLevel:                  parseLogLevel(cfg.LogLevel),
		IgnoreRecordNotFoundError: cfg.IgnoreNotFound,
		Colorful:                  cfg.ColorfulLog,
	}

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logConfig,
	)

	// GORM 配置
	gormConfig := &gorm.Config{
		Logger:                                   gormLogger,
		PrepareStmt:                              cfg.PrepareStmt,
		DisableNestedTransaction:                 cfg.DisableNestedTx,
		AllowGlobalUpdate:                        cfg.AllowGlobalUpdate,
		DisableAutomaticPing:                     cfg.DisableAutomaticPing,
		DisableForeignKeyConstraintWhenMigrating: cfg.DisableForeignKeyCheck,
	}

	// 根据驱动类型创建主库连接
	dialector, err := createPrimaryDialector(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.MaxIdleTime)

	client := &Client{
		DB:     db,
		config: cfg,
	}

	// 配置 DBResolver（主从 + 分库）
	if err := client.setupDBResolver(cfg); err != nil {
		return nil, fmt.Errorf("failed to setup dbresolver: %w", err)
	}

	return client, nil
}

// GetDB 获取原始 *gorm.DB
func (c *Client) GetDB() *gorm.DB {
	return c.DB
}

// GetSQLDB 获取原始 *sql.DB
func (c *Client) GetSQLDB() (*sql.DB, error) {
	return c.DB.DB()
}

// Close 关闭数据库连接
func (c *Client) Close() error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping 测试数据库连接
func (c *Client) Ping() error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// Stats 获取连接池状态
func (c *Client) Stats() sql.DBStats {
	sqlDB, _ := c.DB.DB()
	return sqlDB.Stats()
}

// parseLogLevel 解析日志级别
func parseLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Warn
	}
}

// createPrimaryDialector 创建主库 Dialector
func createPrimaryDialector(driver, dsn string) (gorm.Dialector, error) {
	switch driver {
	case "mysql":
		return mysql.Open(dsn), nil
	case "postgres":
		return postgres.Open(dsn), nil
	case "sqlite":
		return sqlite.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported driver: %s (支持: mysql, postgres, sqlite)", driver)
	}
}

// createMySQLDialector 创建 MySQL Dialector
func (c *Client) createMySQLDialector(dsn string) gorm.Dialector {
	return mysql.Open(dsn)
}

// createPostgresDialector 创建 PostgreSQL Dialector
func (c *Client) createPostgresDialector(dsn string) gorm.Dialector {
	return postgres.Open(dsn)
}

// createSQLiteDialector 创建 SQLite Dialector
func (c *Client) createSQLiteDialector(dsn string) gorm.Dialector {
	return sqlite.Open(dsn)
}
