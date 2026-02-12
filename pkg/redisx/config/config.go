package config

import (
	"time"

	redisxerrors "github.com/tedwangl/go-util/pkg/redisx/errors"
)

// Config 是RedisX的主配置结构
type Config struct {
	Mode  string // 部署模式: single, sentinel, cluster, multi-master
	Debug bool   // 是否开启调试模式

	// 单节点配置
	Single *SingleConfig `json:"single,omitempty" yaml:"single,omitempty"`

	// 哨兵配置
	Sentinel *SentinelConfig `json:"sentinel,omitempty" yaml:"sentinel,omitempty"`

	// 集群配置
	Cluster *ClusterConfig `json:"cluster,omitempty" yaml:"cluster,omitempty"`

	// 多主多从配置
	MultiMaster *MultiMasterConfig `json:"multi-master,omitempty" yaml:"multi-master,omitempty"`

	// 通用配置
	Username     string        `json:"username,omitempty" yaml:"username,omitempty"` // Redis 6.0+ ACL 用户名
	Password     string        `json:"password,omitempty" yaml:"password,omitempty"`
	DB           int           `json:"db" yaml:"db"`
	PoolSize     int           `json:"pool_size" yaml:"pool_size"`
	MinIdleConns int           `json:"min_idle_conns" yaml:"min_idle_conns"`
	MaxRetries   int           `json:"max_retries" yaml:"max_retries"`
	DialTimeout  time.Duration `json:"dial_timeout" yaml:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
	PoolTimeout  time.Duration `json:"pool_timeout" yaml:"pool_timeout"`
}

// SingleConfig 单节点配置
type SingleConfig struct {
	Addr string `json:"addr" yaml:"addr"` // 例如: "127.0.0.1:6379"
}

// SentinelConfig 哨兵配置
type SentinelConfig struct {
	MasterName       string   `json:"master_name" yaml:"master_name"`
	SentinelAddrs    []string `json:"sentinel_addrs" yaml:"sentinel_addrs"`
	SentinelPassword string   `json:"sentinel_password,omitempty" yaml:"sentinel_password,omitempty"` // 哨兵密码
}

// ClusterConfig 集群配置
type ClusterConfig struct {
	Addrs []string `json:"addrs" yaml:"addrs"` // 集群节点地址列表
}

// MultiMasterConfig 多主多从配置
type MultiMasterConfig struct {
	Masters []MasterConfig `json:"masters" yaml:"masters"`
}

// MasterConfig 主节点配置
type MasterConfig struct {
	Addr   string   `json:"addr" yaml:"addr"`
	Slaves []string `json:"slaves,omitempty" yaml:"slaves,omitempty"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Mode:         "single",
		Debug:        false,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
		Single:       &SingleConfig{Addr: "127.0.0.1:6379"},
	}
}

// Validate 验证配置是否有效
func (c *Config) Validate() error {
	if c == nil {
		return redisxerrors.ErrConfigNil
	}

	switch c.Mode {
	case "single":
		if c.Single == nil || c.Single.Addr == "" {
			return redisxerrors.ErrConfigSingleAddr
		}
	case "sentinel":
		if c.Sentinel == nil {
			return redisxerrors.ErrConfigSentinelNil
		}
		if c.Sentinel.MasterName == "" {
			return redisxerrors.ErrConfigSentinelMasterName
		}
		if len(c.Sentinel.SentinelAddrs) == 0 {
			return redisxerrors.ErrConfigSentinelAddrs
		}
	case "cluster":
		if c.Cluster == nil || len(c.Cluster.Addrs) == 0 {
			return redisxerrors.ErrConfigClusterAddrs
		}
	case "multi-master":
		if c.MultiMaster == nil || len(c.MultiMaster.Masters) == 0 {
			return redisxerrors.ErrConfigMultiMasterMasters
		}
		for i, master := range c.MultiMaster.Masters {
			if master.Addr == "" {
				return ErrConfigMultiMasterAddr(i)
			}
		}
	default:
		return redisxerrors.ErrConfigMode
	}

	return nil
}
