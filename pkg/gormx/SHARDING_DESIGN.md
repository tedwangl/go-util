# 分片设计说明

## 分片路由原理

### 路由绑定时机

分片路由规则在 `NewClient()` 时通过 `dbresolver.Register()` 注册，**一旦注册就永久绑定**，运行时不会改变。

```go
// 初始化时注册
client, _ := gormx.NewClient(cfg)

// 之后所有查询都按照初始化时的规则路由
client.DB.Table("orders_001").Find(&orders) // → shard1
client.DB.Table("orders_002").Find(&orders) // → shard1
```

### 路由规则优先级

1. **精确匹配** > 通配符匹配
2. **先注册的分片优先**
3. **未匹配到分片** → 使用默认连接

### 示例

```go
cfg.WithSharding([]gormx.ShardConfig{
    {
        Name:   "shard1",
        Tables: []string{"orders_*"},  // 匹配 orders_ 开头的表
        DSN:    "shard1.db.local",
    },
    {
        Name:   "shard2",
        Tables: []string{"users_*"},   // 匹配 users_ 开头的表
        DSN:    "shard2.db.local",
    },
}, false)

// 路由结果（永久绑定）
orders_001 → shard1
orders_999 → shard1
users_001  → shard2
users_999  → shard2
products   → default（未匹配）
```

---

## 两种分片模式

### 模式 1：混合模式（推荐）

部分表分片，其他表在默认库。

```go
cfg := gormx.NewConfig("mysql", "default.db.local:3306")

cfg.WithSharding([]gormx.ShardConfig{
    {
        Name:   "orders_shard",
        Tables: []string{"orders_*", "order_items_*"},
        DSN:    "shard1.db.local:3306",
    },
}, false, "default-slave.db.local:3306") // pureSharding=false

// 路由
orders_001    → shard1
order_items_1 → shard1
users         → default
products      → default
```

**优点**：
- 灵活，可以逐步迁移表到分片
- 默认库可以存放配置表、字典表等

**缺点**：
- 需要维护默认库

---

### 模式 2：纯分片模式

所有表都在分片中，不使用默认库。

```go
cfg := gormx.NewConfig("mysql", "")  // DSN 留空，会自动使用第一个分片

cfg.WithSharding([]gormx.ShardConfig{
    {
        ID:     0,
        Name:   "shard0",
        Tables: []string{"users_*", "orders_*"},
        DSN:    "shard0.db.local:3306",
    },
    {
        ID:     1,
        Name:   "shard1",
        Tables: []string{"users_*", "orders_*"},
        DSN:    "shard1.db.local:3306",
    },
}, true) // pureSharding=true

// 应用层计算分片 ID
userID := 12345
shardID := userID % 2  // 0 或 1

tableName := fmt.Sprintf("users_%d", shardID)
client.DB.Table(tableName).Where("id = ?", userID).Find(&user)

// 路由
users_0  → shard0
orders_0 → shard0
users_1  → shard1
orders_1 → shard1
```

**优点**：
- 无默认库，架构更清晰
- 所有数据都分片，扩展性好

**缺点**：
- 所有表都必须匹配分片规则
- 需要应用层计算分片 ID

---

## 分片策略

### 策略 1：按时间分片（推荐用混合模式）

```go
cfg.WithSharding([]gormx.ShardConfig{
    {
        Name:   "orders_2024",
        Tables: []string{"orders_2024*"},
        DSN:    "shard-2024.db.local:3306",
    },
    {
        Name:   "orders_2025",
        Tables: []string{"orders_2025*"},
        DSN:    "shard-2025.db.local:3306",
    },
}, false)

// 应用层按时间路由
year := time.Now().Year()
month := time.Now().Month()
tableName := fmt.Sprintf("orders_%d%02d", year, month)
client.DB.Table(tableName).Create(&order)
```

### 策略 2：按 Hash 分片（推荐用纯分片模式）

```go
cfg.WithSharding([]gormx.ShardConfig{
    {ID: 0, Name: "shard0", Tables: []string{"*_0"}, DSN: "shard0.db.local"},
    {ID: 1, Name: "shard1", Tables: []string{"*_1"}, DSN: "shard1.db.local"},
    {ID: 2, Name: "shard2", Tables: []string{"*_2"}, DSN: "shard2.db.local"},
    {ID: 3, Name: "shard3", Tables: []string{"*_3"}, DSN: "shard3.db.local"},
}, true)

// 应用层计算 Hash
userID := 12345
shardID := userID % 4
tableName := fmt.Sprintf("users_%d", shardID)
client.DB.Table(tableName).Where("id = ?", userID).Find(&user)
```

### 策略 3：按租户分片（推荐用纯分片模式）

```go
cfg.WithSharding([]gormx.ShardConfig{
    {ID: 0, Name: "tenant_a", Tables: []string{"tenant_a_*"}, DSN: "tenant-a.db.local"},
    {ID: 1, Name: "tenant_b", Tables: []string{"tenant_b_*"}, DSN: "tenant-b.db.local"},
}, true)

// 应用层按租户路由
tenantID := "tenant_a"
tableName := fmt.Sprintf("%s_orders", tenantID)
client.DB.Table(tableName).Find(&orders)
```

---

## 注意事项

### 1. 跨分片查询

DBResolver 不支持跨分片查询，需要应用层处理：

```go
// ❌ 错误：无法跨分片 JOIN
client.DB.Table("orders_0").
    Joins("JOIN users_1 ON orders_0.user_id = users_1.id").
    Find(&results)

// ✅ 正确：分别查询，应用层聚合
var orders []Order
client.DB.Table("orders_0").Find(&orders)

var users []User
client.DB.Table("users_1").Find(&users)

// 应用层 JOIN
```

### 2. 分布式事务

DBResolver 不支持分布式事务，每个分片独立事务：

```go
// ❌ 错误：无法跨分片事务
client.DB.Transaction(func(tx *gorm.DB) error {
    tx.Table("orders_0").Create(&order)  // shard0
    tx.Table("users_1").Update(&user)    // shard1（不同分片）
    return nil
})

// ✅ 正确：使用 Saga 或 TCC 模式
```

### 3. 分片扩容

分片规则一旦注册就不能改变，扩容需要：

1. 停止应用
2. 修改配置，添加新分片
3. 重启应用
4. 数据迁移（应用层或工具）

```go
// 扩容前：2 个分片
shardID := userID % 2

// 扩容后：4 个分片
shardID := userID % 4

// 需要重新计算并迁移数据
```

---

## 最佳实践

1. **混合模式适合渐进式迁移**：先把大表分片，小表留在默认库
2. **纯分片模式适合新项目**：从一开始就规划好分片策略
3. **分片数量建议 2 的幂次**：方便扩容（2 → 4 → 8 → 16）
4. **避免跨分片查询**：设计时尽量让相关数据在同一分片
5. **监控每个分片的负载**：及时发现热点分片

---

## 总结

- **路由规则永久绑定**：初始化时确定，运行时不变
- **两种模式**：混合模式（灵活）vs 纯分片模式（清晰）
- **应用层负责**：计算分片 ID、跨分片查询、分布式事务
- **gormx 负责**：连接管理、读写分离、自动路由
