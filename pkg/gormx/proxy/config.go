package proxy

import (
	"time"
)

// Config ProxySQL + Sharding 配置
type Config struct {
	// 模式: single, master-slave, sharding
	Mode string `json:"mode" yaml:"mode"`

	// ProxySQL 连接配置
	ProxySQL *ProxySQLConfig `json:"proxysql,omitempty" yaml:"proxysql,omitempty"`

	// 分片配置
	Sharding *ShardingConfig `json:"sharding,omitempty" yaml:"sharding,omitempty"`

	// 通用配置
	MaxOpenConns int           `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	MaxLifetime  time.Duration `json:"max_lifetime" yaml:"max_lifetime"`
	MaxIdleTime  time.Duration `json:"max_idle_time" yaml:"max_idle_time"`

	// 日志配置
	LogLevel      string        `json:"log_level" yaml:"log_level"`
	SlowThreshold time.Duration `json:"slow_threshold" yaml:"slow_threshold"`
}

// ProxySQLConfig ProxySQL 配置
type ProxySQLConfig struct {
	// ProxySQL 地址（应用连接这个）
	Addr     string `json:"addr" yaml:"addr"` // 例如: "127.0.0.1:6033"
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Database string `json:"database" yaml:"database"`
}

// ShardingConfig 分片配置
type ShardingConfig struct {
	// 是否启用分片
	Enable bool `json:"enable" yaml:"enable"`

	// 分片数量
	ShardCount int `json:"shard_count" yaml:"shard_count"`

	// 分片表配置
	Tables []*ShardingTable `json:"tables" yaml:"tables"`

	// 分片算法: mod, range, hash
	Algorithm string `json:"algorithm" yaml:"algorithm"`
}

// ShardingTable 分片表配置
type ShardingTable struct {
	// 表名（逻辑表）
	TableName string `json:"table_name" yaml:"table_name"`

	// 分片键
	ShardingKey string `json:"sharding_key" yaml:"sharding_key"`

	// 分片数量（可选，默认使用全局配置）
	ShardCount int `json:"shard_count,omitempty" yaml:"shard_count,omitempty"`

	// 分片算法（可选，默认使用全局配置）
	Algorithm string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`

	// 是否生成主键
	GeneratePrimaryKey bool `json:"generate_primary_key" yaml:"generate_primary_key"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Mode:          "single",
		MaxOpenConns:  100,
		MaxIdleConns:  10,
		MaxLifetime:   time.Hour,
		MaxIdleTime:   10 * time.Minute,
		LogLevel:      "warn",
		SlowThreshold: 200 * time.Millisecond,
	}
}

// DefaultProxySQLConfig 默认 ProxySQL 配置
func DefaultProxySQLConfig() *ProxySQLConfig {
	return &ProxySQLConfig{
		Addr:     "127.0.0.1:6033",
		Username: "root",
		Password: "root123",
		Database: "testdb",
	}
}

// DefaultShardingConfig 默认分片配置
func DefaultShardingConfig() *ShardingConfig {
	return &ShardingConfig{
		Enable:     true,
		ShardCount: 4,
		Algorithm:  "mod",
		Tables:     []*ShardingTable{},
	}
}
