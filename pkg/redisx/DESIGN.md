# RedisX 设计文档

## 概述

RedisX 是一个基于 go-redis 的 Redis 操作封装库，旨在提供统一、易用的 Redis 操作接口，屏蔽不同部署架构的差异性。

## 设计目标

1. **统一API**：无论底层是单节点、主从、集群还是多主多从，对外提供统一的操作接口
2. **配置优先**：通过配置文件管理 Redis 连接，支持多种部署模式
3. **功能模块化**：将功能划分为服务器缓存、用户数据缓存、锁三个独立模块
4. **高级特性支持**：支持看门狗、Lua脚本、管道、事务、Watch等高级操作
5. **易于使用**：简化 Redis 操作，提供友好的错误处理和日志记录

## 架构设计

### 目录结构

```
redisx/
├── README.md                    # 项目说明文档
├── DESIGN.md                    # 本设计文档
├── go.mod                       # Go模块定义
├── config/                      # 配置管理
│   ├── config.go               # 配置结构和加载
│   └── loader.go               # 配置加载器（支持文件、环境变量等）
├── client/                      # Redis客户端封装
│   ├── client.go               # 客户端接口定义
│   ├── single.go               # 单节点实现
│   ├── sentinel.go             # 哨兵模式实现
│   ├── cluster.go              # 集群模式实现
│   └── multi_master.go         # 多主多从模式实现
├── cache/                       # 缓存模块
│   ├── cache.go                # 缓存接口定义
│   ├── server_cache.go         # 服务器缓存实现
│   └── user_cache.go           # 用户数据缓存实现
├── lock/                        # 锁模块
│   ├── lock.go                 # 锁接口定义
│   ├── single_lock.go          # 单锁实现
│   ├── redlock.go              # 红锁实现
│   └── watchdog.go             # 看门狗机制
├── advanced/                    # 高级操作模块
│   ├── lua.go                  # Lua脚本支持
│   ├── pipeline.go             # 管道操作
│   ├── transaction.go          # 事务操作
│   └── watch.go                # Watch操作
├── errors/                      # 错误处理
│   └── errors.go               # 自定义错误类型
└── utils/                       # 工具函数
    ├── key.go                  # Key生成工具
    └── retry.go                # 重试机制
```

## 核心模块设计

### 1. 配置模块 (config/)

#### 配置结构

```go
type Config struct {
    Mode         string            // 部署模式: single, sentinel, cluster, multi-master
    Debug        bool              // 是否开启调试模式
    
    // 单节点配置
    Single *SingleConfig          `json:"single,omitempty"`
    
    // 哨兵配置
    Sentinel *SentinelConfig       `json:"sentinel,omitempty"`
    
    // 集群配置
    Cluster *ClusterConfig         `json:"cluster,omitempty"`
    
    // 多主多从配置
    MultiMaster *MultiMasterConfig `json:"multi-master,omitempty"`
    
    // 通用配置
    Password     string            `json:"password,omitempty"`
    DB           int               `json:"db"`
    PoolSize     int               `json:"pool_size"`
    MinIdleConns int               `json:"min_idle_conns"`
    MaxRetries   int               `json:"max_retries"`
    DialTimeout  time.Duration     `json:"dial_timeout"`
    ReadTimeout  time.Duration     `json:"read_timeout"`
    WriteTimeout time.Duration     `json:"write_timeout"`
    PoolTimeout  time.Duration     `json:"pool_timeout"`
}

type SingleConfig struct {
    Addr string `json:"addr"`      // 例如: "127.0.0.1:6379"
}

type SentinelConfig struct {
    MasterName   string   `json:"master_name"`
    SentinelAddrs []string `json:"sentinel_addrs"`
}

type ClusterConfig struct {
    Addrs []string `json:"addrs"`  // 集群节点地址列表
}

type MultiMasterConfig struct {
    Masters []MasterConfig `json:"masters"`
}

type MasterConfig struct {
    Addr   string   `json:"addr"`
    Slaves []string `json:"slaves,omitempty"`
}
```

#### 配置加载

支持从以下来源加载配置：
- JSON/YAML/TOML 文件
- 环境变量
- 编码的配置字符串

### 2. 客户端模块 (client/)

#### 客户端接口

```go
type Client interface {
    // 基础操作
    Get(ctx context.Context, key string) (*redis.StringCmd, error)
    Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
    Del(ctx context.Context, keys ...string) *redis.IntCmd
    Exists(ctx context.Context, keys ...string) *redis.IntCmd
    Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
    TTL(ctx context.Context, key string) *redis.DurationCmd
    
    // 批量操作
    MGet(ctx context.Context, keys ...string) *redis.SliceCmd
    MSet(ctx context.Context, values ...interface{}) *redis.StatusCmd
    
    // 连接管理
    Ping(ctx context.Context) *redis.StatusCmd
    Close() error
    
    // 获取底层客户端
    GetClient() interface{}
}
```

#### 部署模式实现

**单节点模式 (single.go)**
```go
type SingleClient struct {
    client *redis.Client
    config *config.SingleConfig
}

func NewSingleClient(cfg *config.SingleConfig, opts *config.Config) (*SingleClient, error)
```

**哨兵模式 (sentinel.go)**
```go
type SentinelClient struct {
    client *redis.Client
    config *config.SentinelConfig
}

func NewSentinelClient(cfg *config.SentinelConfig, opts *config.Config) (*SentinelClient, error)
```

**集群模式 (cluster.go)**
```go
type ClusterClient struct {
    client *redis.ClusterClient
    config *config.ClusterConfig
}

func NewClusterClient(cfg *config.ClusterConfig, opts *config.Config) (*ClusterClient, error)
```

**多主多从模式 (multi_master.go)**
```go
type MultiMasterClient struct {
    masters []*redis.Client
    slaves  []*redis.Client
    config  *config.MultiMasterConfig
    router  *Router  // 读写路由
}

func NewMultiMasterClient(cfg *config.MultiMasterConfig, opts *config.Config) (*MultiMasterClient, error)
```

### 3. 缓存模块 (cache/)

#### 缓存接口

```go
type Cache interface {
    // 基础缓存操作
    Get(ctx context.Context, key string) (interface{}, error)
    Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
    Delete(ctx context.Context, keys ...string) error
    Exists(ctx context.Context, key string) (bool, error)
    
    // 批量操作
    GetMulti(ctx context.Context, keys []string) (map[string]interface{}, error)
    SetMulti(ctx context.Context, items map[string]interface{}, expiration time.Duration) error
    
    // 计数器
    Incr(ctx context.Context, key string) (int64, error)
    Decr(ctx context.Context, key string) (int64, error)
    
    // 过期时间
    Expire(ctx context.Context, key string, expiration time.Duration) error
    TTL(ctx context.Context, key string) (time.Duration, error)
    
    // 清空
    Clear(ctx context.Context, pattern string) error
}
```

#### 服务器缓存 (server_cache.go)

专门用于服务器级别的缓存，如配置信息、系统状态等。

```go
type ServerCache struct {
    client client.Client
    prefix string
}

func NewServerCache(client client.Client, prefix string) *ServerCache

// 特定方法
func (sc *ServerCache) GetConfig(ctx context.Context, key string) (string, error)
func (sc *ServerCache) SetConfig(ctx context.Context, key string, value string, expiration time.Duration) error
func (sc *ServerCache) GetSystemStatus(ctx context.Context) (map[string]interface{}, error)
```

#### 用户数据缓存 (user_cache.go)

专门用于用户数据的缓存，如用户信息、会话数据等。

```go
type UserCache struct {
    client client.Client
    prefix string
}

func NewUserCache(client client.Client, prefix string) *UserCache

// 特定方法
func (uc *UserCache) GetUserInfo(ctx context.Context, userID string) (map[string]interface{}, error)
func (uc *UserCache) SetUserInfo(ctx context.Context, userID string, info map[string]interface{}, expiration time.Duration) error
func (uc *UserCache) GetUserSession(ctx context.Context, sessionID string) (map[string]interface{}, error)
func (uc *UserCache) SetUserSession(ctx context.Context, sessionID string, session map[string]interface{}, expiration time.Duration) error
```

### 4. 锁模块 (lock/)

#### 锁接口

```go
type Lock interface {
    Lock(ctx context.Context) (bool, error)
    Unlock(ctx context.Context) error
    TryLock(ctx context.Context) (bool, error)
    Extend(ctx context.Context, expiration time.Duration) (bool, error)
}
```

#### 单锁实现 (single_lock.go)

基于 Redis 的 SET NX EX 命令实现的分布式锁。

```go
type SingleLock struct {
    client     client.Client
    key        string
    value      string
    expiration time.Duration
    watchdog   *Watchdog
}

func NewSingleLock(client client.Client, key string, expiration time.Duration) *SingleLock

func (sl *SingleLock) Lock(ctx context.Context) (bool, error)
func (sl *SingleLock) Unlock(ctx context.Context) error
func (sl *SingleLock) TryLock(ctx context.Context) (bool, error)
func (sl *SingleLock) Extend(ctx context.Context, expiration time.Duration) (bool, error)
```

#### 红锁实现 (redlock.go)

基于 Redlock 算法的分布式锁，支持多个 Redis 实例。

```go
type RedLock struct {
    clients []client.Client
    locks   []*SingleLock
    quorum  int
    key     string
    value   string
    expiration time.Duration
}

func NewRedLock(clients []client.Client, key string, expiration time.Duration) *RedLock

func (rl *RedLock) Lock(ctx context.Context) (bool, error)
func (rl *RedLock) Unlock(ctx context.Context) error
func (rl *RedLock) TryLock(ctx context.Context) (bool, error)
```

#### 看门狗机制 (watchdog.go)

自动续期机制，防止锁过期。

```go
type Watchdog struct {
    lock      Lock
    interval  time.Duration
    stopChan  chan struct{}
    expiration time.Duration
}

func NewWatchdog(lock Lock, interval time.Duration) *Watchdog

func (w *Watchdog) Start(ctx context.Context)
func (w *Watchdog) Stop()
```

### 5. 高级操作模块 (advanced/)

#### Lua脚本支持 (lua.go)

```go
type LuaManager struct {
    client client.Client
    scripts map[string]*redis.Script
}

func NewLuaManager(client client.Client) *LuaManager

func (lm *LuaManager) RegisterScript(name string, script string)
func (lm *LuaManager) Execute(ctx context.Context, name string, keys []string, args ...interface{}) (interface{}, error)
```

#### 管道操作 (pipeline.go)

```go
type PipelineManager struct {
    client client.Client
}

func NewPipelineManager(client client.Client) *PipelineManager

func (pm *PipelineManager) Execute(ctx context.Context, fn func(pipe redis.Pipeliner) error) ([]redis.Cmder, error)
```

#### 事务操作 (transaction.go)

```go
type TransactionManager struct {
    client client.Client
}

func NewTransactionManager(client client.Client) *TransactionManager

func (tm *TransactionManager) Execute(ctx context.Context, fn func(tx redis.Tx) error) error
```

#### Watch操作 (watch.go)

```go
type WatchManager struct {
    client client.Client
}

func NewWatchManager(client client.Client) *WatchManager

func (wm *WatchManager) Watch(ctx context.Context, keys []string, fn func(tx *redis.Tx) error) error
```

### 6. 错误处理模块 (errors/)

```go
var (
    ErrLockNotAcquired = errors.New("lock: lock not acquired")
    ErrLockNotHeld     = errors.New("lock: lock not held")
    ErrConfigInvalid   = errors.New("config: invalid configuration")
    ErrConnectionFailed = errors.New("connection: connection failed")
)

type RedisError struct {
    Code    string
    Message string
    Err     error
}

func (e *RedisError) Error() string
func (e *RedisError) Unwrap() error
```

### 7. 工具模块 (utils/)

#### Key生成工具 (key.go)

```go
type KeyGenerator struct {
    prefix string
    separator string
}

func NewKeyGenerator(prefix string) *KeyGenerator

func (kg *KeyGenerator) Generate(parts ...string) string
func (kg *KeyGenerator) Parse(key string) []string
```

#### 重试机制 (retry.go)

```go
type RetryPolicy struct {
    MaxAttempts int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
}

func Retry(ctx context.Context, policy RetryPolicy, fn func() error) error
```

## 统一API设计

### RedisX 主入口

```go
type RedisX struct {
    client    client.Client
    config    *config.Config
    serverCache *cache.ServerCache
    userCache   *cache.UserCache
}

func New(cfg *config.Config) (*RedisX, error)
func (rx *RedisX) Close() error

// 获取各个模块
func (rx *RedisX) ServerCache() *cache.ServerCache
func (rx *RedisX) UserCache() *cache.UserCache
func (rx *RedisX) NewLock(key string, expiration time.Duration) lock.Lock
func (rx *RedisX) NewRedLock(key string, expiration time.Duration) lock.Lock
func (rx *RedisX) Lua() *advanced.LuaManager
func (rx *RedisX) Pipeline() *advanced.PipelineManager
func (rx *RedisX) Transaction() *advanced.TransactionManager
func (rx *RedisX) Watch() *advanced.WatchManager
```

## 使用示例

### 初始化

```go
// 从配置文件初始化
cfg, err := config.LoadFromFile("redis.yaml")
if err != nil {
    log.Fatal(err)
}

redisx, err := redisx.New(cfg)
if err != nil {
    log.Fatal(err)
}
defer redisx.Close()
```

### 服务器缓存使用

```go
sc := redisx.ServerCache()

// 设置配置
err := sc.SetConfig(ctx, "app.timeout", "30s", time.Hour)
if err != nil {
    log.Fatal(err)
}

// 获取配置
timeout, err := sc.GetConfig(ctx, "app.timeout")
if err != nil {
    log.Fatal(err)
}
fmt.Println(timeout) // "30s"
```

### 用户数据缓存使用

```go
uc := redisx.UserCache()

// 设置用户信息
userInfo := map[string]interface{}{
    "name": "张三",
    "age":  30,
    "email": "zhangsan@example.com",
}
err := uc.SetUserInfo(ctx, "user123", userInfo, time.Hour)
if err != nil {
    log.Fatal(err)
}

// 获取用户信息
info, err := uc.GetUserInfo(ctx, "user123")
if err != nil {
    log.Fatal(err)
}
fmt.Println(info["name"]) // "张三"
```

### 单锁使用

```go
l := redisx.NewLock("my_lock", 10*time.Second)

// 获取锁
acquired, err := l.Lock(ctx)
if err != nil {
    log.Fatal(err)
}
if !acquired {
    fmt.Println("获取锁失败")
    return
}
defer l.Unlock(ctx)

// 启动看门狗
watchdog := lock.NewWatchdog(l, 5*time.Second)
watchdog.Start(ctx)
defer watchdog.Stop()

// 执行业务逻辑
fmt.Println("执行业务逻辑...")
```

### 红锁使用

```go
rl := redisx.NewRedLock("my_redlock", 10*time.Second)

// 获取红锁
acquired, err := rl.Lock(ctx)
if err != nil {
    log.Fatal(err)
}
if !acquired {
    fmt.Println("获取红锁失败")
    return
}
defer rl.Unlock(ctx)

// 执行业务逻辑
fmt.Println("执行业务逻辑...")
```

### Lua脚本使用

```go
lua := redisx.Lua()

// 注册脚本
lua.RegisterScript("increment", `
    local key = KEYS[1]
    local increment = tonumber(ARGV[1])
    local current = redis.call("GET", key) or 0
    local new = current + increment
    redis.call("SET", key, new)
    return new
`)

// 执行脚本
result, err := lua.Execute(ctx, "increment", []string{"counter"}, 10)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result) // 10
```

### 管道操作使用

```go
pipeline := redisx.Pipeline()

results, err := pipeline.Execute(ctx, func(pipe redis.Pipeliner) error {
    pipe.Set(ctx, "key1", "value1", 0)
    pipe.Set(ctx, "key2", "value2", 0)
    pipe.Get(ctx, "key1")
    return nil
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(results[2].(*redis.StringCmd).Val()) // "value1"
```

### 事务操作使用

```go
txManager := redisx.Transaction()

err := txManager.Execute(ctx, func(tx redis.Tx) error {
    _, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
        pipe.Set(ctx, "key1", "value1", 0)
        pipe.Set(ctx, "key2", "value2", 0)
        return nil
    })
    return err
})
if err != nil {
    log.Fatal(err)
}
```

### Watch操作使用

```go
watchManager := redisx.Watch()

err := watchManager.Watch(ctx, []string{"counter"}, func(tx *redis.Tx) error {
    n, err := tx.Get(ctx, "counter").Int()
    if err != nil && err != redis.Nil {
        return err
    }
    
    _, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
        pipe.Set(ctx, "counter", n+1, 0)
        return nil
    })
    return err
})
if err != nil {
    log.Fatal(err)
}
```

## 配置示例

### 单节点配置 (redis.yaml)

```yaml
mode: single
debug: false
password: ""
db: 0
pool_size: 10
min_idle_conns: 5
max_retries: 3
dial_timeout: 5s
read_timeout: 3s
write_timeout: 3s
pool_timeout: 4s

single:
  addr: "127.0.0.1:6379"
```

### 哨兵配置 (redis.yaml)

```yaml
mode: sentinel
debug: false
password: ""
db: 0
pool_size: 10
min_idle_conns: 5
max_retries: 3
dial_timeout: 5s
read_timeout: 3s
write_timeout: 3s
pool_timeout: 4s

sentinel:
  master_name: "mymaster"
  sentinel_addrs:
    - "127.0.0.1:26379"
    - "127.0.0.1:26380"
    - "127.0.0.1:26381"
```

### 集群配置 (redis.yaml)

```yaml
mode: cluster
debug: false
password: ""
db: 0
pool_size: 10
min_idle_conns: 5
max_retries: 3
dial_timeout: 5s
read_timeout: 3s
write_timeout: 3s
pool_timeout: 4s

cluster:
  addrs:
    - "127.0.0.1:7000"
    - "127.0.0.1:7001"
    - "127.0.0.1:7002"
    - "127.0.0.1:7003"
    - "127.0.0.1:7004"
    - "127.0.0.1:7005"
```

### 多主多从配置 (redis.yaml)

```yaml
mode: multi-master
debug: false
password: ""
db: 0
pool_size: 10
min_idle_conns: 5
max_retries: 3
dial_timeout: 5s
read_timeout: 3s
write_timeout: 3s
pool_timeout: 4s

multi-master:
  masters:
    - addr: "127.0.0.1:6379"
      slaves:
        - "127.0.0.1:6380"
        - "127.0.0.1:6381"
    - addr: "127.0.0.1:6382"
      slaves:
        - "127.0.0.1:6383"
        - "127.0.0.1:6384"
```

## 性能优化

1. **连接池管理**：合理配置连接池大小，避免连接泄漏
2. **批量操作**：使用 MGET/MSET 等批量命令减少网络往返
3. **管道操作**：使用 Pipeline 减少网络往返
4. **Lua脚本**：将复杂操作封装为 Lua 脚本，减少网络往返
5. **读写分离**：在主从模式下，读操作路由到从节点
6. **本地缓存**：对热点数据使用本地缓存减少 Redis 访问

## 错误处理

1. **重试机制**：对网络错误实现自动重试
2. **超时控制**：合理设置各种超时参数
3. **错误日志**：记录详细的错误信息便于排查
4. **降级策略**：在 Redis 不可用时提供降级方案

## 测试策略

1. **单元测试**：对每个模块进行单元测试
2. **集成测试**：测试不同部署模式的集成
3. **压力测试**：测试高并发场景下的性能
4. **故障测试**：测试网络故障、节点故障等场景

## 后续扩展

1. **监控指标**：添加 Prometheus 监控指标
2. **链路追踪**：集成 OpenTelemetry 进行分布式追踪
3. **缓存预热**：支持缓存预热功能
4. **缓存击穿保护**：实现缓存击穿保护机制
5. **缓存雪崩保护**：实现缓存雪崩保护机制

## 总结

RedisX 通过统一的 API 设计，屏蔽了不同 Redis 部署模式的差异性，提供了简单易用的缓存、锁和高级操作接口。配置优先的设计使得部署和运维更加简单，模块化的架构便于维护和扩展。
