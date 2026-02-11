package advanced

import (
	"context"
	"fmt"
	"github.com/tedwangl/go-util/pkg/redisx/client"

	"github.com/go-redis/redis/v8"
)

type WatchHandler struct {
	client client.Client
}

func NewWatchHandler(cli client.Client) *WatchHandler {
	return &WatchHandler{
		client: cli,
	}
}

func (wh *WatchHandler) Watch(ctx context.Context, fn func(tx *redis.Tx) error, keys ...string) error {
	redisClient := wh.client.GetClient().(*redis.Client)

	err := redisClient.Watch(ctx, fn, keys...)

	return err
}

func (wh *WatchHandler) WatchWithRetry(ctx context.Context, maxRetries int, fn func(tx *redis.Tx) error, keys ...string) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		err := wh.Watch(ctx, fn, keys...)
		if err == nil {
			return nil
		}
		lastErr = err

		if err == redis.TxFailedErr {
			continue
		}

		return err
	}

	return fmt.Errorf("watch failed after %d retries: %w", maxRetries, lastErr)
}

func OptimisticIncrement(ctx context.Context, cli client.Client, key string) (int64, error) {
	handler := NewWatchHandler(cli)

	var result int64
	err := handler.Watch(ctx, func(tx *redis.Tx) error {
		cmd := tx.Get(ctx, key)
		value, err := cmd.Int64()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("get value failed: %w", err)
		}

		newValue := value + 1
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, newValue, 0)
			return nil
		})
		if err != nil {
			return err
		}
		result = newValue

		return nil
	}, key)

	return result, err
}

func OptimisticDecrement(ctx context.Context, cli client.Client, key string) (int64, error) {
	handler := NewWatchHandler(cli)

	var result int64
	err := handler.Watch(ctx, func(tx *redis.Tx) error {
		cmd := tx.Get(ctx, key)
		value, err := cmd.Int64()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("get value failed: %w", err)
		}

		newValue := value - 1
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, newValue, 0)
			return nil
		})
		if err != nil {
			return err
		}
		result = newValue

		return nil
	}, key)

	return result, err
}

func CheckAndIncrement(ctx context.Context, cli client.Client, key string, max int64) (int64, error) {
	handler := NewWatchHandler(cli)

	var result int64
	err := handler.Watch(ctx, func(tx *redis.Tx) error {
		cmd := tx.Get(ctx, key)
		value, err := cmd.Int64()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("get value failed: %w", err)
		}

		if value >= max {
			return fmt.Errorf("value exceeds maximum")
		}

		newValue := value + 1
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, newValue, 0)
			return nil
		})
		if err != nil {
			return err
		}
		result = newValue

		return nil
	}, key)

	return result, err
}

func WatchCheckAndSet(ctx context.Context, cli client.Client, key string, expectedValue interface{}, newValue interface{}) (bool, error) {
	handler := NewWatchHandler(cli)

	var success bool
	err := handler.Watch(ctx, func(tx *redis.Tx) error {
		cmd := tx.Get(ctx, key)
		current, err := cmd.Result()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("get value failed: %w", err)
		}

		if current == expectedValue {
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, key, newValue, 0)
				return nil
			})
			if err != nil {
				return err
			}
			success = true
		} else {
			success = false
		}

		return nil
	}, key)

	return success, err
}

func WatchTransfer(ctx context.Context, cli client.Client, fromKey, toKey string, amount int64) error {
	handler := NewWatchHandler(cli)

	err := handler.Watch(ctx, func(tx *redis.Tx) error {
		fromCmd := tx.Get(ctx, fromKey)
		fromBalance, err := fromCmd.Int64()
		if err != nil {
			return fmt.Errorf("get from balance failed: %w", err)
		}

		if fromBalance < amount {
			return fmt.Errorf("insufficient balance")
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.IncrBy(ctx, fromKey, -amount)
			pipe.IncrBy(ctx, toKey, amount)
			return nil
		})
		return err
	}, fromKey, toKey)

	return err
}

func WatchBatchUpdate(ctx context.Context, cli client.Client, updates map[string]int64) error {
	handler := NewWatchHandler(cli)

	keys := make([]string, 0, len(updates))
	for key := range updates {
		keys = append(keys, key)
	}

	err := handler.Watch(ctx, func(tx *redis.Tx) error {
		_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			for key, delta := range updates {
				pipe.IncrBy(ctx, key, delta)
			}
			return nil
		})
		return err
	}, keys...)

	return err
}

func WatchDecrementIfEnough(ctx context.Context, cli client.Client, key string, amount int64) (int64, error) {
	handler := NewWatchHandler(cli)

	var result int64
	err := handler.Watch(ctx, func(tx *redis.Tx) error {
		cmd := tx.Get(ctx, key)
		value, err := cmd.Int64()
		if err != nil {
			return fmt.Errorf("get value failed: %w", err)
		}

		if value < amount {
			return fmt.Errorf("insufficient value")
		}

		newValue := value - amount
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, newValue, 0)
			return nil
		})
		if err != nil {
			return err
		}
		result = newValue

		return nil
	}, key)

	return result, err
}

type CASOperation struct {
	handler *WatchHandler
}

func NewCASOperation(cli client.Client) *CASOperation {
	return &CASOperation{
		handler: NewWatchHandler(cli),
	}
}

func (cas *CASOperation) CompareAndSwap(ctx context.Context, key string, oldValue, newValue interface{}) (bool, error) {
	return WatchCheckAndSet(ctx, cas.handler.client, key, oldValue, newValue)
}

func (cas *CASOperation) CompareAndSwapWithRetry(ctx context.Context, key string, oldValue, newValue interface{}, maxRetries int) (bool, error) {
	var success bool
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		var err error
		success, err = cas.CompareAndSwap(ctx, key, oldValue, newValue)
		if err == nil {
			return success, nil
		}
		lastErr = err

		if err == redis.TxFailedErr {
			continue
		}

		return false, err
	}

	return false, fmt.Errorf("CAS failed after %d retries: %w", maxRetries, lastErr)
}

func (cas *CASOperation) GetAndSet(ctx context.Context, key string, newValue interface{}) (interface{}, error) {
	handler := NewWatchHandler(cas.handler.client)

	var oldValue interface{}
	err := handler.Watch(ctx, func(tx *redis.Tx) error {
		cmd := tx.Get(ctx, key)
		current, err := cmd.Result()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("get value failed: %w", err)
		}

		oldValue = current
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, newValue, 0)
			return nil
		})
		return err
	}, key)

	return oldValue, err
}
