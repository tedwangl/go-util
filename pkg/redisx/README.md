# RedisX

RedisX 是一个基于 go-redis 的 Redis 操作封装库，提供统一、易用的 Redis 操作接口，屏蔽不同部署架构的差异性。

## 特性

- **统一API**：支持单节点、主从、集群、多主多从等多种部署模式
- **配置优先**：通过配置文件管理 Redis 连接，简化部署和运维
- **功能模块化**：
  - 服务器缓存：用于服务器级别的缓存（配置信息、系统状态等）
  - 用户数据缓存：用于用户数据的缓存（用户信息、会话数据等）
  - 锁机制：支持单锁和红锁，包含看门狗自动续期
- **高级操作**：支持 Lua 脚本、管道、事务、Watch 等高级特性
- **易于使用**：简化 Redis 操作，提供友好的错误处理和日志记录

## 安装

```bash
go get github.com/beego/beego/v2/redisx
```

## 快速开始

### 1. 创建配置文件

创建 `redis.yaml` 配置文件：

```yaml
mode: single
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

### 2. 初始化 RedisX

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/beego/beego/v2/redisx"
    "github.com/tedwangl/go-util/pkg/redisx/config"
)

func main() {
    // 从配置文件加载
    cfg, err := config.LoadFromFile("redis.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // 初始化 RedisX
    rx, err := redisx.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer rx.Close()
    
    // 使用 RedisX
    ctx := context.Background()
    
    // 服务器缓存示例
    sc := rx.ServerCache()
    err = sc.SetConfig(ctx, "app.timeout", "30s", time.Hour)
    if err != nil {
        log.Fatal(err)
    }
    
    timeout, err := sc.GetConfig(ctx, "app.timeout")
    if err != nil {
        log.Fatal(err)
    }
    log.Println("Timeout:", timeout)
}
```

## 部署模式

### 单节点模式

```yaml
mode: single
single:
  addr: "127.0.0.1:6379"
```

### 哨兵模式

```yaml
mode: sentinel
sentinel:
  master_name: "mymaster"
  sentinel_addrs:
    - "127.0.0.1:26379"
    - "127.0.0.1:26380"
    - "127.0.0.1:26381"
```

### 集群模式

```yaml
mode: cluster
cluster:
  addrs:
    - "127.0.0.1:7000"
    - "127.0.0.1:7001"
    - "127.0.0.1:7002"
```

### 多主多从模式

```yaml
mode: multi-master
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

## 功能模块

### 服务器缓存

用于服务器级别的缓存，如配置信息、系统状态等。

```go
sc := rx.ServerCache()

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

// 获取系统状态
status, err := sc.GetSystemStatus(ctx)
if err != nil {
    log.Fatal(err)
}
```

### 用户数据缓存

用于用户数据的缓存，如用户信息、会话数据等。

```go
uc := rx.UserCache()

// 设置用户信息
userInfo := map[string]interface{}{
    "name":  "张三",
    "age":   30,
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

// 设置用户会话
session := map[string]interface{}{
    "user_id": "user123",
    "login_time": time.Now(),
}
err = uc.SetUserSession(ctx, "session456", session, 24*time.Hour)
if err != nil {
    log.Fatal(err)
}
```

### 锁机制

#### 单锁

```go
l := rx.NewLock("my_lock", 10*time.Second)

// 获取锁
acquired, err := l.Lock(ctx)
if err != nil {
    log.Fatal(err)
}
if !acquired {
    log.Println("获取锁失败")
    return
}
defer l.Unlock(ctx)

// 启动看门狗（自动续期）
watchdog := lock.NewWatchdog(l, 5*time.Second)
watchdog.Start(ctx)
defer watchdog.Stop()

// 执行业务逻辑
log.Println("执行业务逻辑...")
```

#### 红锁

```go
rl := rx.NewRedLock("my_redlock", 10*time.Second)

// 获取红锁
acquired, err := rl.Lock(ctx)
if err != nil {
    log.Fatal(err)
}
if !acquired {
    log.Println("获取红锁失败")
    return
}
defer rl.Unlock(ctx)

// 执行业务逻辑
log.Println("执行业务逻辑...")
```

### 高级操作

#### Lua 脚本

```go
lua := rx.Lua()

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
log.Println("Result:", result)
```

#### 管道操作

```go
pipeline := rx.Pipeline()

results, err := pipeline.Execute(ctx, func(pipe redis.Pipeliner) error {
    pipe.Set(ctx, "key1", "value1", 0)
    pipe.Set(ctx, "key2", "value2", 0)
    pipe.Get(ctx, "key1")
    return nil
})
if err != nil {
    log.Fatal(err)
}

value := results[2].(*redis.StringCmd).Val()
log.Println("Value:", value)
```

#### 事务操作

```go
txManager := rx.Transaction()

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

#### Watch 操作

```go
watchManager := rx.Watch()

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

## 配置说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| mode | 部署模式：single, sentinel, cluster, multi-master | - |
| debug | 是否开启调试模式 | false |
| password | Redis 密码 | "" |
| db | 数据库编号 | 0 |
| pool_size | 连接池大小 | 10 |
| min_idle_conns | 最小空闲连接数 | 5 |
| max_retries | 最大重试次数 | 3 |
| dial_timeout | 连接超时 | 5s |
| read_timeout | 读取超时 | 3s |
| write_timeout | 写入超时 | 3s |
| pool_timeout | 从连接池获取连接的超时 | 4s |

## 性能优化建议

1. **合理配置连接池**：根据业务量调整 `pool_size` 和 `min_idle_conns`
2. **使用批量操作**：使用 `MGET`/`MSET` 等批量命令减少网络往返
3. **使用管道**：对多个连续操作使用 Pipeline 减少网络往返
4. **使用 Lua 脚本**：将复杂操作封装为 Lua 脚本
5. **读写分离**：在主从模式下，读操作自动路由到从节点

## 错误处理

RedisX 提供了详细的错误信息，建议在关键操作中进行错误处理：

```go
result, err := sc.GetConfig(ctx, "app.timeout")
if err != nil {
    if errors.Is(err, redisx.ErrKeyNotFound) {
        log.Println("配置不存在")
    } else {
        log.Printf("获取配置失败: %v", err)
    }
    return
}
```

## 测试

```bash
# 运行测试
go test ./...

# 运行测试并查看覆盖率
go test -cover ./...

# 运行基准测试
go test -bench=. -benchmem ./...
```

## 文档

详细设计文档请参考 [DESIGN.md](./DESIGN.md)

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

Apache License 2.0
