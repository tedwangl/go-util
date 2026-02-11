package client

import (
	"context"
	"github.com/tedwangl/go-util/pkg/redisx/config"
	"time"

	"github.com/go-redis/redis/v8"
)

// ClusterClient 集群模式Redis客户端
type ClusterClient struct {
	client *redis.ClusterClient
	config *config.ClusterConfig
	opts   *config.Config
}

// NewClusterClient 创建集群模式Redis客户端
func NewClusterClient(cfg *config.ClusterConfig, opts *config.Config) (*ClusterClient, error) {
	if cfg == nil {
		return nil, ErrConfigNil
	}

	if opts == nil {
		opts = config.DefaultConfig()
	}

	redisOpts := &redis.ClusterOptions{
		Addrs:        cfg.Addrs,
		Password:     opts.Password,
		PoolSize:     opts.PoolSize,
		MinIdleConns: opts.MinIdleConns,
		MaxRetries:   opts.MaxRetries,
		DialTimeout:  opts.DialTimeout,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		PoolTimeout:  opts.PoolTimeout,
	}

	client := redis.NewClusterClient(redisOpts)

	return &ClusterClient{
		client: client,
		config: cfg,
		opts:   opts,
	}, nil
}

// Get 获取键值
func (c *ClusterClient) Get(ctx context.Context, key string) (*redis.StringCmd, error) {
	return c.client.Get(ctx, key), nil
}

// Set 设置键值
func (c *ClusterClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return c.client.Set(ctx, key, value, expiration)
}

// SetNX 设置键值（仅当键不存在时）
func (c *ClusterClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return c.client.SetNX(ctx, key, value, expiration)
}

// Del 删除键
func (c *ClusterClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.client.Del(ctx, keys...)
}

// Exists 检查键是否存在
func (c *ClusterClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.client.Exists(ctx, keys...)
}

// Expire 设置键过期时间
func (c *ClusterClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return c.client.Expire(ctx, key, expiration)
}

// TTL 获取键剩余过期时间
func (c *ClusterClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	cmd := c.client.TTL(ctx, key)
	return cmd.Result()
}

// MGet 批量获取键值
func (c *ClusterClient) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	return c.client.MGet(ctx, keys...)
}

// MSet 批量设置键值
func (c *ClusterClient) MSet(ctx context.Context, values ...interface{}) *redis.StatusCmd {
	return c.client.MSet(ctx, values...)
}

// LPush 左侧推入列表
func (c *ClusterClient) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.LPush(ctx, key, values...)
}

// RPush 右侧推入列表
func (c *ClusterClient) RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.RPush(ctx, key, values...)
}

// LPop 左侧弹出列表
func (c *ClusterClient) LPop(ctx context.Context, key string) *redis.StringCmd {
	return c.client.LPop(ctx, key)
}

// RPop 右侧弹出列表
func (c *ClusterClient) RPop(ctx context.Context, key string) *redis.StringCmd {
	return c.client.RPop(ctx, key)
}

// LLen 获取列表长度
func (c *ClusterClient) LLen(ctx context.Context, key string) *redis.IntCmd {
	return c.client.LLen(ctx, key)
}

// HGet 获取哈希字段
func (c *ClusterClient) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return c.client.HGet(ctx, key, field)
}

// HSet 设置哈希字段
func (c *ClusterClient) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.HSet(ctx, key, values...)
}

// HDel 删除哈希字段
func (c *ClusterClient) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	return c.client.HDel(ctx, key, fields...)
}

// HGetAll 获取哈希所有字段
func (c *ClusterClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	cmd := c.client.HGetAll(ctx, key)
	return cmd.Result()
}

// SAdd 添加集合成员
func (c *ClusterClient) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return c.client.SAdd(ctx, key, members...)
}

// SRem 删除集合成员
func (c *ClusterClient) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return c.client.SRem(ctx, key, members...)
}

// SMembers 获取集合所有成员
func (c *ClusterClient) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	return c.client.SMembers(ctx, key)
}

// SIsMember 检查集合成员是否存在
func (c *ClusterClient) SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd {
	return c.client.SIsMember(ctx, key, member)
}

// ZAdd 添加有序集合成员
func (c *ClusterClient) ZAdd(ctx context.Context, key string, members ...*redis.Z) *redis.IntCmd {
	return c.client.ZAdd(ctx, key, members...)
}

// ZRem 删除有序集合成员
func (c *ClusterClient) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return c.client.ZRem(ctx, key, members...)
}

// ZRange 获取有序集合范围
func (c *ClusterClient) ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return c.client.ZRange(ctx, key, start, stop)
}

// ZScore 获取有序集合成员分数
func (c *ClusterClient) ZScore(ctx context.Context, key string, member string) *redis.FloatCmd {
	return c.client.ZScore(ctx, key, member)
}

// Incr 递增计数器
func (c *ClusterClient) Incr(ctx context.Context, key string) *redis.IntCmd {
	return c.client.Incr(ctx, key)
}

// IncrBy 递增指定值
func (c *ClusterClient) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return c.client.IncrBy(ctx, key, value)
}

// Decr 递减计数器
func (c *ClusterClient) Decr(ctx context.Context, key string) *redis.IntCmd {
	return c.client.Decr(ctx, key)
}

// DecrBy 递减指定值
func (c *ClusterClient) DecrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return c.client.DecrBy(ctx, key, value)
}

// Ping 测试连接
func (c *ClusterClient) Ping(ctx context.Context) *redis.StatusCmd {
	return c.client.Ping(ctx)
}

// Close 关闭连接
func (c *ClusterClient) Close() error {
	return c.client.Close()
}

// GetClient 获取底层客户端
func (c *ClusterClient) GetClient() interface{} {
	return c.client
}

// Pipeline 创建管道
func (c *ClusterClient) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}

// TxPipeline 创建事务管道
func (c *ClusterClient) TxPipeline() redis.Pipeliner {
	return c.client.TxPipeline()
}

// Eval 执行Lua脚本
func (c *ClusterClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return c.client.Eval(ctx, script, keys, args...)
}

// EvalSha 执行Lua脚本（通过SHA1）
func (c *ClusterClient) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd {
	return c.client.EvalSha(ctx, sha1, keys, args...)
}
