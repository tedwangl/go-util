package advanced

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tedwangl/go-util/pkg/redisx/client"
)

type Transaction struct {
	client client.Client
	tx     redis.Pipeliner
}

func NewTransaction(cli client.Client) *Transaction {
	return &Transaction{
		client: cli,
		tx:     cli.TxPipeline(),
	}
}

func (t *Transaction) Get(ctx context.Context, key string) *redis.StringCmd {
	return t.tx.Get(ctx, key)
}

func (t *Transaction) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return t.tx.Set(ctx, key, value, expiration)
}

func (t *Transaction) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return t.tx.Del(ctx, keys...)
}

func (t *Transaction) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return t.tx.Exists(ctx, keys...)
}

func (t *Transaction) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return t.tx.Expire(ctx, key, expiration)
}

func (t *Transaction) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	return t.tx.MGet(ctx, keys...)
}

func (t *Transaction) MSet(ctx context.Context, values ...interface{}) *redis.StatusCmd {
	return t.tx.MSet(ctx, values...)
}

func (t *Transaction) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return t.tx.LPush(ctx, key, values...)
}

func (t *Transaction) RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return t.tx.RPush(ctx, key, values...)
}

func (t *Transaction) LPop(ctx context.Context, key string) *redis.StringCmd {
	return t.tx.LPop(ctx, key)
}

func (t *Transaction) RPop(ctx context.Context, key string) *redis.StringCmd {
	return t.tx.RPop(ctx, key)
}

func (t *Transaction) LLen(ctx context.Context, key string) *redis.IntCmd {
	return t.tx.LLen(ctx, key)
}

func (t *Transaction) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return t.tx.HGet(ctx, key, field)
}

func (t *Transaction) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return t.tx.HSet(ctx, key, values...)
}

func (t *Transaction) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	return t.tx.HDel(ctx, key, fields...)
}

func (t *Transaction) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	return t.tx.HGetAll(ctx, key)
}

func (t *Transaction) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return t.tx.SAdd(ctx, key, members...)
}

func (t *Transaction) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return t.tx.SRem(ctx, key, members...)
}

func (t *Transaction) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	return t.tx.SMembers(ctx, key)
}

func (t *Transaction) SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd {
	return t.tx.SIsMember(ctx, key, member)
}

func (t *Transaction) ZAdd(ctx context.Context, key string, members ...*redis.Z) *redis.IntCmd {
	// v9 API 变化：ZAdd 参数从 ...*redis.Z 改为 ...redis.Z
	zMembers := make([]redis.Z, len(members))
	for i, m := range members {
		zMembers[i] = *m
	}
	return t.tx.ZAdd(ctx, key, zMembers...)
}

func (t *Transaction) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return t.tx.ZRem(ctx, key, members...)
}

func (t *Transaction) ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return t.tx.ZRange(ctx, key, start, stop)
}

func (t *Transaction) ZScore(ctx context.Context, key string, member string) *redis.FloatCmd {
	return t.tx.ZScore(ctx, key, member)
}

func (t *Transaction) Incr(ctx context.Context, key string) *redis.IntCmd {
	return t.tx.Incr(ctx, key)
}

func (t *Transaction) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return t.tx.IncrBy(ctx, key, value)
}

func (t *Transaction) Decr(ctx context.Context, key string) *redis.IntCmd {
	return t.tx.Decr(ctx, key)
}

func (t *Transaction) DecrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return t.tx.DecrBy(ctx, key, value)
}

func (t *Transaction) Exec(ctx context.Context) ([]redis.Cmder, error) {
	return t.tx.Exec(ctx)
}

func (t *Transaction) Discard() error {
	// v9 的 Discard 不返回 error
	t.tx.Discard()
	return nil
}

func (t *Transaction) Close() error {
	// v9 移除了 Close 方法，不需要手动关闭
	return nil
}

func (t *Transaction) Len() int {
	return t.tx.Len()
}

type TransactionHandler struct {
	client client.Client
}

func NewTransactionHandler(cli client.Client) *TransactionHandler {
	return &TransactionHandler{
		client: cli,
	}
}

func (th *TransactionHandler) Exec(ctx context.Context, fn func(tx *Transaction) error) ([]redis.Cmder, error) {
	tx := NewTransaction(th.client)
	defer tx.Close()

	if err := fn(tx); err != nil {
		if discardErr := tx.Discard(); discardErr != nil {
			return nil, fmt.Errorf("transaction error: %w, discard error: %v", err, discardErr)
		}
		return nil, err
	}

	return tx.Exec(ctx)
}

func (th *TransactionHandler) ExecWithRetry(ctx context.Context, maxRetries int, fn func(tx *Transaction) error) ([]redis.Cmder, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		tx := NewTransaction(th.client)

		if err := fn(tx); err != nil {
			lastErr = err
			tx.Discard()
			continue
		}

		result, err := tx.Exec(ctx)
		if err == nil {
			return result, nil
		}
		lastErr = err
		tx.Close()
	}

	return nil, fmt.Errorf("transaction failed after %d retries: %w", maxRetries, lastErr)
}

func TransferBalance(ctx context.Context, cli client.Client, fromKey, toKey string, amount int64) error {
	handler := NewTransactionHandler(cli)

	_, err := handler.Exec(ctx, func(tx *Transaction) error {
		fromCmd := tx.Get(ctx, fromKey)

		fromBalance, err := fromCmd.Int64()
		if err != nil {
			return fmt.Errorf("get from balance failed: %w", err)
		}

		if fromBalance < amount {
			return fmt.Errorf("insufficient balance")
		}

		tx.IncrBy(ctx, fromKey, -amount)
		tx.IncrBy(ctx, toKey, amount)

		return nil
	})

	return err
}

func DecrementWithCheck(ctx context.Context, cli client.Client, key string, amount int64) (int64, error) {
	handler := NewTransactionHandler(cli)

	var result int64
	_, err := handler.Exec(ctx, func(tx *Transaction) error {
		cmd := tx.Get(ctx, key)
		value, err := cmd.Int64()
		if err != nil {
			return fmt.Errorf("get value failed: %w", err)
		}

		if value < amount {
			return fmt.Errorf("insufficient value")
		}

		newValue := value - amount
		tx.Set(ctx, key, newValue, 0)
		result = newValue

		return nil
	})

	return result, err
}

func BatchUpdateInTransaction(ctx context.Context, cli client.Client, updates map[string]int64) error {
	handler := NewTransactionHandler(cli)

	_, err := handler.Exec(ctx, func(tx *Transaction) error {
		for key, delta := range updates {
			tx.IncrBy(ctx, key, delta)
		}
		return nil
	})

	return err
}

func CheckAndSet(ctx context.Context, cli client.Client, key string, expectedValue interface{}, newValue interface{}) (bool, error) {
	handler := NewTransactionHandler(cli)

	var success bool
	_, err := handler.Exec(ctx, func(tx *Transaction) error {
		cmd := tx.Get(ctx, key)
		current, err := cmd.Result()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("get value failed: %w", err)
		}

		if current == expectedValue {
			tx.Set(ctx, key, newValue, 0)
			success = true
		} else {
			success = false
		}

		return nil
	})

	return success, err
}

func AtomicIncrement(ctx context.Context, cli client.Client, key string, max int64) (int64, error) {
	handler := NewTransactionHandler(cli)

	var result int64
	_, err := handler.Exec(ctx, func(tx *Transaction) error {
		cmd := tx.Get(ctx, key)
		value, err := cmd.Int64()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("get value failed: %w", err)
		}

		if value >= max {
			return fmt.Errorf("value exceeds maximum")
		}

		newValue := value + 1
		tx.Set(ctx, key, newValue, 0)
		result = newValue

		return nil
	})

	return result, err
}
