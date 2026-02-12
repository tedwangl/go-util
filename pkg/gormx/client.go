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
	"gorm.io/plugin/dbresolver"
)

// Client GORM 客户端封装
type Client struct {
	*gorm.DB
	config *Config

	// 分片连接（如果启用了分片）
	shardDBs []*gorm.DB
}

// NewClient 创建 GORM 客户端
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	client := &Client{
		config:   cfg,
		shardDBs: make([]*gorm.DB, 0),
	}

	// 分片模式：直接初始化分片连接，不需要主连接
	if cfg.HasSharding() {
		if err := client.setupShardingConnections(cfg); err != nil {
			return nil, fmt.Errorf("failed to setup sharding: %w", err)
		}
		// 将第一个分片的 DB 设置为默认 DB（用于非分片操作）
		if len(client.shardDBs) > 0 {
			client.DB = client.shardDBs[0]
		}
		return client, nil
	}

	// 非分片模式：创建主连接
	// 确定主连接 DSN
	var primaryDSN string
	if cfg.HasMultiDatabase() {
		// 多数据库模式：完全忽略 Config.DSN，使用第一个数据库的 DSN
		if len(cfg.multiDB.Databases) == 0 {
			return nil, fmt.Errorf("multi-database mode requires at least one database")
		}
		primaryDSN = cfg.multiDB.Databases[0].DSN
		if primaryDSN == "" {
			return nil, fmt.Errorf("first database DSN cannot be empty in multi-database mode")
		}
	} else {
		// 单库/主从模式：使用 Config.DSN
		primaryDSN = cfg.DSN
		if primaryDSN == "" {
			return nil, fmt.Errorf("DSN cannot be empty")
		}
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
	dialector, err := createPrimaryDialector(cfg.Driver, primaryDSN)
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

	client.DB = db

	// 配置 DBResolver（主从 + 多数据库）
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
	// 关闭分片连接
	for _, shardDB := range c.shardDBs {
		if sqlDB, err := shardDB.DB(); err == nil {
			sqlDB.Close()
		}
	}

	// 关闭主连接
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

// Shard 指定分片进行操作（应用层提供分片键）
// 返回对应分片的 *gorm.DB 实例
// 用法：client.Shard(userID).Model(&User{}).Where("id = ?", userID).First(&user)
func (c *Client) Shard(shardKey interface{}) *gorm.DB {
	if c.config.sharding == nil || len(c.shardDBs) == 0 {
		return c.DB
	}

	shardID := c.config.ShardID(shardKey)
	if shardID < 0 || shardID >= len(c.shardDBs) {
		return c.DB // 返回默认连接
	}

	return c.shardDBs[shardID]
}

// ShardByID 直接指定分片 ID
func (c *Client) ShardByID(shardID int) *gorm.DB {
	if c.config.sharding == nil || len(c.shardDBs) == 0 {
		return c.DB
	}

	if shardID < 0 || shardID >= len(c.shardDBs) {
		return c.DB
	}

	return c.shardDBs[shardID]
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

// setupShardingConnections 初始化分片连接
func (c *Client) setupShardingConnections(cfg *Config) error {
	if cfg.sharding == nil || len(cfg.sharding.Shards) == 0 {
		return fmt.Errorf("sharding config is empty")
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

	// 为每个分片创建独立的 DB 实例
	c.shardDBs = make([]*gorm.DB, len(cfg.sharding.Shards))

	for i, shard := range cfg.sharding.Shards {
		// 创建主库连接
		dialector, err := createPrimaryDialector(cfg.Driver, shard.DSN)
		if err != nil {
			return fmt.Errorf("failed to create shard %d dialector: %w", shard.ID, err)
		}

		// 每个分片使用独立的 gormConfig 实例，避免共享状态
		shardGormConfig := &gorm.Config{
			Logger:                                   gormLogger,
			PrepareStmt:                              cfg.PrepareStmt,
			DisableNestedTransaction:                 cfg.DisableNestedTx,
			AllowGlobalUpdate:                        cfg.AllowGlobalUpdate,
			DisableAutomaticPing:                     cfg.DisableAutomaticPing,
			DisableForeignKeyConstraintWhenMigrating: cfg.DisableForeignKeyCheck,
		}

		db, err := gorm.Open(dialector, shardGormConfig)
		if err != nil {
			return fmt.Errorf("failed to connect shard %d: %w", shard.ID, err)
		}

		// 配置连接池
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("failed to get shard %d sql.DB: %w", shard.ID, err)
		}

		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)
		sqlDB.SetConnMaxIdleTime(cfg.MaxIdleTime)

		// 配置主从（如果有从库）
		// 注意：每个分片的 DB 实例是独立的，可以单独配置主从
		if shard.ReplicaDSN != "" {
			replicaDialector, err := createPrimaryDialector(cfg.Driver, shard.ReplicaDSN)
			if err != nil {
				return fmt.Errorf("failed to create shard %d replica dialector: %w", shard.ID, err)
			}

			// 为这个分片配置主从
			resolver := dbresolver.Register(dbresolver.Config{
				Replicas: []gorm.Dialector{replicaDialector},
				Policy:   dbresolver.RandomPolicy{},
			})

			if err := db.Use(resolver); err != nil {
				return fmt.Errorf("failed to register shard %d replica: %w", shard.ID, err)
			}
		}

		c.shardDBs[i] = db
	}

	return nil
}
