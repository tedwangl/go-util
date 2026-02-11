package advanced

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/tedwangl/go-util/pkg/redisx/client"
)

type LuaScript struct {
	script string
	sha1   string
	client client.Client
}

func NewLuaScript(script string, cli client.Client) *LuaScript {
	return &LuaScript{
		script: script,
		client: cli,
	}
}

func (ls *LuaScript) Load(ctx context.Context) (string, error) {
	cmd := ls.client.Eval(ctx, fmt.Sprintf("return redis.call('SCRIPT', 'LOAD', %q)", ls.script), []string{})
	result, err := cmd.Result()
	if err != nil {
		return "", fmt.Errorf("failed to load lua script: %w", err)
	}
	sha1, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected result type from script load")
	}
	ls.sha1 = sha1
	return sha1, nil
}

func (ls *LuaScript) Exec(ctx context.Context, keys []string, args ...interface{}) (*redis.Cmd, error) {
	if ls.sha1 != "" {
		cmd := ls.client.EvalSha(ctx, ls.sha1, keys, args...)
		if cmd.Err() == nil {
			return cmd, nil
		}
	}
	return ls.client.Eval(ctx, ls.script, keys, args...), nil
}

func (ls *LuaScript) ExecSha(ctx context.Context, keys []string, args ...interface{}) (*redis.Cmd, error) {
	if ls.sha1 == "" {
		_, err := ls.Load(ctx)
		if err != nil {
			return nil, err
		}
	}
	cmd := ls.client.EvalSha(ctx, ls.sha1, keys, args...)
	if cmd.Err() != nil {
		return nil, fmt.Errorf("failed to execute lua script by sha1: %w", cmd.Err())
	}
	return cmd, nil
}

func (ls *LuaScript) GetScript() string {
	return ls.script
}

func (ls *LuaScript) GetSHA1() string {
	return ls.sha1
}

func (ls *LuaScript) Exists(ctx context.Context) (bool, error) {
	if ls.sha1 == "" {
		return false, nil
	}
	cmd := ls.client.Eval(ctx, "return redis.call('SCRIPT', 'EXISTS', ARGV[1])", []string{}, ls.sha1)
	result, err := cmd.Result()
	if err != nil {
		return false, fmt.Errorf("failed to check script exists: %w", err)
	}
	if arr, ok := result.([]interface{}); ok && len(arr) > 0 {
		if exists, ok := arr[0].(int64); ok {
			return exists == 1, nil
		}
	}
	return false, nil
}

func (ls *LuaScript) Flush(ctx context.Context) error {
	cmd := ls.client.Eval(ctx, "return redis.call('SCRIPT', 'FLUSH')", []string{})
	if cmd.Err() != nil {
		return fmt.Errorf("failed to flush scripts: %w", cmd.Err())
	}
	return nil
}

type ScriptManager struct {
	client  client.Client
	scripts map[string]*LuaScript
}

func NewScriptManager(cli client.Client) *ScriptManager {
	return &ScriptManager{
		client:  cli,
		scripts: make(map[string]*LuaScript),
	}
}

func (sm *ScriptManager) Register(name, script string) error {
	ls := NewLuaScript(script, sm.client)
	sm.scripts[name] = ls
	return nil
}

func (sm *ScriptManager) Load(ctx context.Context, name string) (string, error) {
	ls, ok := sm.scripts[name]
	if !ok {
		return "", fmt.Errorf("script '%s' not found", name)
	}
	return ls.Load(ctx)
}

func (sm *ScriptManager) LoadAll(ctx context.Context) error {
	for name := range sm.scripts {
		_, err := sm.Load(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to load script '%s': %w", name, err)
		}
	}
	return nil
}

func (sm *ScriptManager) Exec(ctx context.Context, name string, keys []string, args ...interface{}) (*redis.Cmd, error) {
	ls, ok := sm.scripts[name]
	if !ok {
		return nil, fmt.Errorf("script '%s' not found", name)
	}
	return ls.Exec(ctx, keys, args...)
}

func (sm *ScriptManager) ExecSha(ctx context.Context, name string, keys []string, args ...interface{}) (*redis.Cmd, error) {
	ls, ok := sm.scripts[name]
	if !ok {
		return nil, fmt.Errorf("script '%s' not found", name)
	}
	return ls.ExecSha(ctx, keys, args...)
}

func (sm *ScriptManager) GetScript(name string) (*LuaScript, error) {
	ls, ok := sm.scripts[name]
	if !ok {
		return nil, fmt.Errorf("script '%s' not found", name)
	}
	return ls, nil
}

func (sm *ScriptManager) Remove(name string) {
	delete(sm.scripts, name)
}

func (sm *ScriptManager) Clear() {
	sm.scripts = make(map[string]*LuaScript)
}

const (
	ScriptIncrByFloat = `
local key = KEYS[1]
local delta = tonumber(ARGV[1])
local current = redis.call('GET', key)
if current then
	current = tonumber(current)
	if current then
		return redis.call('SET', key, current + delta)
	end
end
return redis.call('SET', key, delta)
`

	ScriptExpireAt = `
local key = KEYS[1]
local timestamp = tonumber(ARGV[1])
if redis.call('EXISTS', key) == 1 then
	return redis.call('EXPIREAT', key, timestamp)
end
return 0
`

	ScriptGetOrSet = `
local key = KEYS[1]
local value = ARGV[1]
local ttl = tonumber(ARGV[2])
local current = redis.call('GET', key)
if current then
	return current
end
if ttl and ttl > 0 then
	redis.call('SETEX', key, ttl, value)
else
	redis.call('SET', key, value)
end
return value
`

	ScriptBatchDelete = `
local count = 0
for i = 1, #KEYS do
	if redis.call('DEL', KEYS[i]) == 1 then
		count = count + 1
	end
end
return count
`

	ScriptHGetAllOrDefault = `
local key = KEYS[1]
local default = ARGV[1]
local result = redis.call('HGETALL', key)
if #result == 0 and default then
	return cjson.decode(default)
end
return result
`

	ScriptZAddIfNotExists = `
local key = KEYS[1]
local score = tonumber(ARGV[1])
local member = ARGV[2]
local current = redis.call('ZSCORE', key, member)
if current then
	return 0
end
	redis.call('ZADD', key, score, member)
	return 1
`

	ScriptZAddOrUpdate = `
local key = KEYS[1]
local score = tonumber(ARGV[1])
local member = ARGV[2]
local current = redis.call('ZSCORE', key, member)
	if current then
		redis.call('ZADD', key, score, member)
		return 0
	end
	redis.call('ZADD', key, score, member)
	return 1
`

	ScriptDistributedLock = `
local key = KEYS[1]
local value = ARGV[1]
local ttl = tonumber(ARGV[2])
local result = redis.call('SET', key, value, 'NX', 'PX', ttl)
if result then
	return 1
end
return 0
`

	ScriptReleaseLock = `
local key = KEYS[1]
local value = ARGV[1]
local current = redis.call('GET', key)
if current == value then
	return redis.call('DEL', key)
end
return 0
`

	ScriptExtendLock = `
local key = KEYS[1]
local value = ARGV[1]
local ttl = tonumber(ARGV[2])
local current = redis.call('GET', key)
if current == value then
	return redis.call('PEXPIRE', key, ttl)
end
return 0
`

	ScriptRateLimit = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local current = redis.call('INCR', key)
	if current == 1 then
		redis.call('EXPIRE', key, window)
	end
	if current > limit then
		return 0
	end
	return 1
`

	ScriptSlidingWindowRateLimit = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)
local count = redis.call('ZCARD', key)
if count < limit then
	redis.call('ZADD', key, now, now)
	return 1
end
return 0
`
)

func RegisterCommonScripts(sm *ScriptManager) {
	sm.Register("incr_by_float", ScriptIncrByFloat)
	sm.Register("expire_at", ScriptExpireAt)
	sm.Register("get_or_set", ScriptGetOrSet)
	sm.Register("batch_delete", ScriptBatchDelete)
	sm.Register("hgetall_or_default", ScriptHGetAllOrDefault)
	sm.Register("zadd_if_not_exists", ScriptZAddIfNotExists)
	sm.Register("zadd_or_update", ScriptZAddOrUpdate)
	sm.Register("distributed_lock", ScriptDistributedLock)
	sm.Register("release_lock", ScriptReleaseLock)
	sm.Register("extend_lock", ScriptExtendLock)
	sm.Register("rate_limit", ScriptRateLimit)
	sm.Register("sliding_window_rate_limit", ScriptSlidingWindowRateLimit)
}
