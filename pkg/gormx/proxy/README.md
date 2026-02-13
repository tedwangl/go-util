# GORM + go-gorm/sharding + ProxySQL

统一的单库、主从、分片数据库管理方案。

## 架构

```
应用 (gormx/proxy)
    ↓
go-gorm/sharding (分片路由)
    ↓
ProxySQL (主从路由)
    ↓
    ├─→ MySQL Master (写)
    ├─→ MySQL Slave1 (读)
    └─→ MySQL Slave2 (读)
```

## 特性

1. **单库模式**：直接连接 ProxySQL
2. **主从模式**：ProxySQL 自动读写分离
3. **分片模式**：go-gorm/sharding 分库分表 + ProxySQL 主从
4. **零侵入**：代码无需关心底层路由
5. **高性能**：代码层面分片，无网络开销

## 快速开始

### 1. 安装依赖

```bash
go get gorm.io/sharding
```

### 2. 单库模式

```go
cfg := &proxy.Config{
    Mode: "single",
    ProxySQL: &proxy.ProxySQLConfig{
        Addr:     "127.0.0.1:6033",
        Username: "root",
        Password: "root123",
        Database: "testdb",
    },
}

client, _ := proxy.NewClient(cfg)
defer client.Close()

// 正常使用
client.Create(&User{Name: "张三"})
```

### 3. 主从模式

```go
cfg := &proxy.Config{
    Mode: "master-slave",
    ProxySQL: &proxy.ProxySQLConfig{
        Addr: "127.0.0.1:6033", // ProxySQL 自动路由
    },
}

client, _ := proxy.NewClient(cfg)

// SELECT → 从库
client.Find(&users)

// INSERT/UPDATE/DELETE → 主库
client.Create(&user)

// 事务 → 全部主库
client.Transaction(func(tx *gorm.DB) error {
    tx.Create(&user)
    tx.Find(&users) // 事务内也走主库
    return nil
})
```

### 4. 分片模式

```go
cfg := &proxy.Config{
    Mode: "sharding",
    ProxySQL: &proxy.ProxySQLConfig{
        Addr: "127.0.0.1:6033",
    },
    Sharding: &proxy.ShardingConfig{
        Enable:     true,
        ShardCount: 4,
        Algorithm:  "mod",
        Tables: []*proxy.ShardingTable{
            {
                TableName:          "orders",
                ShardingKey:        "user_id",
                ShardCount:         4,
                GeneratePrimaryKey: true,
            },
        },
    },
}

client, _ := proxy.NewClient(cfg)

// 自动迁移（创建 orders_0, orders_1, orders_2, orders_3）
client.AutoMigrate(&Order{})

// 插入（自动路由到对应分片）
client.Create(&Order{UserID: 1001, Amount: 99.99})

// 查询（需要指定分片键）
client.Where("user_id = ?", 1001).Find(&orders)
```

### 5. 分片 + 主从

```go
// ProxySQL 处理主从，sharding 处理分片
cfg := &proxy.Config{
    Mode: "sharding",
    ProxySQL: &proxy.ProxySQLConfig{
        Addr: "127.0.0.1:6033", // 主从自动路由
    },
    Sharding: &proxy.ShardingConfig{
        Enable:     true,
        ShardCount: 8,
        Tables: []*proxy.ShardingTable{
            {
                TableName:   "orders",
                ShardingKey: "user_id",
                ShardCount:  8,
            },
        },
    },
}

client, _ := proxy.NewClient(cfg)

// 写 → 主库 + 分片路由
client.Create(&Order{UserID: 1001})

// 读 → 从库 + 分片路由
client.Where("user_id = ?", 1001).Find(&orders)
```

## 分片算法

### 1. Mod 取模（默认）

```go
ShardingTable{
    TableName:   "orders",
    ShardingKey: "user_id",
    Algorithm:   "mod",
    ShardCount:  4,
}

// user_id=1001 → orders_1 (1001 % 4 = 1)
// user_id=1002 → orders_2 (1002 % 4 = 2)
```

### 2. Hash 哈希

```go
ShardingTable{
    TableName:   "orders",
    ShardingKey: "user_id",
    Algorithm:   "hash",
    ShardCount:  4,
}

// 使用 CRC32 哈希
```

## 注意事项

### 1. 分片键必须在查询条件中

```go
// ✅ 正确
client.Where("user_id = ?", 1001).Find(&orders)

// ❌ 错误（无法确定分片）
client.Find(&orders)
```

### 2. 跨分片查询

```go
// 需要遍历所有分片
for i := 0; i < shardCount; i++ {
    client.Table(fmt.Sprintf("orders_%d", i)).Find(&orders)
}
```

### 3. 事务限制

```go
// 事务只能在单个分片内
client.Transaction(func(tx *gorm.DB) error {
    // 同一个 user_id，在同一个分片
    tx.Create(&Order{UserID: 1001, Amount: 99.99})
    tx.Create(&OrderItem{OrderID: 1, UserID: 1001})
    return nil
})
```

### 4. 主键生成

```go
ShardingTable{
    GeneratePrimaryKey: true, // 自动生成分布式 ID
}
```

## 配置示例

### 开发环境

```yaml
mode: single
proxysql:
  addr: "127.0.0.1:6033"
  username: "root"
  password: "root123"
  database: "testdb"
max_open_conns: 10
log_level: "info"
```

### 生产环境（主从）

```yaml
mode: master-slave
proxysql:
  addr: "proxysql.prod:6033"
  username: "app_user"
  password: "xxx"
  database: "prod_db"
max_open_conns: 200
max_idle_conns: 50
log_level: "warn"
slow_threshold: 200ms
```

### 生产环境（分片）

```yaml
mode: sharding
proxysql:
  addr: "proxysql.prod:6033"
  username: "app_user"
  password: "xxx"
  database: "prod_db"
sharding:
  enable: true
  shard_count: 16
  algorithm: "mod"
  tables:
    - table_name: "orders"
      sharding_key: "user_id"
      shard_count: 16
      generate_primary_key: true
    - table_name: "order_items"
      sharding_key: "order_id"
      shard_count: 16
      generate_primary_key: true
max_open_conns: 500
log_level: "error"
```

## 性能对比

| 方案 | 读写分离 | 分库分表 | 网络开销 | 复杂度 |
|------|---------|---------|---------|--------|
| 单 GORM | ❌ | ❌ | 0 | 低 |
| GORM + DBResolver | ✅ | ❌ | 0 | 中 |
| GORM + ProxySQL | ✅ | ❌ | ~1ms | 低 |
| GORM + sharding + ProxySQL | ✅ | ✅ | ~1ms | 中 |

## 最佳实践

1. **开发环境**：单库模式
2. **小规模生产**：主从模式（< 1000 QPS）
3. **中等规模**：分片模式（1000-10000 QPS）
4. **大规模**：分片 + 主从（> 10000 QPS）

## 故障排查

### 1. 连接失败

```bash
# 测试 ProxySQL 连接
mysql -h127.0.0.1 -P6033 -uroot -proot123 testdb
```

### 2. 分片路由错误

```go
// 开启 SQL 日志
cfg.LogLevel = "info"

// 查看实际执行的表名
```

### 3. 性能问题

```bash
# 查看 ProxySQL 连接池
mysql -h127.0.0.1 -P6032 -uadmin -padmin -e "
SELECT * FROM stats_mysql_connection_pool;
"
```
