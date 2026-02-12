package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/tedwangl/go-util/pkg/redisx/client"
)

// UserCache 用户数据缓存
type UserCache struct {
	client client.Client
	prefix string
}

// NewUserCache 创建用户数据缓存
func NewUserCache(client client.Client, prefix string) *UserCache {
	if prefix == "" {
		prefix = "user"
	}

	return &UserCache{
		client: client,
		prefix: prefix,
	}
}

// key 生成缓存键
func (c *UserCache) key(key string) string {
	return fmt.Sprintf("%s:%s", c.prefix, key)
}

// Get 获取缓存值
func (c *UserCache) Get(ctx context.Context, key string) (interface{}, error) {
	cmd, err := c.client.Get(ctx, c.key(key))
	if err != nil {
		return nil, err
	}

	val, err := cmd.Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	// 尝试反序列化JSON
	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		// 不是JSON，直接返回字符串
		return val, nil
	}

	return result, nil
}

// Set 设置缓存值
func (c *UserCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	val, err := c.marshalValue(value)
	if err != nil {
		return err
	}

	cmd := c.client.Set(ctx, c.key(key), val, expiration)
	return cmd.Err()
}

// Delete 删除缓存键
func (c *UserCache) Delete(ctx context.Context, keys ...string) error {
	cacheKeys := make([]string, len(keys))
	for i, key := range keys {
		cacheKeys[i] = c.key(key)
	}

	cmd := c.client.Del(ctx, cacheKeys...)
	return cmd.Err()
}

// Exists 检查缓存键是否存在
func (c *UserCache) Exists(ctx context.Context, key string) (bool, error) {
	cmd := c.client.Exists(ctx, c.key(key))
	val, err := cmd.Result()
	if err != nil {
		return false, err
	}

	return val > 0, nil
}

// GetMulti 批量获取缓存值
func (c *UserCache) GetMulti(ctx context.Context, keys []string) (map[string]interface{}, error) {
	cacheKeys := make([]string, len(keys))
	keyMap := make(map[string]string, len(keys))

	for i, key := range keys {
		cacheKey := c.key(key)
		cacheKeys[i] = cacheKey
		keyMap[cacheKey] = key
	}

	cmd := c.client.MGet(ctx, cacheKeys...)
	vals, err := cmd.Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{}, len(keys))
	for i, cacheKey := range cacheKeys {
		if i < len(vals) && vals[i] != nil {
			// 尝试反序列化JSON
			var value interface{}
			if err := json.Unmarshal([]byte(vals[i].(string)), &value); err != nil {
				// 不是JSON，直接返回字符串
				result[keyMap[cacheKey]] = vals[i]
			} else {
				result[keyMap[cacheKey]] = value
			}
		}
	}

	return result, nil
}

// SetMulti 批量设置缓存值
func (c *UserCache) SetMulti(ctx context.Context, items map[string]interface{}, expiration time.Duration) error {
	values := make([]interface{}, 0, len(items)*2)

	for key, value := range items {
		val, err := c.marshalValue(value)
		if err != nil {
			return err
		}

		values = append(values, c.key(key), val)
	}

	cmd := c.client.MSet(ctx, values...)
	if err := cmd.Err(); err != nil {
		return err
	}

	// 设置过期时间
	for key := range items {
		cmd := c.client.Expire(ctx, c.key(key), expiration)
		if err := cmd.Err(); err != nil {
			return err
		}
	}

	return nil
}

// Incr 递增计数器
func (c *UserCache) Incr(ctx context.Context, key string) (int64, error) {
	cmd := c.client.Incr(ctx, c.key(key))
	return cmd.Result()
}

// Decr 递减计数器
func (c *UserCache) Decr(ctx context.Context, key string) (int64, error) {
	cmd := c.client.Decr(ctx, c.key(key))
	return cmd.Result()
}

// Expire 设置缓存过期时间
func (c *UserCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	cmd := c.client.Expire(ctx, c.key(key), expiration)
	return cmd.Err()
}

// TTL 获取缓存剩余过期时间
func (c *UserCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, c.key(key))
}

// Clear 清空缓存
func (c *UserCache) Clear(ctx context.Context, pattern string) error {
	// 这里简化实现，实际应该使用SCAN命令
	return nil
}

// GetUserInfo 获取用户信息
func (c *UserCache) GetUserInfo(ctx context.Context, userID string) (map[string]interface{}, error) {
	val, err := c.Get(ctx, fmt.Sprintf("info:%s", userID))
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, nil
	}

	info, ok := val.(map[string]interface{})
	if !ok {
		// 尝试类型转换
		if str, ok := val.(string); ok {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(str), &result); err == nil {
				return result, nil
			}
		}
		return nil, fmt.Errorf("invalid user info format")
	}

	return info, nil
}

// SetUserInfo 设置用户信息
func (c *UserCache) SetUserInfo(ctx context.Context, userID string, info map[string]interface{}, expiration time.Duration) error {
	return c.Set(ctx, fmt.Sprintf("info:%s", userID), info, expiration)
}

// GetUserSession 获取用户会话
func (c *UserCache) GetUserSession(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	val, err := c.Get(ctx, fmt.Sprintf("session:%s", sessionID))
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, nil
	}

	session, ok := val.(map[string]interface{})
	if !ok {
		// 尝试类型转换
		if str, ok := val.(string); ok {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(str), &result); err == nil {
				return result, nil
			}
		}
		return nil, fmt.Errorf("invalid session format")
	}

	return session, nil
}

// SetUserSession 设置用户会话
func (c *UserCache) SetUserSession(ctx context.Context, sessionID string, session map[string]interface{}, expiration time.Duration) error {
	return c.Set(ctx, fmt.Sprintf("session:%s", sessionID), session, expiration)
}

// DeleteUserInfo 删除用户信息
func (c *UserCache) DeleteUserInfo(ctx context.Context, userID string) error {
	return c.Delete(ctx, fmt.Sprintf("info:%s", userID))
}

// DeleteUserSession 删除用户会话
func (c *UserCache) DeleteUserSession(ctx context.Context, sessionID string) error {
	return c.Delete(ctx, fmt.Sprintf("session:%s", sessionID))
}

// marshalValue 序列化值
func (c *UserCache) marshalValue(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int, int64, float64, bool:
		return fmt.Sprintf("%v", v), nil
	default:
		// 其他类型序列化为JSON
		data, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}
