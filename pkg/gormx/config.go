package gormx

import (
	"time"
)

// Config GORM 配置
type Config struct {
	// 数据库类型: mysql, postgres, sqlite
	Driver string `json:"driver" yaml:"driver"`

	// 主库连接
	// - 单库/主从：必填，实际使用的主库地址
	// - 多数据库：可选，留空则自动使用第一个数据库的 DSN
	DSN          string        `json:"dsn" yaml:"dsn"`
	MaxOpenConns int           `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	MaxLifetime  time.Duration `json:"max_lifetime" yaml:"max_lifetime"`
	MaxIdleTime  time.Duration `json:"max_idle_time" yaml:"max_idle_time"`

	// 日志配置
	LogLevel       string        `json:"log_level" yaml:"log_level"` // silent, error, warn, info
	SlowThreshold  time.Duration `json:"slow_threshold" yaml:"slow_threshold"`
	IgnoreNotFound bool          `json:"ignore_not_found" yaml:"ignore_not_found"`
	ColorfulLog    bool          `json:"colorful_log" yaml:"colorful_log"`

	// 性能配置
	PrepareStmt            bool `json:"prepare_stmt" yaml:"prepare_stmt"`
	DisableNestedTx        bool `json:"disable_nested_tx" yaml:"disable_nested_tx"`
	AllowGlobalUpdate      bool `json:"allow_global_update" yaml:"allow_global_update"`
	DisableAutomaticPing   bool `json:"disable_automatic_ping" yaml:"disable_automatic_ping"`
	DisableForeignKeyCheck bool `json:"disable_foreign_key_check" yaml:"disable_foreign_key_check"`

	// 高级配置（可选）
	replica  *ReplicaConfig
	multiDB  *MultiDatabaseConfig
	sharding *ShardingConfig
}

// ReplicaConfig 主从配置
type ReplicaConfig struct {
	// 从库地址（VIP/域名）
	ReplicaDSN string `json:"replica_dsn" yaml:"replica_dsn"`
}

// MultiDatabaseConfig 多数据库配置（按表名分库）
type MultiDatabaseConfig struct {
	// 数据库列表（每个数据库独立，必须指定表名）
	Databases []DatabaseConfig `json:"databases" yaml:"databases"`
}

// DatabaseConfig 单个数据库配置
type DatabaseConfig struct {
	// 数据库名称
	Name string `json:"name" yaml:"name"`

	// 表名匹配规则（必填，支持通配符，如 "orders_*"）
	Tables []string `json:"tables" yaml:"tables"`

	// 主库地址（VIP/域名）
	DSN string `json:"dsn" yaml:"dsn"`

	// 从库地址（可选）
	ReplicaDSN string `json:"replica_dsn,omitempty" yaml:"replica_dsn,omitempty"`
}

// NewConfig 创建配置（推荐使用）
// 注意：多数据库模式下，driver 参数仍需提供，dsn 参数会被忽略
func NewConfig(driver, dsn string) *Config {
	return &Config{
		Driver:                 driver,
		DSN:                    dsn,
		MaxOpenConns:           100,
		MaxIdleConns:           10,
		MaxLifetime:            time.Hour,
		MaxIdleTime:            10 * time.Minute,
		LogLevel:               "warn",
		SlowThreshold:          200 * time.Millisecond,
		IgnoreNotFound:         false,
		ColorfulLog:            true,
		PrepareStmt:            true,
		DisableNestedTx:        false,
		AllowGlobalUpdate:      false,
		DisableAutomaticPing:   false,
		DisableForeignKeyCheck: false,
	}
}

// DefaultConfig 默认配置（已废弃，使用 NewConfig 代替）
// Deprecated: 使用 NewConfig(driver, dsn) 代替
func DefaultConfig() *Config {
	return &Config{
		Driver:                 "mysql",
		MaxOpenConns:           100,
		MaxIdleConns:           10,
		MaxLifetime:            time.Hour,
		MaxIdleTime:            10 * time.Minute,
		LogLevel:               "warn",
		SlowThreshold:          200 * time.Millisecond,
		IgnoreNotFound:         false,
		ColorfulLog:            true,
		PrepareStmt:            true,
		DisableNestedTx:        false,
		AllowGlobalUpdate:      false,
		DisableAutomaticPing:   false,
		DisableForeignKeyCheck: false,
	}
}

// WithReplica 配置主从读写分离
func (c *Config) WithReplica(replicaDSN string) *Config {
	c.replica = &ReplicaConfig{
		ReplicaDSN: replicaDSN,
	}
	return c
}

// WithMultiDatabase 配置多数据库（按表名分库）
// 注意：多数据库模式下，Config.DSN 和 Config.replica 会被忽略
func (c *Config) WithMultiDatabase(databases []DatabaseConfig) *Config {
	c.multiDB = &MultiDatabaseConfig{
		Databases: databases,
	}
	return c
}

// HasReplica 是否配置了主从
func (c *Config) HasReplica() bool {
	return c.replica != nil && c.replica.ReplicaDSN != ""
}

// HasMultiDatabase 是否配置了多数据库
func (c *Config) HasMultiDatabase() bool {
	return c.multiDB != nil && len(c.multiDB.Databases) > 0
}
