package client

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/tedwangl/go-util/pkg/redisx/config"
)

// SentinelClient 哨兵模式Redis客户端
type SentinelClient struct {
	client *redis.Client
	config *config.SentinelConfig
	opts   *config.Config
}

// NewSentinelClient 创建哨兵模式Redis客户端
func NewSentinelClient(cfg *config.SentinelConfig, opts *config.Config) (*SentinelClient, error) {
	if cfg == nil {
		return nil, ErrConfigNil
	}

	if opts == nil {
		opts = config.DefaultConfig()
	}

	redisOpts := &redis.FailoverOptions{
		MasterName:       cfg.MasterName,
		SentinelAddrs:    cfg.SentinelAddrs,
		SentinelPassword: cfg.SentinelPassword,
		Username:         opts.Username,
		Password:         opts.Password,
		DB:               opts.DB,
		PoolSize:         opts.PoolSize,
		MinIdleConns:     opts.MinIdleConns,
		MaxRetries:       opts.MaxRetries,
		DialTimeout:      opts.DialTimeout,
		ReadTimeout:      opts.ReadTimeout,
		WriteTimeout:     opts.WriteTimeout,
		PoolTimeout:      opts.PoolTimeout,
	}

	client := redis.NewFailoverClient(redisOpts)

	return &SentinelClient{
		client: client,
		config: cfg,
		opts:   opts,
	}, nil
}

// Get 获取键值
func (c *SentinelClient) Get(ctx context.Context, key string) (*redis.StringCmd, error) {
	return c.client.Get(ctx, key), nil
}

// Set 设置键值
func (c *SentinelClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return c.client.Set(ctx, key, value, expiration)
}

// SetNX 设置键值（仅当键不存在时）
func (c *SentinelClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return c.client.SetNX(ctx, key, value, expiration)
}

// Del 删除键
func (c *SentinelClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.client.Del(ctx, keys...)
}

// Exists 检查键是否存在
func (c *SentinelClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.client.Exists(ctx, keys...)
}

// Expire 设置键过期时间
func (c *SentinelClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return c.client.Expire(ctx, key, expiration)
}

// TTL 获取键剩余过期时间
func (c *SentinelClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	cmd := c.client.TTL(ctx, key)
	return cmd.Result()
}

// MGet 批量获取键值
func (c *SentinelClient) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	return c.client.MGet(ctx, keys...)
}

// MSet 批量设置键值
func (c *SentinelClient) MSet(ctx context.Context, values ...interface{}) *redis.StatusCmd {
	return c.client.MSet(ctx, values...)
}

// LPush 左侧推入列表
func (c *SentinelClient) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.LPush(ctx, key, values...)
}

// RPush 右侧推入列表
func (c *SentinelClient) RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.RPush(ctx, key, values...)
}

// LPop 左侧弹出列表
func (c *SentinelClient) LPop(ctx context.Context, key string) *redis.StringCmd {
	return c.client.LPop(ctx, key)
}

// RPop 右侧弹出列表
func (c *SentinelClient) RPop(ctx context.Context, key string) *redis.StringCmd {
	return c.client.RPop(ctx, key)
}

// LLen 获取列表长度
func (c *SentinelClient) LLen(ctx context.Context, key string) *redis.IntCmd {
	return c.client.LLen(ctx, key)
}

// HGet 获取哈希字段
func (c *SentinelClient) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return c.client.HGet(ctx, key, field)
}

// HSet 设置哈希字段
func (c *SentinelClient) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.HSet(ctx, key, values...)
}

// HDel 删除哈希字段
func (c *SentinelClient) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	return c.client.HDel(ctx, key, fields...)
}

// HGetAll 获取哈希表所有字段和值
func (c *SentinelClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	cmd := c.client.HGetAll(ctx, key)
	return cmd.Result()
}

// SAdd 添加集合成员
func (c *SentinelClient) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return c.client.SAdd(ctx, key, members...)
}

// SRem 删除集合成员
func (c *SentinelClient) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return c.client.SRem(ctx, key, members...)
}

// SMembers 获取集合所有成员
func (c *SentinelClient) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	return c.client.SMembers(ctx, key)
}

// SIsMember 检查集合成员是否存在
func (c *SentinelClient) SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd {
	return c.client.SIsMember(ctx, key, member)
}

// ZAdd 添加有序集合成员
func (c *SentinelClient) ZAdd(ctx context.Context, key string, members ...*redis.Z) *redis.IntCmd {
	// v9 API 变化：ZAdd 参数从 ...*redis.Z 改为 ...redis.Z
	zMembers := make([]redis.Z, len(members))
	for i, m := range members {
		zMembers[i] = *m
	}
	return c.client.ZAdd(ctx, key, zMembers...)
}

// ZRem 删除有序集合成员
func (c *SentinelClient) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return c.client.ZRem(ctx, key, members...)
}

// ZRange 获取有序集合范围
func (c *SentinelClient) ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return c.client.ZRange(ctx, key, start, stop)
}

// ZScore 获取有序集合成员分数
func (c *SentinelClient) ZScore(ctx context.Context, key string, member string) *redis.FloatCmd {
	return c.client.ZScore(ctx, key, member)
}

// Incr 递增计数器
func (c *SentinelClient) Incr(ctx context.Context, key string) *redis.IntCmd {
	return c.client.Incr(ctx, key)
}

// IncrBy 递增指定值
func (c *SentinelClient) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return c.client.IncrBy(ctx, key, value)
}

// Decr 递减计数器
func (c *SentinelClient) Decr(ctx context.Context, key string) *redis.IntCmd {
	return c.client.Decr(ctx, key)
}

// DecrBy 递减指定值
func (c *SentinelClient) DecrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return c.client.DecrBy(ctx, key, value)
}

// Ping 测试连接
func (c *SentinelClient) Ping(ctx context.Context) *redis.StatusCmd {
	return c.client.Ping(ctx)
}

// Close 关闭连接
func (c *SentinelClient) Close() error {
	return c.client.Close()
}

// GetClient 获取底层客户端
func (c *SentinelClient) GetClient() interface{} {
	return c.client
}

// Pipeline 创建管道
func (c *SentinelClient) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}

// TxPipeline 创建事务管道
func (c *SentinelClient) TxPipeline() redis.Pipeliner {
	return c.client.TxPipeline()
}

// Eval 执行Lua脚本
func (c *SentinelClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return c.client.Eval(ctx, script, keys, args...)
}

// EvalSha 执行Lua脚本（通过SHA1）
func (c *SentinelClient) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd {
	return c.client.EvalSha(ctx, sha1, keys, args...)
}
