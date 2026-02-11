package client

import (
	"context"
	"fmt"
	"github.com/tedwangl/go-util/pkg/redisx/config"
	"time"

	"github.com/go-redis/redis/v8"
)

// NewClient 根据配置创建Redis客户端
func NewClient(cfg *config.Config) (Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	switch cfg.Mode {
	case "single":
		return NewSingleClient(cfg.Single, cfg)
	case "sentinel":
		return NewSentinelClient(cfg.Sentinel, cfg)
	case "cluster":
		return NewClusterClient(cfg.Cluster, cfg)
	case "multi-master":
		return NewMultiMasterClient(cfg.MultiMaster, cfg)
	default:
		return nil, fmt.Errorf("不支持的部署模式: %s", cfg.Mode)
	}
}

// Client Redis客户端接口
type Client interface {
	// 基础操作
	Get(ctx context.Context, key string) (*redis.StringCmd, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	TTL(ctx context.Context, key string) (time.Duration, error)

	// 批量操作
	MGet(ctx context.Context, keys ...string) *redis.SliceCmd
	MSet(ctx context.Context, values ...interface{}) *redis.StatusCmd

	// 列表操作
	LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	LPop(ctx context.Context, key string) *redis.StringCmd
	RPop(ctx context.Context, key string) *redis.StringCmd
	LLen(ctx context.Context, key string) *redis.IntCmd

	// 哈希操作
	HGet(ctx context.Context, key, field string) *redis.StringCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
	HGetAll(ctx context.Context, key string) (map[string]string, error)

	// 集合操作
	SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
	SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd

	// 有序集合操作
	ZAdd(ctx context.Context, key string, members ...*redis.Z) *redis.IntCmd
	ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd
	ZScore(ctx context.Context, key string, member string) *redis.FloatCmd

	// 计数器操作
	Incr(ctx context.Context, key string) *redis.IntCmd
	IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd
	Decr(ctx context.Context, key string) *redis.IntCmd
	DecrBy(ctx context.Context, key string, value int64) *redis.IntCmd

	// 连接管理
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error

	// 获取底层客户端
	GetClient() interface{}

	// 高级操作
	Pipeline() redis.Pipeliner
	TxPipeline() redis.Pipeliner
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd
}
