# gormx 测试指南

本文档说明如何运行 gormx 的集成测试。

## 前置条件

1. 安装 Docker 和 Docker Compose
2. 确保端口 3306-3318 未被占用

## 快速开始

```bash
# 1. 启动测试数据库
cd pkg/gormx
docker-compose up -d

# 2. 等待数据库启动（约 10 秒）
sleep 10

# 3. 运行集成测试
make test-integration

# 4. 清理环境
docker-compose down -v
```

## 测试场景

### 场景 1：单库

**配置**：
```go
cfg := gormx.NewConfig("mysql", "root:root123@tcp(localhost:3306)/testdb")
```

**预期结果**：
- ✅ 连接池数量：1
- ✅ 写入和查询都走同一个数据库

### 场景 2：主从

**配置**：
```go
cfg := gormx.NewConfig("mysql", "root:root123@tcp(localhost:3307)/testdb")
cfg.WithReplica("root:root123@tcp(localhost:3308)/testdb")
```

**预期结果**：
- ✅ 连接池数量：2（主+从）
- ✅ 写入走主库（3307）
- ✅ 查询走从库（3308）
- ✅ 事务内的所有操作走主库

### 场景 3：混合分片

**配置**：
```go
cfg := gormx.NewConfig("mysql", "root:root123@tcp(localhost:3309)/testdb")
cfg.WithSharding([]gormx.ShardConfig{
    {
        Name:       "shard1",
        Tables:     []string{"test_orders_1"},
        DSN:        "root:root123@tcp(localhost:3311)/shard1",
        ReplicaDSN: "root:root123@tcp(localhost:3312)/shard1",
    },
    {
        Name:       "shard2",
        Tables:     []string{"test_orders_2"},
        DSN:        "root:root123@tcp(localhost:3313)/shard2",
        ReplicaDSN: "root:root123@tcp(localhost:3314)/shard2",
    },
}, false, "root:root123@tcp(localhost:3310)/testdb")
```

**预期结果**：
- ✅ 连接池数量：6（默认主+默认从+2分片*2）
- ✅ `test_users` 表走默认库（3309主/3310从）
- ✅ `test_orders_1` 表走 shard1（3311主/3312从）
- ✅ `test_orders_2` 表走 shard2（3313主/3314从）

### 场景 4：纯分片

**配置**：
```go
cfg := gormx.NewConfig("mysql", "")  // DSN 留空
cfg.WithSharding([]gormx.ShardConfig{
    {
        ID:         0,
        Name:       "shard0",
        Tables:     []string{"test_users_0", "test_orders_0"},
        DSN:        "root:root123@tcp(localhost:3315)/shard0",
        ReplicaDSN: "root:root123@tcp(localhost:3316)/shard0",
    },
    {
        ID:         1,
        Name:       "shard1",
        Tables:     []string{"test_users_1", "test_orders_1"},
        DSN:        "root:root123@tcp(localhost:3317)/shard1",
        ReplicaDSN: "root:root123@tcp(localhost:3318)/shard1",
    },
}, true)  // pureSharding=true
```

**预期结果**：
- ✅ 连接池数量：4（第一个分片主库被复用+第一个分片从+第二个分片主从）
- ✅ 所有表都在分片中，按表名路由
- ✅ 第一个分片的主库作为默认连接，避免重复创建连接池

## 关键修复：DBResolver "registered" 错误

### 问题

在场景 3 和场景 4 中，DBResolver 抛出 "registered" 错误：

```
failed to register shard shard2: registered
```

### 原因

多次调用 `db.Use(dbresolver.Register())` 会导致插件重复注册。

### 解决方案

使用链式调用 `Register()`，只调用一次 `db.Use()`：

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

## 故障排查

### 端口冲突

如果端口被占用，修改 `docker-compose.yml` 中的端口映射：

```yaml
ports:
  - "13306:3306"  # 使用 13306 代替 3306
```

### 数据库未启动

检查容器状态：

```bash
docker-compose ps
```

查看日志：

```bash
docker-compose logs mysql-single
```

### 测试失败

1. 确保所有容器都在运行
2. 检查端口是否正确
3. 查看测试日志中的 SQL 语句
4. 手动连接数据库验证：

```bash
mysql -h 127.0.0.1 -P 3306 -u root -proot123 testdb
```

## 性能测试

运行性能基准测试：

```bash
make benchmark
```

## 清理

```bash
# 停止并删除容器
docker-compose down

# 删除数据卷
docker-compose down -v
```

## 注意事项

1. 测试使用的是模拟主从同步，实际生产环境中由 MySQL 主从复制自动完成
2. 连接池统计可能因为连接复用而显示较少的连接数
3. 测试数据库密码为 `root123`，生产环境请使用强密码
4. 所有 4 个场景测试已通过验证 ✅
