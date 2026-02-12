# 连接池数量分析

## 场景 1：单库

```go
cfg := gormx.NewConfig("mysql", "db.local:3306")
client, _ := gormx.NewClient(cfg)
```

**连接池数量：1 个**
- 主库连接池 → `db.local:3306`

---

## 场景 2：主从

```go
cfg := gormx.NewConfig("mysql", "master.db.local:3306")
cfg.WithReplica("slave.db.local:3306")
client, _ := gormx.NewClient(cfg)
```

**连接池数量：2 个**
- 主库连接池 → `master.db.local:3306`
- 从库连接池 → `slave.db.local:3306`

---

## 场景 3：混合分片（有默认库）

```go
cfg := gormx.NewConfig("mysql", "default.db.local:3306")
cfg.WithSharding([]gormx.ShardConfig{
    {
        Name:       "shard1",
        Tables:     []string{"orders_*"},
        DSN:        "shard1.db.local:3306",
        ReplicaDSN: "shard1-slave.db.local:3306",
    },
    {
        Name:       "shard2",
        Tables:     []string{"users_*"},
        DSN:        "shard2.db.local:3306",
        ReplicaDSN: "shard2-slave.db.local:3306",
    },
}, false, "default-slave.db.local:3306")
client, _ := gormx.NewClient(cfg)
```

**连接池数量：6 个**
- 默认主库 → `default.db.local:3306`
- 默认从库 → `default-slave.db.local:3306`
- shard1 主库 → `shard1.db.local:3306`
- shard1 从库 → `shard1-slave.db.local:3306`
- shard2 主库 → `shard2.db.local:3306`
- shard2 从库 → `shard2-slave.db.local:3306`

**计算公式：**
```
总数 = 1(默认主) + 1(默认从) + 分片数 * 2
     = 1 + 1 + 2 * 2
     = 6
```

---

## 场景 4：纯分片（无默认库）- 关键优化

```go
cfg := gormx.NewConfig("mysql", "")  // DSN 留空
cfg.WithSharding([]gormx.ShardConfig{
    {
        ID:         0,
        Name:       "shard0",
        Tables:     []string{"*_0"},
        DSN:        "shard0.db.local:3306",
        ReplicaDSN: "shard0-slave.db.local:3306",
    },
    {
        ID:         1,
        Name:       "shard1",
        Tables:     []string{"*_1"},
        DSN:        "shard1.db.local:3306",
        ReplicaDSN: "shard1-slave.db.local:3306",
    },
    {
        ID:         2,
        Name:       "shard2",
        Tables:     []string{"*_2"},
        DSN:        "shard2.db.local:3306",
        ReplicaDSN: "shard2-slave.db.local:3306",
    },
}, true)  // pureSharding=true
client, _ := gormx.NewClient(cfg)
```

### 优化前（错误）：7 个连接池 ❌

```
主连接（shard0）→ shard0.db.local:3306  ← 重复！
shard0 主库     → shard0.db.local:3306  ← 重复！
shard0 从库     → shard0-slave.db.local:3306
shard1 主库     → shard1.db.local:3306
shard1 从库     → shard1-slave.db.local:3306
shard2 主库     → shard2.db.local:3306
shard2 从库     → shard2-slave.db.local:3306
```

### 优化后（正确）：6 个连接池 ✅

```
主连接（shard0）→ shard0.db.local:3306  ← 复用
shard0 从库     → shard0-slave.db.local:3306
shard1 主库     → shard1.db.local:3306
shard1 从库     → shard1-slave.db.local:3306
shard2 主库     → shard2.db.local:3306
shard2 从库     → shard2-slave.db.local:3306
```

**关键代码：**
```go
// resolver.go - setupSharding()
for i, shard := range sharding.Shards {
    // 纯分片模式：跳过第一个分片（已作为主连接）
    if sharding.PureSharding && i == 0 {
        // 只配置从库
        if shard.ReplicaDSN != "" {
            c.setupReplica(&ReplicaConfig{ReplicaDSN: shard.ReplicaDSN})
        }
        continue  // 跳过主库注册
    }
    // ... 其他分片正常注册
}
```

**计算公式：**
```
总数 = 1(第一个分片主库，复用) + 1(第一个分片从库) + (分片数-1) * 2
     = 1 + 1 + (3-1) * 2
     = 6
```

---

## 连接池配置

每个连接池独立配置：

```go
cfg := gormx.NewConfig("mysql", "db.local:3306")
cfg.MaxOpenConns = 100  // 每个连接池最多 100 个连接
cfg.MaxIdleConns = 10   // 每个连接池最多 10 个空闲连接
```

### 场景 4（纯分片 3 个分片）的实际连接数

```
每个连接池：最多 100 个连接
总连接池数：6 个
理论最大连接数：6 * 100 = 600 个

实际连接数取决于：
- 应用并发量
- 连接池空闲策略
- 数据库负载
```

---

## 优化建议

### 1. 根据场景调整连接池大小

```go
// 高并发场景
cfg.MaxOpenConns = 200
cfg.MaxIdleConns = 50

// 低并发场景
cfg.MaxOpenConns = 20
cfg.MaxIdleConns = 5
```

### 2. 监控连接池状态

```go
stats := client.Stats()
fmt.Printf("OpenConnections: %d\n", stats.OpenConnections)
fmt.Printf("InUse: %d\n", stats.InUse)
fmt.Printf("Idle: %d\n", stats.Idle)
```

### 3. 避免连接泄漏

```go
// ✅ 正确：使用 defer
client, _ := gormx.NewClient(cfg)
defer client.Close()

// ❌ 错误：忘记关闭
client, _ := gormx.NewClient(cfg)
// ... 程序退出时连接未释放
```

---

## 总结

| 场景 | 连接池数量 | 计算公式 |
|------|-----------|---------|
| 单库 | 1 | 1 |
| 主从 | 2 | 1 + 1 |
| 混合分片 | 2 + 分片数*2 | 1(默认主) + 1(默认从) + N*2 |
| 纯分片 | 1 + 分片数*2 - 1 | 1(第一个分片主) + 1(第一个分片从) + (N-1)*2 |

**关键优化**：纯分片模式下，第一个分片的主库连接被复用，避免重复创建连接池。
