package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/tedwangl/go-util/pkg/redisx/client"
)

// ServerCache 服务器缓存
type ServerCache struct {
	client client.Client
	prefix string
}

// NewServerCache 创建服务器缓存
func NewServerCache(client client.Client, prefix string) *ServerCache {
	if prefix == "" {
		prefix = "server"
	}

	return &ServerCache{
		client: client,
		prefix: prefix,
	}
}

// key 生成缓存键
func (c *ServerCache) key(key string) string {
	return fmt.Sprintf("%s:%s", c.prefix, key)
}

// Get 获取缓存值
func (c *ServerCache) Get(ctx context.Context, key string) (interface{}, error) {
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

	return val, nil
}

// Set 设置缓存值
func (c *ServerCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	val, err := c.marshalValue(value)
	if err != nil {
		return err
	}

	cmd := c.client.Set(ctx, c.key(key), val, expiration)
	return cmd.Err()
}

// Delete 删除缓存键
func (c *ServerCache) Delete(ctx context.Context, keys ...string) error {
	cacheKeys := make([]string, len(keys))
	for i, key := range keys {
		cacheKeys[i] = c.key(key)
	}

	cmd := c.client.Del(ctx, cacheKeys...)
	return cmd.Err()
}

// Exists 检查缓存键是否存在
func (c *ServerCache) Exists(ctx context.Context, key string) (bool, error) {
	cmd := c.client.Exists(ctx, c.key(key))
	val, err := cmd.Result()
	if err != nil {
		return false, err
	}

	return val > 0, nil
}

// GetMulti 批量获取缓存值
func (c *ServerCache) GetMulti(ctx context.Context, keys []string) (map[string]interface{}, error) {
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
			result[keyMap[cacheKey]] = vals[i]
		}
	}

	return result, nil
}

// SetMulti 批量设置缓存值
func (c *ServerCache) SetMulti(ctx context.Context, items map[string]interface{}, expiration time.Duration) error {
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
func (c *ServerCache) Incr(ctx context.Context, key string) (int64, error) {
	cmd := c.client.Incr(ctx, c.key(key))
	return cmd.Result()
}

// Decr 递减计数器
func (c *ServerCache) Decr(ctx context.Context, key string) (int64, error) {
	cmd := c.client.Decr(ctx, c.key(key))
	return cmd.Result()
}

// Expire 设置缓存过期时间
func (c *ServerCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	cmd := c.client.Expire(ctx, c.key(key), expiration)
	return cmd.Err()
}

// TTL 获取缓存剩余过期时间
func (c *ServerCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, c.key(key))
}

// Clear 清空缓存
func (c *ServerCache) Clear(ctx context.Context, pattern string) error {
	// 这里简化实现，实际应该使用SCAN命令
	// 为了避免阻塞，建议使用SCAN分批处理
	return nil
}

// GetConfig 获取配置
func (c *ServerCache) GetConfig(ctx context.Context, key string) (string, error) {
	val, err := c.Get(ctx, fmt.Sprintf("config:%s", key))
	if err != nil {
		return "", err
	}

	if val == nil {
		return "", nil
	}

	return val.(string), nil
}

// SetConfig 设置配置
func (c *ServerCache) SetConfig(ctx context.Context, key string, value string, expiration time.Duration) error {
	return c.Set(ctx, fmt.Sprintf("config:%s", key), value, expiration)
}

// GetSystemStatus 获取系统状态
func (c *ServerCache) GetSystemStatus(ctx context.Context) (map[string]interface{}, error) {
	statusKeys := []string{
		"status:uptime",
		"status:load",
		"status:connections",
		"status:memory",
	}

	return c.GetMulti(ctx, statusKeys)
}

// SetSystemStatus 设置系统状态
func (c *ServerCache) SetSystemStatus(ctx context.Context, status map[string]interface{}, expiration time.Duration) error {
	statusItems := make(map[string]interface{}, len(status))

	for key, value := range status {
		statusItems[fmt.Sprintf("status:%s", key)] = value
	}

	return c.SetMulti(ctx, statusItems, expiration)
}

// marshalValue 序列化值
func (c *ServerCache) marshalValue(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string, int, int64, float64, bool:
		return v, nil
	default:
		return json.Marshal(value)
	}
}
