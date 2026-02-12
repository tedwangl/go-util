package advanced

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tedwangl/go-util/pkg/redisx/client"
)

type Pipeline struct {
	client client.Client
	pipe   redis.Pipeliner
}

func NewPipeline(cli client.Client) *Pipeline {
	return &Pipeline{
		client: cli,
		pipe:   cli.Pipeline(),
	}
}

func (p *Pipeline) Get(ctx context.Context, key string) *redis.StringCmd {
	return p.pipe.Get(ctx, key)
}

func (p *Pipeline) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return p.pipe.Set(ctx, key, value, expiration)
}

func (p *Pipeline) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return p.pipe.Del(ctx, keys...)
}

func (p *Pipeline) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return p.pipe.Exists(ctx, keys...)
}

func (p *Pipeline) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return p.pipe.Expire(ctx, key, expiration)
}

func (p *Pipeline) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	return p.pipe.MGet(ctx, keys...)
}

func (p *Pipeline) MSet(ctx context.Context, values ...interface{}) *redis.StatusCmd {
	return p.pipe.MSet(ctx, values...)
}

func (p *Pipeline) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return p.pipe.LPush(ctx, key, values...)
}

func (p *Pipeline) RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return p.pipe.RPush(ctx, key, values...)
}

func (p *Pipeline) LPop(ctx context.Context, key string) *redis.StringCmd {
	return p.pipe.LPop(ctx, key)
}

func (p *Pipeline) RPop(ctx context.Context, key string) *redis.StringCmd {
	return p.pipe.RPop(ctx, key)
}

func (p *Pipeline) LLen(ctx context.Context, key string) *redis.IntCmd {
	return p.pipe.LLen(ctx, key)
}

func (p *Pipeline) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return p.pipe.HGet(ctx, key, field)
}

func (p *Pipeline) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return p.pipe.HSet(ctx, key, values...)
}

func (p *Pipeline) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	return p.pipe.HDel(ctx, key, fields...)
}

func (p *Pipeline) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	return p.pipe.HGetAll(ctx, key)
}

func (p *Pipeline) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return p.pipe.SAdd(ctx, key, members...)
}

func (p *Pipeline) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return p.pipe.SRem(ctx, key, members...)
}

func (p *Pipeline) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	return p.pipe.SMembers(ctx, key)
}

func (p *Pipeline) SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd {
	return p.pipe.SIsMember(ctx, key, member)
}

func (p *Pipeline) ZAdd(ctx context.Context, key string, members ...*redis.Z) *redis.IntCmd {
	// v9 API 变化：ZAdd 参数从 ...*redis.Z 改为 ...redis.Z
	zMembers := make([]redis.Z, len(members))
	for i, m := range members {
		zMembers[i] = *m
	}
	return p.pipe.ZAdd(ctx, key, zMembers...)
}

func (p *Pipeline) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return p.pipe.ZRem(ctx, key, members...)
}

func (p *Pipeline) ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return p.pipe.ZRange(ctx, key, start, stop)
}

func (p *Pipeline) ZScore(ctx context.Context, key string, member string) *redis.FloatCmd {
	return p.pipe.ZScore(ctx, key, member)
}

func (p *Pipeline) Incr(ctx context.Context, key string) *redis.IntCmd {
	return p.pipe.Incr(ctx, key)
}

func (p *Pipeline) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return p.pipe.IncrBy(ctx, key, value)
}

func (p *Pipeline) Decr(ctx context.Context, key string) *redis.IntCmd {
	return p.pipe.Decr(ctx, key)
}

func (p *Pipeline) DecrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return p.pipe.DecrBy(ctx, key, value)
}

func (p *Pipeline) Exec(ctx context.Context) ([]redis.Cmder, error) {
	return p.pipe.Exec(ctx)
}

func (p *Pipeline) Close() error {
	// v9 移除了 Close 方法，不需要手动关闭
	return nil
}

func (p *Pipeline) Discard() error {
	// v9 的 Discard 不返回 error
	p.pipe.Discard()
	return nil
}

func (p *Pipeline) Len() int {
	return p.pipe.Len()
}

type BatchOperation struct {
	operations []func(*Pipeline)
}

func NewBatchOperation() *BatchOperation {
	return &BatchOperation{
		operations: make([]func(*Pipeline), 0),
	}
}

func (b *BatchOperation) Add(op func(*Pipeline)) {
	b.operations = append(b.operations, op)
}

func (b *BatchOperation) Execute(ctx context.Context, cli client.Client) ([]redis.Cmder, error) {
	pipe := NewPipeline(cli)
	defer pipe.Close()

	for _, op := range b.operations {
		op(pipe)
	}

	return pipe.Exec(ctx)
}

func BatchGet(ctx context.Context, cli client.Client, keys ...string) ([]interface{}, error) {
	pipe := NewPipeline(cli)
	defer pipe.Close()

	cmds := make([]*redis.StringCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("pipeline exec failed: %w", err)
	}

	results := make([]interface{}, len(keys))
	for i, cmd := range cmds {
		val, err := cmd.Result()
		if err != nil {
			results[i] = nil
		} else {
			results[i] = val
		}
	}

	return results, nil
}

func BatchSet(ctx context.Context, cli client.Client, pairs map[string]interface{}, expiration time.Duration) error {
	pipe := NewPipeline(cli)
	defer pipe.Close()

	for key, value := range pairs {
		pipe.Set(ctx, key, value, expiration)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("pipeline exec failed: %w", err)
	}

	return nil
}

func BatchDel(ctx context.Context, cli client.Client, keys ...string) (int64, error) {
	pipe := NewPipeline(cli)
	defer pipe.Close()

	cmd := pipe.Del(ctx, keys...)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("pipeline exec failed: %w", err)
	}

	return cmd.Val(), nil
}

func BatchExists(ctx context.Context, cli client.Client, keys ...string) (int64, error) {
	pipe := NewPipeline(cli)
	defer pipe.Close()

	cmd := pipe.Exists(ctx, keys...)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("pipeline exec failed: %w", err)
	}

	return cmd.Val(), nil
}

func BatchIncr(ctx context.Context, cli client.Client, keys ...string) ([]int64, error) {
	pipe := NewPipeline(cli)
	defer pipe.Close()

	cmds := make([]*redis.IntCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.Incr(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("pipeline exec failed: %w", err)
	}

	results := make([]int64, len(keys))
	for i, cmd := range cmds {
		results[i] = cmd.Val()
	}

	return results, nil
}
