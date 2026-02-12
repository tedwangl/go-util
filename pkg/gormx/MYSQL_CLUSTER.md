# MySQL 集群方案与 gormx 配置

## 1. MySQL Group Replication (MGR)

### 架构
```
应用 (gormx)
    ↓
MySQL Router (VIP: mysql-cluster.local:6446)
    ↓
MGR 集群（多主或单主模式）
    ├─ mysql1 (Primary)
    ├─ mysql2 (Secondary)
    └─ mysql3 (Secondary)
```

### gormx 配置

```yaml
# 单主模式（推荐）
database:
  driver: mysql
  dsn: "root:password@tcp(mysql-cluster.local:6446)/db?charset=utf8mb4"
  replica_dsn: "root:password@tcp(mysql-cluster.local:6447)/db?charset=utf8mb4"
  # 6446: 读写端口（路由到 Primary）
  # 6447: 只读端口（路由到 Secondary）
```

```go
cfg := &gormx.Config{
    Driver:     "mysql",
    DSN:        "root:password@tcp(mysql-cluster.local:6446)/db?charset=utf8mb4",
    ReplicaDSN: "root:password@tcp(mysql-cluster.local:6447)/db?charset=utf8mb4",
}

client, _ := gormx.NewClient(cfg)

// SELECT → 6447 (Secondary 节点)
client.DB.Find(&users)

// INSERT/UPDATE/DELETE → 6446 (Primary 节点)
client.DB.Create(&User{Name: "张三"})
```

### 特点
- MySQL Router 自动故障切换
- 应用无需感知节点变化
- 支持读写分离

---

## 2. InnoDB Cluster (MGR + MySQL Router + MySQL Shell)

### 架构
```
应用 (gormx)
    ↓
MySQL Router (负载均衡)
    ↓
InnoDB Cluster
    ├─ mysql1 (RW)
    ├─ mysql2 (RO)
    └─ mysql3 (RO)
```

### gormx 配置

```yaml
database:
  driver: mysql
  dsn: "root:password@tcp(mysql-router.local:6446)/db?charset=utf8mb4"
  replica_dsn: "root:password@tcp(mysql-router.local:6447)/db?charset=utf8mb4"
```

### 特点
- 完整的高可用方案
- 自动故障检测和切换
- 配置方式与 MGR 相同

---

## 3. Galera Cluster (PXC/MariaDB Galera)

### 架构
```
应用 (gormx)
    ↓
HAProxy/ProxySQL (VIP)
    ↓
Galera 集群（多主同步复制）
    ├─ galera1 (Active)
    ├─ galera2 (Active)
    └─ galera3 (Active)
```

### gormx 配置

```yaml
# 方案 1：通过 HAProxy 负载均衡
database:
  driver: mysql
  dsn: "root:password@tcp(galera-lb.local:3306)/db?charset=utf8mb4"
  # 所有节点都可写，HAProxy 自动负载均衡

# 方案 2：读写分离（通过 HAProxy 不同端口）
database:
  driver: mysql
  dsn: "root:password@tcp(galera-lb.local:3306)/db?charset=utf8mb4"
  replica_dsn: "root:password@tcp(galera-lb.local:3307)/db?charset=utf8mb4"
  # 3306: 写端口（路由到主节点）
  # 3307: 读端口（负载均衡到所有节点）
```

### 特点
- 多主同步复制
- 任意节点可写
- 需要 HAProxy/ProxySQL 做负载均衡

---

## 4. 传统主从 + Orchestrator

### 架构
```
应用 (gormx)
    ↓
VIP (Keepalived/MHA)
    ↓
Orchestrator 管理
    ├─ mysql-master (Active)
    ├─ mysql-slave1 (Standby)
    └─ mysql-slave2 (Standby)
```

### gormx 配置

```yaml
database:
  driver: mysql
  dsn: "root:password@tcp(mysql-master.local:3306)/db?charset=utf8mb4"
  replica_dsn: "root:password@tcp(mysql-slave.local:3306)/db?charset=utf8mb4"
  # mysql-master.local: VIP 指向当前主库
  # mysql-slave.local: VIP 指向从库（可以是多个从库的负载均衡）
```

### 特点
- 经典方案，成熟稳定
- Orchestrator 自动故障切换
- 需要配置 VIP

---

## 5. MySQL NDB Cluster

### 架构
```
应用 (gormx)
    ↓
MySQL Server (SQL 节点)
    ↓
NDB 存储引擎
    ├─ Data Node 1
    ├─ Data Node 2
    └─ Data Node 3
```

### gormx 配置

```yaml
database:
  driver: mysql
  dsn: "root:password@tcp(ndb-sql1.local:3306)/db?charset=utf8mb4"
  # 可以配置多个 SQL 节点做负载均衡
```

### 特点
- 内存数据库，高性能
- 自动分片和冗余
- 适合高并发场景

---

## 6. 云厂商托管方案

### AWS RDS/Aurora

```yaml
database:
  driver: mysql
  dsn: "root:password@tcp(aurora-cluster.cluster-xxx.us-east-1.rds.amazonaws.com:3306)/db"
  replica_dsn: "root:password@tcp(aurora-cluster.cluster-ro-xxx.us-east-1.rds.amazonaws.com:3306)/db"
  # cluster endpoint: 写入端点（自动路由到主节点）
  # cluster-ro endpoint: 只读端点（负载均衡到只读副本）
```

### 阿里云 RDS/PolarDB

```yaml
database:
  driver: mysql
  dsn: "root:password@tcp(rm-xxx.mysql.rds.aliyuncs.com:3306)/db"
  replica_dsn: "root:password@tcp(rr-xxx.mysql.rds.aliyuncs.com:3306)/db"
  # rm-xxx: 主实例
  # rr-xxx: 只读实例
```

### 腾讯云 CDB

```yaml
database:
  driver: mysql
  dsn: "root:password@tcp(cdb-xxx.sql.tencentcdb.com:3306)/db"
  replica_dsn: "root:password@tcp(cdb-ro-xxx.sql.tencentcdb.com:3306)/db"
```

### 特点
- 完全托管，无需运维
- 自动备份和故障切换
- 按需扩展

---

## 配置对比

| 方案 | DSN 配置 | ReplicaDSN 配置 | 故障切换 | 负载均衡 |
|------|----------|-----------------|----------|----------|
| MGR + Router | Router:6446 | Router:6447 | 自动 | Router |
| InnoDB Cluster | Router:6446 | Router:6447 | 自动 | Router |
| Galera + HAProxy | HAProxy:3306 | HAProxy:3307 | 自动 | HAProxy |
| 主从 + Orchestrator | VIP:3306 | VIP:3306 | 自动 | 需配置 |
| NDB Cluster | SQL节点:3306 | - | 自动 | 应用层 |
| 云厂商托管 | 主实例 | 只读实例 | 自动 | 云厂商 |

---

## 推荐方案

### 自建环境
1. **中小规模**：传统主从 + Orchestrator
2. **高可用要求**：InnoDB Cluster (MGR + Router)
3. **多主需求**：Galera Cluster + HAProxy

### 云环境
1. **AWS**：Aurora MySQL
2. **阿里云**：PolarDB MySQL
3. **腾讯云**：CDB MySQL

---

## gormx 使用建议

### 1. 所有方案都只需配置入口地址

```go
cfg := &gormx.Config{
    Driver:     "mysql",
    DSN:        "主库入口地址（VIP/域名/负载均衡）",
    ReplicaDSN: "从库入口地址（可选）",
}
```

### 2. 不需要关心底层节点

- 主从切换由基础设施层处理
- gormx 只负责读写分离路由
- 应用代码无需修改

### 3. 动态配置更新

```go
// 使用 Viper 监听配置变化
loader := gormx.NewViperConfigLoader(v, "database")
cfg.ConfigLoader = loader

// VIP 地址变化时自动更新
client, _ := gormx.NewClient(cfg)
```

### 4. 健康检查

```go
// 定期检查连接状态
if err := client.Ping(); err != nil {
    log.Printf("database ping failed: %v", err)
}
```

---

## 总结

无论使用哪种 MySQL 集群方案，gormx 的配置方式都是一致的：

1. **DSN** - 主库入口地址
2. **ReplicaDSN** - 从库入口地址（可选）
3. 底层的故障切换、负载均衡由基础设施层处理
4. 应用层只需要知道入口地址即可
