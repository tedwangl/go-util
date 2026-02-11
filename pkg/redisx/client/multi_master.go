package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/tedwangl/go-util/pkg/redisx/config"
)

var (
	// ErrConfigNil 配置为nil
	ErrConfigNil = errors.New("config is nil")
	// ErrNoMasterAvailable 无可用主节点
	ErrNoMasterAvailable = errors.New("no master available")
	// ErrNoSlaveAvailable 无可用从节点
	ErrNoSlaveAvailable = errors.New("no slave available")
)

// MultiMasterClient 多主多从Redis客户端
type MultiMasterClient struct {
	masters []*redis.Client
	slaves  []*redis.Client
	config  *config.MultiMasterConfig
	opts    *config.Config
	router  *Router
	mu      sync.RWMutex
}

// Router 读写路由器
type Router struct {
	masters []*redis.Client
	slaves  []*redis.Client
	mu      sync.RWMutex
}

// NewMultiMasterClient 创建多主多从Redis客户端
func NewMultiMasterClient(cfg *config.MultiMasterConfig, opts *config.Config) (*MultiMasterClient, error) {
	if cfg == nil {
		return nil, ErrConfigNil
	}

	if opts == nil {
		opts = config.DefaultConfig()
	}

	masters := make([]*redis.Client, 0, len(cfg.Masters))
	slaves := make([]*redis.Client, 0)

	for _, master := range cfg.Masters {
		// 创建主节点客户端
		masterOpts := &redis.Options{
			Addr:         master.Addr,
			Password:     opts.Password,
			DB:           opts.DB,
			PoolSize:     opts.PoolSize / len(cfg.Masters),
			MinIdleConns: opts.MinIdleConns / len(cfg.Masters),
			MaxRetries:   opts.MaxRetries,
			DialTimeout:  opts.DialTimeout,
			ReadTimeout:  opts.ReadTimeout,
			WriteTimeout: opts.WriteTimeout,
			PoolTimeout:  opts.PoolTimeout,
		}

		masterClient := redis.NewClient(masterOpts)
		masters = append(masters, masterClient)

		// 创建从节点客户端
		for _, slaveAddr := range master.Slaves {
			slaveOpts := &redis.Options{
				Addr:         slaveAddr,
				Password:     opts.Password,
				DB:           opts.DB,
				PoolSize:     opts.PoolSize / len(master.Slaves) / len(cfg.Masters),
				MinIdleConns: opts.MinIdleConns / len(master.Slaves) / len(cfg.Masters),
				MaxRetries:   opts.MaxRetries,
				DialTimeout:  opts.DialTimeout,
				ReadTimeout:  opts.ReadTimeout,
				WriteTimeout: opts.WriteTimeout,
				PoolTimeout:  opts.PoolTimeout,
			}

			slaveClient := redis.NewClient(slaveOpts)
			slaves = append(slaves, slaveClient)
		}
	}

	router := &Router{
		masters: masters,
		slaves:  slaves,
	}

	return &MultiMasterClient{
		masters: masters,
		slaves:  slaves,
		config:  cfg,
		opts:    opts,
		router:  router,
	}, nil
}

// getMaster 获取主节点客户端
func (r *Router) getMaster() (*redis.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.masters) == 0 {
		return nil, ErrNoMasterAvailable
	}

	// 简单的轮询策略
	for _, master := range r.masters {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		if err := master.Ping(ctx).Err(); err == nil {
			return master, nil
		}
	}

	return nil, ErrNoMasterAvailable
}

// getSlave 获取从节点客户端
func (r *Router) getSlave() (*redis.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.slaves) == 0 {
		// 无从节点时使用主节点
		return r.getMaster()
	}

	// 简单的轮询策略
	for _, slave := range r.slaves {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		if err := slave.Ping(ctx).Err(); err == nil {
			return slave, nil
		}
	}

	// 从节点不可用时使用主节点
	return r.getMaster()
}

// Get 获取键值（读操作，使用从节点）
func (c *MultiMasterClient) Get(ctx context.Context, key string) (*redis.StringCmd, error) {
	slave, err := c.router.getSlave()
	if err != nil {
		return nil, err
	}
	return slave.Get(ctx, key), nil
}

// Set 设置键值（写操作，使用主节点）
func (c *MultiMasterClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	master, err := c.router.getMaster()
	if err != nil {
		return redis.NewStatusCmd(ctx, err)
	}
	return master.Set(ctx, key, value, expiration)
}

// SetNX 设置键值（仅当键不存在时，写操作，使用主节点）
func (c *MultiMasterClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	master, err := c.router.getMaster()
	if err != nil {
		return redis.NewBoolCmd(ctx, err)
	}
	return master.SetNX(ctx, key, value, expiration)
}

// Del 删除键（写操作，使用主节点）
func (c *MultiMasterClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewIntCmd(ctx, err)
	}
	return master.Del(ctx, keys...)
}

// Exists 检查键是否存在（读操作，使用从节点）
func (c *MultiMasterClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	slave, err := c.router.getSlave()
	if err != nil {
		// 无从节点时使用主节点
		master, err := c.router.getMaster()
		if err != nil {
			return redis.NewIntCmd(ctx, err)
		}
		return master.Exists(ctx, keys...)
	}
	return slave.Exists(ctx, keys...)
}

// Expire 设置键过期时间（写操作，使用主节点）
func (c *MultiMasterClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewBoolCmd(ctx, err)
	}
	return master.Expire(ctx, key, expiration)
}

// TTL 获取键剩余过期时间（读操作，使用从节点）
func (c *MultiMasterClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	// 尝试从从节点获取
	slave, err := c.router.getSlave()
	if err == nil {
		cmd := slave.TTL(ctx, key)
		return cmd.Result()
	}

	// 尝试从主节点获取
	master, err := c.router.getMaster()
	if err == nil {
		cmd := master.TTL(ctx, key)
		return cmd.Result()
	}

	// 无可用节点，返回错误
	return 0, err
}

// MGet 批量获取键值（读操作，使用从节点）
func (c *MultiMasterClient) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	slave, err := c.router.getSlave()
	if err != nil {
		// 无从节点时使用主节点
		master, err := c.router.getMaster()
		if err != nil {
			return redis.NewSliceCmd(ctx, err)
		}
		return master.MGet(ctx, keys...)
	}
	return slave.MGet(ctx, keys...)
}

// MSet 批量设置键值（写操作，使用主节点）
func (c *MultiMasterClient) MSet(ctx context.Context, values ...interface{}) *redis.StatusCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewStatusCmd(ctx, err)
	}
	return master.MSet(ctx, values...)
}

// LPush 左侧推入列表（写操作，使用主节点）
func (c *MultiMasterClient) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewIntCmd(ctx, err)
	}
	return master.LPush(ctx, key, values...)
}

// RPush 右侧推入列表（写操作，使用主节点）
func (c *MultiMasterClient) RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewIntCmd(ctx, err)
	}
	return master.RPush(ctx, key, values...)
}

// LPop 左侧弹出列表（写操作，使用主节点）
func (c *MultiMasterClient) LPop(ctx context.Context, key string) *redis.StringCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewStringCmd(ctx, err)
	}
	return master.LPop(ctx, key)
}

// RPop 右侧弹出列表（写操作，使用主节点）
func (c *MultiMasterClient) RPop(ctx context.Context, key string) *redis.StringCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewStringCmd(ctx, err)
	}
	return master.RPop(ctx, key)
}

// LLen 获取列表长度（读操作，使用从节点）
func (c *MultiMasterClient) LLen(ctx context.Context, key string) *redis.IntCmd {
	slave, err := c.router.getSlave()
	if err != nil {
		// 无从节点时使用主节点
		master, err := c.router.getMaster()
		if err != nil {
			return redis.NewIntCmd(ctx, err)
		}
		return master.LLen(ctx, key)
	}
	return slave.LLen(ctx, key)
}

// HGet 获取哈希字段（读操作，使用从节点）
func (c *MultiMasterClient) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	slave, err := c.router.getSlave()
	if err != nil {
		// 无从节点时使用主节点
		master, err := c.router.getMaster()
		if err != nil {
			return redis.NewStringCmd(ctx, err)
		}
		return master.HGet(ctx, key, field)
	}
	return slave.HGet(ctx, key, field)
}

// HSet 设置哈希字段（写操作，使用主节点）
func (c *MultiMasterClient) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewIntCmd(ctx, err)
	}
	return master.HSet(ctx, key, values...)
}

// HDel 删除哈希字段（写操作，使用主节点）
func (c *MultiMasterClient) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewIntCmd(ctx, err)
	}
	return master.HDel(ctx, key, fields...)
}

// HGetAll 获取哈希所有字段（读操作，使用从节点）
func (c *MultiMasterClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	slave, err := c.router.getSlave()
	if err != nil {
		return nil, err
	}

	cmd := slave.HGetAll(ctx, key)
	return cmd.Result()
}

// SAdd 添加集合成员（写操作，使用主节点）
func (c *MultiMasterClient) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewIntCmd(ctx, err)
	}
	return master.SAdd(ctx, key, members...)
}

// SRem 删除集合成员（写操作，使用主节点）
func (c *MultiMasterClient) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewIntCmd(ctx, err)
	}
	return master.SRem(ctx, key, members...)
}

// SMembers 获取集合所有成员（读操作，使用从节点）
func (c *MultiMasterClient) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	slave, err := c.router.getSlave()
	if err != nil {
		// 无从节点时使用主节点
		master, err := c.router.getMaster()
		if err != nil {
			return redis.NewStringSliceCmd(ctx, err)
		}
		return master.SMembers(ctx, key)
	}
	return slave.SMembers(ctx, key)
}

// SIsMember 检查集合成员是否存在（读操作，使用从节点）
func (c *MultiMasterClient) SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd {
	slave, err := c.router.getSlave()
	if err != nil {
		// 无从节点时使用主节点
		master, err := c.router.getMaster()
		if err != nil {
			return redis.NewBoolCmd(ctx, err)
		}
		return master.SIsMember(ctx, key, member)
	}
	return slave.SIsMember(ctx, key, member)
}

// ZAdd 添加有序集合成员（写操作，使用主节点）
func (c *MultiMasterClient) ZAdd(ctx context.Context, key string, members ...*redis.Z) *redis.IntCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewIntCmd(ctx, err)
	}
	return master.ZAdd(ctx, key, members...)
}

// ZRem 删除有序集合成员（写操作，使用主节点）
func (c *MultiMasterClient) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewIntCmd(ctx, err)
	}
	return master.ZRem(ctx, key, members...)
}

// ZRange 获取有序集合范围（读操作，使用从节点）
func (c *MultiMasterClient) ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	slave, err := c.router.getSlave()
	if err != nil {
		// 无从节点时使用主节点
		master, err := c.router.getMaster()
		if err != nil {
			return redis.NewStringSliceCmd(ctx, err)
		}
		return master.ZRange(ctx, key, start, stop)
	}
	return slave.ZRange(ctx, key, start, stop)
}

// ZScore 获取有序集合成员分数（读操作，使用从节点）
func (c *MultiMasterClient) ZScore(ctx context.Context, key string, member string) *redis.FloatCmd {
	slave, err := c.router.getSlave()
	if err != nil {
		// 无从节点时使用主节点
		master, err := c.router.getMaster()
		if err != nil {
			return redis.NewFloatCmd(ctx, err)
		}
		return master.ZScore(ctx, key, member)
	}
	return slave.ZScore(ctx, key, member)
}

// Incr 递增计数器（写操作，使用主节点）
func (c *MultiMasterClient) Incr(ctx context.Context, key string) *redis.IntCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewIntCmd(ctx, err)
	}
	return master.Incr(ctx, key)
}

// IncrBy 递增指定值（写操作，使用主节点）
func (c *MultiMasterClient) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewIntCmd(ctx, err)
	}
	return master.IncrBy(ctx, key, value)
}

// Decr 递减计数器（写操作，使用主节点）
func (c *MultiMasterClient) Decr(ctx context.Context, key string) *redis.IntCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewIntCmd(ctx, err)
	}
	return master.Decr(ctx, key)
}

// DecrBy 递减指定值（写操作，使用主节点）
func (c *MultiMasterClient) DecrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewIntCmd(ctx, err)
	}
	return master.DecrBy(ctx, key, value)
}

// Ping 测试连接
func (c *MultiMasterClient) Ping(ctx context.Context) *redis.StatusCmd {
	master, err := c.router.getMaster()
	if err != nil {
		return redis.NewStatusCmd(ctx, err)
	}
	return master.Ping(ctx)
}

// Close 关闭连接
func (c *MultiMasterClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var lastErr error

	for _, master := range c.masters {
		if err := master.Close(); err != nil {
			lastErr = fmt.Errorf("close master error: %w", err)
		}
	}

	for _, slave := range c.slaves {
		if err := slave.Close(); err != nil {
			lastErr = fmt.Errorf("close slave error: %w", err)
		}
	}

	return lastErr
}

// GetClient 获取底层客户端
func (c *MultiMasterClient) GetClient() interface{} {
	return map[string]interface{}{
		"masters": c.masters,
		"slaves":  c.slaves,
	}
}

// Pipeline 创建管道
func (c *MultiMasterClient) Pipeline() redis.Pipeliner {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回nil
		return nil
	}
	return master.Pipeline()
}

// TxPipeline 创建事务管道
func (c *MultiMasterClient) TxPipeline() redis.Pipeliner {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回nil
		return nil
	}
	return master.TxPipeline()
}

// Eval 执行Lua脚本（写操作，使用主节点）
func (c *MultiMasterClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewCmd(ctx, err)
	}
	return master.Eval(ctx, script, keys, args...)
}

// EvalSha 执行Lua脚本（通过SHA1）
func (c *MultiMasterClient) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd {
	master, err := c.router.getMaster()
	if err != nil {
		// 无主节点时返回错误
		return redis.NewCmd(ctx, err)
	}
	return master.EvalSha(ctx, sha1, keys, args...)
}
