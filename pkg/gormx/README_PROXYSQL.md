# GORM + ProxySQL 主从读写分离

## 架构说明

```
应用 (gormx)
    ↓
ProxySQL (6033)
    ↓
    ├─→ MySQL Master (写)
    ├─→ MySQL Slave1 (读)
    └─→ MySQL Slave2 (读)
```

## 优势

1. **应用无感知**：代码零改动，只需连接 ProxySQL
2. **自动路由**：
   - SELECT → 从库（负载均衡）
   - INSERT/UPDATE/DELETE → 主库
   - SELECT FOR UPDATE → 主库
   - 事务内所有操作 → 主库
3. **故障切换**：主库挂了自动切换
4. **连接池管理**：ProxySQL 统一管理连接

## 快速开始

### 1. 启动环境

```bash
cd pkg/gormx

# 方式 1：使用脚本（推荐）
chmod +x proxysql_setup.sh
./proxysql_setup.sh

# 方式 2：手动启动
docker-compose -f proxysql.example.yaml up -d
```

### 2. 应用配置

```go
package main

import (
    "github.com/tedwangl/go-util/pkg/gormx"
)

func main() {
    // 只需连接 ProxySQL，其他不变
    cfg := &gormx.Config{
        Driver:       "mysql",
        DSN:          "root:root123@tcp(127.0.0.1:6033)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
        MaxOpenConns: 100,
        MaxIdleConns: 10,
        LogLevel:     "info",
    }

    client, err := gormx.NewClient(cfg)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // 正常使用，ProxySQL 自动路由
    var users []User
    
    // SELECT → 从库
    client.DB.Find(&users)
    
    // INSERT → 主库
    client.DB.Create(&User{Name: "张三"})
    
    // UPDATE → 主库
    client.DB.Model(&User{}).Where("id = ?", 1).Update("name", "李四")
    
    // SELECT FOR UPDATE → 主库
    client.DB.Clauses(clause.Locking{Strength: "UPDATE"}).Find(&users)
    
    // 事务 → 全部走主库
    client.DB.Transaction(func(tx *gorm.DB) error {
        tx.Create(&User{Name: "王五"})
        tx.Find(&users)  // 事务内的 SELECT 也走主库
        return nil
    })
}
```

### 3. 验证路由

```bash
# 连接 ProxySQL 管理端口
mysql -h127.0.0.1 -P6032 -uadmin -padmin

# 查看连接池状态
SELECT hostgroup, srv_host, status, Queries, Bytes_data_sent, Bytes_data_recv 
FROM stats_mysql_connection_pool;

# 查看查询统计
SELECT rule_id, hits, destination_hostgroup 
FROM stats_mysql_query_rules 
ORDER BY hits DESC;

# 实时查看查询路由
SELECT * FROM stats_mysql_query_digest 
ORDER BY last_seen DESC LIMIT 10;
```

## ProxySQL 配置说明

### 主机组（Hostgroup）

- **hostgroup=0**：写组（主库）
- **hostgroup=1**：读组（从库）

### 路由规则优先级

1. `SELECT ... FOR UPDATE` → 主库
2. `SELECT` → 从库
3. `INSERT/UPDATE/DELETE` → 主库

### 负载均衡

从库使用 `weight` 权重进行负载均衡：
- weight=1：平均分配
- weight=2：2倍流量

## 生产环境配置

### 1. 连接池调优

```cnf
mysql_variables=
{
    max_connections=2048              # 最大连接数
    default_query_timeout=36000000    # 查询超时（10小时）
    connect_timeout_server=3000       # 连接超时（3秒）
    monitor_ping_interval=10000       # 心跳间隔（10秒）
}
```

### 2. 监控配置

```cnf
mysql_variables=
{
    monitor_username="monitor"
    monitor_password="monitor"
    monitor_read_only_interval=1500   # 检查只读状态间隔
    monitor_read_only_timeout=500     # 只读检查超时
}
```

### 3. 高可用配置

```cnf
mysql_replication_hostgroups =
(
    {
        writer_hostgroup=0
        reader_hostgroup=1
        check_type="read_only"        # 通过 read_only 变量判断主从
        comment="主从自动切换"
    }
)
```

## 常见问题

### 1. 从库延迟怎么办？

**方案 1**：强制走主库
```go
// 使用 Hint 强制主库
db.Exec("/* hostgroup=0 */ SELECT * FROM users")
```

**方案 2**：配置延迟阈值
```sql
-- ProxySQL 管理端
UPDATE mysql_servers 
SET max_replication_lag=10 
WHERE hostgroup=1;
```

### 2. 如何查看主从延迟？

```bash
mysql -h127.0.0.1 -P6032 -uadmin -padmin -e "
SELECT hostgroup, srv_host, status, 
       Queries, Latency_us/1000 as Latency_ms 
FROM stats_mysql_connection_pool;
"
```

### 3. 主库故障切换

ProxySQL 会自动检测主库状态：
- 主库挂了 → 自动标记为 OFFLINE
- 从库提升为主库 → 自动加入 hostgroup=0

### 4. 事务一定走主库吗？

是的。ProxySQL 检测到 `BEGIN/START TRANSACTION` 后，该连接的所有后续查询都路由到主库，直到 `COMMIT/ROLLBACK`。

## 监控指标

### 关键指标

```sql
-- 连接数
SELECT hostgroup, SUM(ConnUsed) as used, SUM(ConnFree) as free 
FROM stats_mysql_connection_pool 
GROUP BY hostgroup;

-- 查询 QPS
SELECT hostgroup, SUM(Queries) as total_queries 
FROM stats_mysql_connection_pool 
GROUP BY hostgroup;

-- 慢查询
SELECT digest_text, count_star, sum_time/1000000 as sum_time_sec 
FROM stats_mysql_query_digest 
WHERE sum_time > 1000000 
ORDER BY sum_time DESC;
```

## 清理环境

```bash
docker-compose -f proxysql.example.yaml down -v
```
