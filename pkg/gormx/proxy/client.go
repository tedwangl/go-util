package proxy

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/sharding"
)

// Client ProxySQL + Sharding 客户端
type Client struct {
	*gorm.DB
	config *Config
}

// NewClient 创建客户端
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 配置日志
	logConfig := logger.Config{
		SlowThreshold:             cfg.SlowThreshold,
		LogLevel:                  parseLogLevel(cfg.LogLevel),
		IgnoreRecordNotFoundError: true,
		Colorful:                  true,
	}

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logConfig,
	)

	// GORM 配置
	gormConfig := &gorm.Config{
		Logger: gormLogger,
	}

	var db *gorm.DB
	var err error

	switch cfg.Mode {
	case "single":
		db, err = newSingleDB(cfg, gormConfig)
	case "master-slave":
		db, err = newMasterSlaveDB(cfg, gormConfig)
	case "sharding":
		db, err = newShardingDB(cfg, gormConfig)
	default:
		return nil, fmt.Errorf("unsupported mode: %s", cfg.Mode)
	}

	if err != nil {
		return nil, err
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

	return &Client{
		DB:     db,
		config: cfg,
	}, nil
}

// newSingleDB 创建单库连接
func newSingleDB(cfg *Config, gormConfig *gorm.Config) (*gorm.DB, error) {
	if cfg.ProxySQL == nil {
		return nil, fmt.Errorf("proxysql config is required for single mode")
	}

	dsn := buildDSN(cfg.ProxySQL)
	return gorm.Open(mysql.Open(dsn), gormConfig)
}

// newMasterSlaveDB 创建主从连接（通过 ProxySQL）
func newMasterSlaveDB(cfg *Config, gormConfig *gorm.Config) (*gorm.DB, error) {
	if cfg.ProxySQL == nil {
		return nil, fmt.Errorf("proxysql config is required for master-slave mode")
	}

	// ProxySQL 自动处理主从路由，应用层无需配置
	dsn := buildDSN(cfg.ProxySQL)
	return gorm.Open(mysql.Open(dsn), gormConfig)
}

// newShardingDB 创建分片连接
func newShardingDB(cfg *Config, gormConfig *gorm.Config) (*gorm.DB, error) {
	if cfg.ProxySQL == nil {
		return nil, fmt.Errorf("proxysql config is required for sharding mode")
	}

	if cfg.Sharding == nil || !cfg.Sharding.Enable {
		return nil, fmt.Errorf("sharding config is required for sharding mode")
	}

	// 连接 ProxySQL
	dsn := buildDSN(cfg.ProxySQL)
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, err
	}

	// 配置分片中间件
	shardingConfig := buildShardingConfig(cfg.Sharding)
	if err := db.Use(shardingConfig); err != nil {
		return nil, fmt.Errorf("failed to use sharding plugin: %w", err)
	}

	return db, nil
}

// buildDSN 构建 DSN
func buildDSN(cfg *ProxySQLConfig) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Addr,
		cfg.Database,
	)
}

// buildShardingConfig 构建分片配置
func buildShardingConfig(cfg *ShardingConfig) *sharding.Sharding {
	shardingConfig := sharding.Config{
		ShardingKey:           "_id",
		NumberOfShards:        uint(cfg.ShardCount),
		PrimaryKeyGenerator:   sharding.PKSnowflake,
		PrimaryKeyGeneratorFn: nil,
	}

	// 使用第一个表的配置
	if len(cfg.Tables) > 0 {
		table := cfg.Tables[0]
		shardingConfig.ShardingKey = table.ShardingKey
		if table.ShardCount > 0 {
			shardingConfig.NumberOfShards = uint(table.ShardCount)
		}
	}

	return sharding.Register(shardingConfig)
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

// Close 关闭连接
func (c *Client) Close() error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
