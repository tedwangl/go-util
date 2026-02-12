# gormx - GORM 轻量封装

## 设计理念

**不过度封装**，只封装连接配置和读写分离，主从切换由基础设施层（Orchestrator/VIP）处理。

## 核心思想

1. **应用层只配置入口地址**（VIP/域名）
2. **主从切换由 Orchestrator 等工具管理**
3. **分布式数据库自带负载均衡**（如 TiDB）
4. **gormx 只负责读写分离路由**

## 架构示例

### 传统主从 + Orchestrator

```
应用 (gormx)
    ↓
DSN: master.db.local (VIP)
ReplicaDSN: slave.db.local (VIP)
    ↓
Orchestrator 自动切换
    ↓
实际主库: db1/db2/db3
实际从库: db4/db5/db6
```

### MySQL Group Replication (MGR)

```
应用 (gormx)
    ↓
DSN: mysql-router.local:6446 (写端口)
ReplicaDSN: mysql-router.local:6447 (读端口)
    ↓
MySQL Router 自动路由
    ↓
MGR 集群: mysql1(Primary) / mysql2(Secondary) / mysql3(Secondary)
```

### Galera Cluster

```
应用 (gormx)
    ↓
DSN: galera-lb.local:3306 (HAProxy)
    ↓
HAProxy 负载均衡
    ↓
Galera 集群: galera1 / galera2 / galera3 (多主同步)
```

### 分布式数据库（TiDB）

```
应用 (gormx)
    ↓
DSN: tidb.cluster.local:4000
    ↓
TiDB Server 自动路由
    ↓
TiKV 存储节点（应用无感知）
```

## 快速开始

### 1. 单库（无读写分离）

```go
cfg := &gormx.Config{
    Driver: "mysql",
    DSN:    "root:password@tcp(master.db.local:3306)/db?charset=utf8mb4",
    MaxOpenConns: 100,
    MaxIdleConns: 10,
}

client, _ := gormx.NewClient(cfg)
defer client.Close()

// 所有操作走主库
client.DB.Find(&users)
client.DB.Create(&User{Name: "张三"})
```

### 2. 主从读写分离

```yaml
# config.yaml
database:
  driver: mysql
  dsn: "root:password@tcp(master.db.local:3306)/db?charset=utf8mb4"
  replica_dsn: "root:password@tcp(slave.db.local:3306)/db?charset=utf8mb4"
  max_open_conns: 100
  max_idle_conns: 10
```

```go
cfg := &gormx.Config{
    Driver:     "mysql",
    DSN:        "root:password@tcp(master.db.local:3306)/db?charset=utf8mb4",
    ReplicaDSN: "root:password@tcp(slave.db.local:3306)/db?charset=utf8mb4",
}

client, _ := gormx.NewClient(cfg)

// SELECT → slave.db.local
var users []User
client.DB.Find(&users)

// INSERT/UPDATE/DELETE → master.db.local
client.DB.Create(&User{Name: "张三"})
```

### 3. 分库分表

```yaml
# config.yaml
database:
  driver: mysql
  dsn: "root:password@tcp(default.db.local:3306)/db?charset=utf8mb4"
  replica_dsn: "root:password@tcp(default-slave.db.local:3306)/db?charset=utf8mb4"
  shards:
    - name: "shard1"
      tables: ["orders_*", "order_items_*"]
      dsn: "root:password@tcp(shard1.db.local:3306)/shard1?charset=utf8mb4"
      replica_dsn: "root:password@tcp(shard1-slave.db.local:3306)/shard1?charset=utf8mb4"
    - name: "shard2"
      tables: ["users_*", "user_profiles_*"]
      dsn: "root:password@tcp(shard2.db.local:3306)/shard2?charset=utf8mb4"
      replica_dsn: "root:password@tcp(shard2-slave.db.local:3306)/shard2?charset=utf8mb4"
```

```go
client, _ := gormx.NewClient(cfg)

// 自动路由到 shard1
var orders []Order
client.DB.Table("orders_202401").Find(&orders) // → shard1-slave.db.local

// 自动路由到 shard2
var users []User
client.DB.Table("users_vip").Find(&users) // → shard2-slave.db.local

// 默认表走默认库
var products []Product
client.DB.Find(&products) // → default-slave.db.local
```

## 路由规则

DBResolver 自动处理：

- `SELECT` → ReplicaDSN（从库）
- `INSERT/UPDATE/DELETE` → DSN（主库）
- `事务内所有操作` → DSN（主库）
- `SELECT FOR UPDATE` → DSN（主库）

## 配置说明

| 参数 | 说明 | 示例 |
|------|------|------|
| DSN | 主库地址（VIP/域名） | master.db.local:3306 |
| ReplicaDSN | 从库地址（VIP/域名） | slave.db.local:3306 |
| Shards[].DSN | 分片主库地址 | shard1.db.local:3306 |
| Shards[].ReplicaDSN | 分片从库地址 | shard1-slave.db.local:3306 |

## 与 Orchestrator 配合

### Orchestrator 配置示例

```json
{
  "clusters": {
    "main": {
      "master": "db1.internal:3306",
      "vip": "master.db.local"
    }
  }
}
```

### 故障切换流程

1. db1 主库故障
2. Orchestrator 检测到故障
3. Orchestrator 将 VIP 切换到 db2
4. 应用无感知，继续使用 master.db.local

## 分布式数据库支持

### TiDB

```go
cfg := &gormx.Config{
    Driver: "mysql",
    DSN:    "root:password@tcp(tidb.cluster.local:4000)/db?charset=utf8mb4",
    // 不需要配置 ReplicaDSN，TiDB 自动负载均衡
}
```

### CockroachDB

```go
cfg := &gormx.Config{
    Driver: "postgres",
    DSN:    "postgresql://root@cockroach.cluster.local:26257/db?sslmode=disable",
    // 不需要配置 ReplicaDSN，CockroachDB 自动负载均衡
}
```

## 最佳实践

1. **使用 VIP/域名**，不要硬编码 IP
2. **主从切换由基础设施层处理**（Orchestrator/Keepalived）
3. **分布式数据库不需要配置从库**
4. **定期健康检查**，监控连接状态

## 示例

查看 `example_test.go` 获取更多示例。


## 测试结果

所有 4 个场景的集成测试已通过：

```bash
cd pkg/gormx
make test-integration
```

### 测试场景

✅ **场景 1：单库** - 1 个连接池  
✅ **场景 2：主从** - 2 个连接池（主+从）  
✅ **场景 3：混合分片** - 6 个连接池（默认主+默认从+2分片*2）  
✅ **场景 4：纯分片** - 4 个连接池（第一个分片主库被复用+第一个分片从+第二个分片主从）

### 关键修复

**问题**：DBResolver 抛出 "registered" 错误

**原因**：多次调用 `db.Use(dbresolver.Register())` 会导致插件重复注册

**解决方案**：使用链式调用 `Register()`，只调用一次 `db.Use()`

```go
// ❌ 错误：多次 Use
db.Use(dbresolver.Register(cfg1, "table1"))
db.Use(dbresolver.Register(cfg2, "table2"))  // 报错：registered

// ✅ 正确：链式 Register，只 Use 一次
plugin := dbresolver.Register(cfg1, "table1").
    Register(cfg2, "table2")
db.Use(plugin)
```

详见 `pkg/gormx/resolver.go` 中的 `setupSharding()` 函数。

### 测试环境

使用 Docker Compose 启动 13 个 MySQL 容器模拟真实环境：

```bash
cd pkg/gormx
docker-compose up -d
```

容器列表：
- 场景 1：1 个单库
- 场景 2：2 个（主+从）
- 场景 3：6 个（默认主从+2个分片各主从）
- 场景 4：4 个（2个分片各主从）
