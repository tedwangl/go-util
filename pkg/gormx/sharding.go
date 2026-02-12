package gormx

import (
	"fmt"
	"hash/crc32"
)

// ShardingConfig 分片配置
type ShardingConfig struct {
	// 分片算法：hash, range, mod
	Algorithm string `json:"algorithm" yaml:"algorithm"`

	// 分片数量（用于 mod 算法）
	ShardCount int `json:"shard_count" yaml:"shard_count"`

	// 物理分片列表
	Shards []ShardNode `json:"shards" yaml:"shards"`
}

// ShardNode 单个分片节点
type ShardNode struct {
	// 分片 ID（0, 1, 2, 3...）
	ID int `json:"id" yaml:"id"`

	// 分片名称
	Name string `json:"name" yaml:"name"`

	// 主库地址
	DSN string `json:"dsn" yaml:"dsn"`

	// 从库地址（可选）
	ReplicaDSN string `json:"replica_dsn,omitempty" yaml:"replica_dsn,omitempty"`

	// 虚拟节点范围（用于一致性哈希，可选）
	VirtualRange [2]int `json:"virtual_range,omitempty" yaml:"virtual_range,omitempty"`
}

// WithSharding 配置分片
func (c *Config) WithSharding(sharding ShardingConfig) *Config {
	c.sharding = &sharding
	return c
}

// HasSharding 是否配置了分片
func (c *Config) HasSharding() bool {
	return c.sharding != nil && len(c.sharding.Shards) > 0
}

// ShardID 计算分片 ID
func (c *Config) ShardID(shardKey interface{}) int {
	if c.sharding == nil {
		return 0
	}

	switch c.sharding.Algorithm {
	case "mod":
		return c.shardIDByMod(shardKey)
	case "hash":
		return c.shardIDByHash(shardKey)
	default:
		return c.shardIDByMod(shardKey)
	}
}

// shardIDByMod 取模算法
func (c *Config) shardIDByMod(shardKey interface{}) int {
	var key int64
	switch v := shardKey.(type) {
	case int:
		key = int64(v)
	case int32:
		key = int64(v)
	case int64:
		key = v
	case uint:
		key = int64(v)
	case uint32:
		key = int64(v)
	case uint64:
		key = int64(v)
	default:
		// 字符串或其他类型，使用哈希
		return c.shardIDByHash(shardKey)
	}

	if c.sharding.ShardCount <= 0 {
		return 0
	}

	return int(key % int64(c.sharding.ShardCount))
}

// shardIDByHash 哈希算法（CRC32）
func (c *Config) shardIDByHash(shardKey interface{}) int {
	str := fmt.Sprint(shardKey)
	hash := crc32.ChecksumIEEE([]byte(str))

	if c.sharding.ShardCount <= 0 {
		return 0
	}

	return int(hash % uint32(c.sharding.ShardCount))
}

// GetShardNode 获取分片节点信息
func (c *Config) GetShardNode(shardID int) *ShardNode {
	if c.sharding == nil || shardID < 0 || shardID >= len(c.sharding.Shards) {
		return nil
	}
	return &c.sharding.Shards[shardID]
}
