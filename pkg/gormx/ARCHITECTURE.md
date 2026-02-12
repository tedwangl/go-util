# gormx 架构原理

## 1. 连接管理原理

### 单库场景

```go
cfg := &gormx.Config{
    Driver: "mysql",
    DSN:    "root:password@tcp(master.db.local:3306)/db",
}
```

**连接数：1 个连接池**
- 主库连接池 → `master.db.local:3306`
- 所有操作（读写）都走这个连接池

### 主从场景

```go
cfg := &gormx.Config{
    Driver:     "mysql",
    DSN:        "root:password@tcp(master.db.local:3306)/db",
    ReplicaDSN: "root:password@tcp(slave.db.local:3306)/db",
}
```

**连接数：2 个连接池**
- 主库连接池 → `master.db.local:3306`（写操作）
- 从库连接池 → `slave.db.local:3306`（读操作）

**代码生效位置：**

```go
// client.go - NewClient()
db, err := gorm.Open(dialector, gormConfig) // 创建主库连接

// resolver.go - setupDBResolver()
if cfg.ReplicaDSN != "" {
    resolverCfg := dbresolver.Config{
        Replicas: []gorm.Dialector{replica}, // 创建从库连接
    }
    c.DB.Use(dbresolver.Register(resolverCfg)) // 注册到 DBResolver
}
```

### 分片场景

```go
cfg := &gormx.Config{
    Driver:     "mysql",
    DSN:        "root:password@tcp(default.db.local:3306)/db",
    ReplicaDSN: "root:password@tcp(default-slave.db.local:3306)/db",
    Shards: []ShardConfig{
        {
            Name:       "shard1",
            Tables:     []string{"orders_*"},
            DSN:        "root:password@tcp(shard1.db.local:3306)/shard1",
            ReplicaDSN: "root:password@tcp(shard1-slave.db.local:3306)/shard1",
        },
        {
            Name:       "shard2",
            Tables:     []string{"users_*"},
            DSN:        "root:password@tcp(shard2.db.local:3306)/shard2",
            ReplicaDSN: "root:password@tcp(shard2-slave.db.local:3306)/shard2",
        },
    },
}
```

**连接数：6 个连接池**
- 默认主库连接池 → `default.db.local:3306`
- 默认从库连接池 → `default-slave.db.local:3306`
- shard1 主库连接池 → `shard1.db.local:3306`
- shard1 从库连接池 → `shard1-slave.db.local:3306`
- shard2 主库连接池 → `shard2.db.local:3306`
- shard2 从库连接池 → `shard2-slave.db.local:3306`

**计算公式：**
```
总连接池数 = 1(默认主) + 1(默认从) + 分片数 * 2(每个分片的主从)
         = 1 + 1 + 2 * 2
         = 6
```

---

## 2. DNS/VIP 解析原理

### DNS 域名方式

```
应用配置: master.db.local
    ↓
DNS 解析
    ↓
实际 IP: 192.168.1.10 (当前主库)
```

**Orchestrator 切换流程：**
1. db1 (192.168.1.10) 故障
2. Orchestrator 检测到故障
3. Orchestrator 更新 DNS：`master.db.local` → 192.168.1.11 (db2)
4. 应用下次连接时自动解析到新 IP

**DNS TTL 影响：**
- TTL=60s：最多 60 秒后生效
- 建议 TTL 设置较短（10-30s）

### VIP (Virtual IP) 方式

```
应用配置: 192.168.1.100 (VIP)
    ↓
Keepalived/MHA
    ↓
实际主库: 192.168.1.10
```

**VIP 切换流程：**
1. db1 (192.168.1.10) 故障
2. Keepalived 检测到故障
3. VIP (192.168.1.100) 漂移到 db2 (192.168.1.11)
4. 应用无感知，继续使用 VIP

**优势：**
- 切换速度快（秒级）
- 无 DNS 缓存问题

---

## 3. DBResolver 路由原理

### 代码执行流程

```go
// 1. 查询操作
client.DB.Find(&users)
    ↓
DBResolver 拦截
    ↓
判断：SELECT 语句
    ↓
路由到：Replicas 连接池（从库）
    ↓
执行：slave.db.local:3306

// 2. 写操作
client.DB.Create(&user)
    ↓
DBResolver 拦截
    ↓
判断：INSERT 语句
    ↓
路由到：Sources 连接池（主库）
    ↓
执行：master.db.local:3306

// 3. 事务操作
client.DB.Transaction(func(tx *gorm.DB) error {
    tx.Create(&user)  // 主库
    tx.Find(&users)   // 主库（事务内强制主库）
    return nil
})
```

### DBResolver 核心代码

```go
// resolver.go
func (c *Client) setupDBResolver(cfg *Config) error {
    // 配置全局主从
    if cfg.ReplicaDSN != "" {
        resolverCfg := dbresolver.Config{
            Policy: dbresolver.RandomPolicy{}, // 负载均衡策略
            Replicas: []gorm.Dialector{replica}, // 从库列表
        }
        c.DB.Use(dbresolver.Register(resolverCfg))
    }

    // 配置分片
    for _, shard := range cfg.Shards {
        shardCfg := dbresolver.Config{
            Sources:  []gorm.Dialector{source},  // 分片主库
            Replicas: []gorm.Dialector{replica}, // 分片从库
        }
        // 按表名路由
        c.DB.Use(dbresolver.Register(shardCfg, shard.Tables...))
    }
}
```

### 路由决策树

```
SQL 请求
    ↓
是否在事务中？
    ├─ 是 → 主库
    └─ 否 → 继续判断
        ↓
    是否匹配分片表？
        ├─ 是 → 使用分片连接池
        └─ 否 → 使用默认连接池
            ↓
        是否为写操作？
            ├─ 是 → 主库连接池
            └─ 否 → 从库连接池
```

---

## 4. 连接池详解

### 连接池配置

```go
cfg := &gormx.Config{
    MaxOpenConns: 100, // 最大连接数
    MaxIdleConns: 10,  // 最大空闲连接数
    MaxLifetime:  1h,  // 连接最大生命周期
    MaxIdleTime:  10m, // 连接最大空闲时间
}
```

### 每个连接池独立管理

```go
// client.go
sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)  // 主库连接池
sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)
sqlDB.SetConnMaxIdleTime(cfg.MaxIdleTime)

// DBResolver 会为每个 Replica/Source 创建独立连接池
// 每个连接池都有自己的 MaxOpenConns/MaxIdleConns
```

### 连接池数量计算

**示例配置：**
```yaml
database:
  max_open_conns: 100
  max_idle_conns: 10
  dsn: "master.db.local"
  replica_dsn: "slave.db.local"
  shards:
    - name: "shard1"
      dsn: "shard1.db.local"
      replica_dsn: "shard1-slave.db.local"
```

**实际连接数：**
- 默认主库：最多 100 个连接
- 默认从库：最多 100 个连接
- shard1 主库：最多 100 个连接
- shard1 从库：最多 100 个连接
- **总计：最多 400 个连接**

**注意：** 每个连接池独立计数，不共享！

---

## 5. 故障切换原理

### Orchestrator 管理主从切换

```
初始状态:
master.db.local → 192.168.1.10 (db1)
slave.db.local  → 192.168.1.11 (db2)

故障发生:
db1 宕机

Orchestrator 操作:
1. 检测到 db1 故障
2. 提升 db2 为新主库
3. 更新 DNS:
   master.db.local → 192.168.1.11 (db2)
   slave.db.local  → 192.168.1.12 (db3)

应用层:
- 现有连接：可能失败（连接到旧主库）
- 新连接：自动连接到新主库
- 重试机制：失败后重新连接
```

### MySQL Router 管理 MGR 集群

```
应用配置:
DSN: mysql-router.local:6446 (写端口)
ReplicaDSN: mysql-router.local:6447 (读端口)

MySQL Router 内部:
6446 → Primary 节点 (自动检测)
6447 → Secondary 节点 (负载均衡)

故障切换:
1. Primary 节点故障
2. MGR 自动选举新 Primary
3. MySQL Router 检测到变化
4. 自动将 6446 路由到新 Primary
5. 应用无感知
```

---

## 6. 分片路由原理

### 表名匹配

```go
// 配置
Shards: []ShardConfig{
    {
        Name:   "shard1",
        Tables: []string{"orders_*", "order_items_*"},
        DSN:    "shard1.db.local",
    },
}

// 查询
client.DB.Table("orders_202401").Find(&orders)
    ↓
DBResolver 匹配表名
    ↓
"orders_202401" 匹配 "orders_*"
    ↓
路由到 shard1.db.local
```

### 通配符规则

- `orders_*` - 匹配 `orders_` 开头的所有表
- `*_log` - 匹配 `_log` 结尾的所有表
- `user` - 精确匹配 `user` 表

### 路由优先级

```
1. 精确匹配 > 通配符匹配
2. 先注册的分片优先
3. 未匹配到分片 → 使用默认连接
```

---

## 7. 实际案例

### 案例 1：电商订单分片

```go
cfg := &gormx.Config{
    Driver: "mysql",
    DSN:    "root:password@tcp(default.db.local:3306)/db",
    ReplicaDSN: "root:password@tcp(default-slave.db.local:3306)/db",
    Shards: []ShardConfig{
        {
            Name:       "orders_2024",
            Tables:     []string{"orders_2024*"},
            DSN:        "root:password@tcp(shard-2024.db.local:3306)/orders_2024",
            ReplicaDSN: "root:password@tcp(shard-2024-slave.db.local:3306)/orders_2024",
        },
        {
            Name:       "orders_2025",
            Tables:     []string{"orders_2025*"},
            DSN:        "root:password@tcp(shard-2025.db.local:3306)/orders_2025",
            ReplicaDSN: "root:password@tcp(shard-2025-slave.db.local:3306)/orders_2025",
        },
    },
}

// 查询 2024 年订单 → shard-2024-slave.db.local
client.DB.Table("orders_202401").Find(&orders)

// 查询 2025 年订单 → shard-2025-slave.db.local
client.DB.Table("orders_202501").Find(&orders)

// 查询用户表 → default-slave.db.local
client.DB.Find(&users)
```

**连接池数量：**
- 默认库：2 个（主 + 从）
- 2024 分片：2 个（主 + 从）
- 2025 分片：2 个（主 + 从）
- **总计：6 个连接池**

### 案例 2：多租户分片

```go
// 根据租户 ID 计算分片
tenantID := 12345
shardID := tenantID % 10 // 10 个分片

tableName := fmt.Sprintf("tenant_%d_orders", shardID)
client.DB.Table(tableName).Find(&orders)
```

---

## 8. 性能优化

### 连接池调优

```go
// 高并发场景
cfg := &gormx.Config{
    MaxOpenConns: 200,  // 增加最大连接数
    MaxIdleConns: 50,   // 增加空闲连接数
    MaxLifetime:  30m,  // 缩短连接生命周期
    MaxIdleTime:  5m,   // 缩短空闲时间
}

// 低并发场景
cfg := &gormx.Config{
    MaxOpenConns: 20,   // 减少连接数
    MaxIdleConns: 5,    // 减少空闲连接
    MaxLifetime:  2h,   // 延长生命周期
    MaxIdleTime:  30m,  // 延长空闲时间
}
```

### 监控指标

```go
// 获取连接池状态
stats := client.Stats()
fmt.Printf("OpenConnections: %d\n", stats.OpenConnections)
fmt.Printf("InUse: %d\n", stats.InUse)
fmt.Printf("Idle: %d\n", stats.Idle)
fmt.Printf("WaitCount: %d\n", stats.WaitCount)
fmt.Printf("WaitDuration: %v\n", stats.WaitDuration)
```

---

## 总结

1. **连接管理**：每个 DSN 对应一个独立连接池
2. **DNS/VIP**：由基础设施层管理，应用层透明
3. **路由决策**：DBResolver 根据 SQL 类型和表名自动路由
4. **故障切换**：基础设施层处理，应用层重连即可
5. **连接池数量**：`1(默认主) + 1(默认从) + 分片数 * 2`
