package gormx

import (
	"time"
)

// Config GORM 配置
type Config struct {
	// 数据库类型: mysql, postgres, sqlite
	Driver string `json:"driver" yaml:"driver"`

	// 连接配置
	DSN          string        `json:"dsn" yaml:"dsn"`
	MaxOpenConns int           `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	MaxLifetime  time.Duration `json:"max_lifetime" yaml:"max_lifetime"`
	MaxIdleTime  time.Duration `json:"max_idle_time" yaml:"max_idle_time"`

	// 主从配置（使用 DBResolver）
	Replicas []string `json:"replicas,omitempty" yaml:"replicas,omitempty"` // 从库 DSN 列表
	Sources  []string `json:"sources,omitempty" yaml:"sources,omitempty"`   // 额外主库 DSN 列表（多主）

	// 分库配置
	Shards []ShardConfig `json:"shards,omitempty" yaml:"shards,omitempty"`

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
}

// DefaultConfig 默认配置
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

// ShardConfig 分库配置
type ShardConfig struct {
	// 分片名称（用于表名匹配）
	Name string `json:"name" yaml:"name"`

	// 表名匹配规则（支持通配符，如 "shard1_*"）
	Tables []string `json:"tables" yaml:"tables"`

	// 主库 DSN
	Sources []string `json:"sources" yaml:"sources"`

	// 从库 DSN
	Replicas []string `json:"replicas,omitempty" yaml:"replicas,omitempty"`
}
