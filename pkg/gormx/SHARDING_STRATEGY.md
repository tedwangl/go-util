# 真正的分片（Sharding）设计方案

## 问题分析

### 当前"多数据库"vs"真正分片"的区别

| 特性 | 多数据库（当前） | 真正分片（目标） |
|------|----------------|----------------|
| 路由依据 | 表名 | 分片键（如 user_id） |
| 表名 | 不同（orders_2024, orders_2025） | 相同（orders） |
| 扩容 | 添加新库，不影响旧数据 | 需要数据重分布 |
| 使用方式 | 应用层指定表名 | 应用层提供分片键 |

### 核心挑战

1. **分片键提取**：如何从 `WHERE user_id = 123` 中提取 `user_id`？
2. **一致性哈希**：如何在扩容时最小化数据迁移？
3. **GORM 限制**：DBResolver 只支持表名路由，不支持条件路由

---

## 方案选择

### 方案 1：应用层分片（推荐）✅

**原理**：应用层计算分片 ID，使用虚拟表名路由

```go
// 配置：4 个物理分片，每个分片 256 个虚拟分片（总共 1024 个虚拟分片）
cfg.WithSharding(gormx.ShardingConfig{
    ShardKey:      "user_id",           // 分片键
    VirtualShards: 1024,                // 虚拟分片数（固定，不随扩容改变）
    PhysicalShards: []gormx.PhysicalShard{
        {ID: 0, DSN: "shard0.db", VirtualRange: [0, 255]},    // 256 个虚拟分片
        {ID: 1, DSN: "shard1.db", VirtualRange: [256, 511]},
        {ID: 2, DSN: "shard2.db", VirtualRange: [512, 767]},
        {ID: 3, DSN: "shard3.db", VirtualRange: [768, 1023]},
    },
})

// 使用：应用层提供分片键
user := &User{ID: 12345, Name: "张三"}
client.Shard("user_id", 12345).Create(user)  // 自动路由到正确的分片

// 查询
var user User
client.Shard("user_id", 12345).Where("id = ?", 12345).First(&user)
```

**优点**：
- 简单可控，应用层明确知道分片逻辑
- 虚拟分片解决扩容问题（只需重新映射虚拟分片到物理分片）
- 不依赖 SQL 解析

**缺点**：
- 需要应用层显式提供分片键
- 代码侵入性较强

---

### 方案 2：SQL 解析分片（复杂）❌

**原理**：拦截 SQL，解析 WHERE 条件提取分片键

```go
// 理想使用方式
var user User
client.DB.Where("user_id = ?", 12345).First(&user)  // 自动提取 user_id 并路由
```

**问题**：
- GORM 的 SQL 生成在 DBResolver 之后，无法提前解析
- 需要实现完整的 SQL 解析器（复杂度高）
- 性能开销大

**结论**：不推荐

---

### 方案 3：中间件代理（Proxy）❌

**原理**：使用 MySQL Proxy（如 Vitess、ShardingSphere）

**优点**：
- 对应用完全透明
- 功能强大（跨分片查询、分布式事务）

**缺点**：
- 需要额外部署中间件
- 增加运维复杂度
- 不符合 gormx 轻量级定位

**结论**：超出 gormx 范围

---

## 推荐方案：虚拟分片 + 应用层路由

### 核心设计

#### 1. 虚拟分片（Virtual Sharding）

```
虚拟分片数：1024（固定，2^10）
物理分片数：4（可扩容）

映射关系：
虚拟分片 0-255   → 物理分片 0
虚拟分片 256-511 → 物理分片 1
虚拟分片 512-767 → 物理分片 2
虚拟分片 768-1023 → 物理分片 3

扩容到 8 个物理分片：
虚拟分片 0-127   → 物理分片 0
虚拟分片 128-255 → 物理分片 4（新）
虚拟分片 256-383 → 物理分片 1
虚拟分片 384-511 → 物理分片 5（新）
...
```

**优势**：
- 虚拟分片数固定，扩容时只需重新映射
- 数据迁移量 = 50%（而不是传统的 75%）

#### 2. 一致性哈希

```go
// 计算虚拟分片 ID
func (c *Client) virtualShardID(shardKey interface{}) int {
    hash := crc32.ChecksumIEEE([]byte(fmt.Sprint(shardKey)))
    return int(hash % c.config.sharding.VirtualShards)
}

// 映射到物理分片
func (c *Client) physicalShardID(virtualID int) int {
    for _, shard := range c.config.sharding.PhysicalShards {
        if virtualID >= shard.VirtualRange[0] && virtualID <= shard.VirtualRange[1] {
            return shard.ID
        }
    }
    return 0 // 默认分片
}
```

#### 3. 表名路由（利用 DBResolver）

```go
// 内部实现：将虚拟分片 ID 编码到表名
func (c *Client) Shard(shardKeyName string, shardKeyValue interface{}) *gorm.DB {
    virtualID := c.virtualShardID(shardKeyValue)
    physicalID := c.physicalShardID(virtualID)
    
    // 使用虚拟表名路由（DBResolver 支持）
    tableSuffix := fmt.Sprintf("_shard_%d", physicalID)
    return c.DB.Table(c.currentTable() + tableSuffix)
}
```

---

## 配置示例

### 配置结构

```go
type ShardingConfig struct {
    // 分片键（用于文档说明，实际由应用层提供）
    ShardKey string
    
    // 虚拟分片数（固定，建议 1024 或 2048）
    VirtualShards int
    
    // 物理分片列表
    PhysicalShards []PhysicalShard
}

type PhysicalShard struct {
    ID           int      // 物理分片 ID
    DSN          string   // 主库地址
    ReplicaDSN   string   // 从库地址（可选）
    VirtualRange [2]int   // 虚拟分片范围 [start, end]
}
```

### 使用示例

```go
// 1. 配置分片
cfg := gormx.NewConfig("mysql", "")
cfg.WithSharding(gormx.ShardingConfig{
    ShardKey:      "user_id",
    VirtualShards: 1024,
    PhysicalShards: []gormx.PhysicalShard{
        {
            ID:           0,
            DSN:          "root:pass@tcp(shard0:3306)/db",
            ReplicaDSN:   "root:pass@tcp(shard0-slave:3306)/db",
            VirtualRange: [2]int{0, 255},
        },
        {
            ID:           1,
            DSN:          "root:pass@tcp(shard1:3306)/db",
            ReplicaDSN:   "root:pass@tcp(shard1-slave:3306)/db",
            VirtualRange: [2]int{256, 511},
        },
        {
            ID:           2,
            DSN:          "root:pass@tcp(shard2:3306)/db",
            ReplicaDSN:   "root:pass@tcp(shard2-slave:3306)/db",
            VirtualRange: [2]int{512, 767},
        },
        {
            ID:           3,
            DSN:          "root:pass@tcp(shard3:3306)/db",
            ReplicaDSN:   "root:pass@tcp(shard3-slave:3306)/db",
            VirtualRange: [2]int{768, 1023},
        },
    },
})

client, _ := gormx.NewClient(cfg)

// 2. 写入数据
user := &User{ID: 12345, Name: "张三"}
client.Shard("user_id", user.ID).Create(user)

// 3. 查询数据
var user User
client.Shard("user_id", 12345).Where("id = ?", 12345).First(&user)

// 4. 批量查询（同一分片）
var users []User
client.Shard("user_id", 12345).Where("user_id = ?", 12345).Find(&users)

// 5. 跨分片查询（应用层聚合）
userIDs := []int64{100, 200, 300, 400}
var allUsers []User
for _, uid := range userIDs {
    var users []User
    client.Shard("user_id", uid).Where("id = ?", uid).Find(&users)
    allUsers = append(allUsers, users...)
}
```

---

## 扩容方案

### 扩容步骤

```
初始：4 个物理分片，1024 个虚拟分片
目标：8 个物理分片，1024 个虚拟分片（不变）

步骤：
1. 部署 4 个新物理分片（shard4-7）
2. 更新配置，重新映射虚拟分片
3. 数据迁移（只迁移 50% 的虚拟分片）
4. 切换流量
```

### 配置变更

```go
// 扩容前
PhysicalShards: []gormx.PhysicalShard{
    {ID: 0, VirtualRange: [0, 255]},    // 256 个虚拟分片
    {ID: 1, VirtualRange: [256, 511]},
    {ID: 2, VirtualRange: [512, 767]},
    {ID: 3, VirtualRange: [768, 1023]},
}

// 扩容后
PhysicalShards: []gormx.PhysicalShard{
    {ID: 0, VirtualRange: [0, 127]},      // 128 个虚拟分片
    {ID: 4, VirtualRange: [128, 255]},    // 新分片（迁移自 shard0）
    {ID: 1, VirtualRange: [256, 383]},
    {ID: 5, VirtualRange: [384, 511]},    // 新分片（迁移自 shard1）
    {ID: 2, VirtualRange: [512, 639]},
    {ID: 6, VirtualRange: [640, 767]},    // 新分片（迁移自 shard2）
    {ID: 3, VirtualRange: [768, 895]},
    {ID: 7, VirtualRange: [896, 1023]},   // 新分片（迁移自 shard3）
}
```

### 数据迁移

```go
// 迁移工具伪代码
func migrate(oldShard, newShard int, virtualRange [2]int) {
    for virtualID := virtualRange[0]; virtualID <= virtualRange[1]; virtualID++ {
        // 1. 从旧分片读取数据
        rows := queryFromShard(oldShard, virtualID)
        
        // 2. 写入新分片
        insertToShard(newShard, rows)
        
        // 3. 验证
        verify(oldShard, newShard, virtualID)
        
        // 4. 删除旧数据（可选）
        deleteFromShard(oldShard, virtualID)
    }
}
```

---

## 部署架构

### 推荐架构

```
应用层（gormx）
    ↓
虚拟分片路由（1024 个虚拟分片）
    ↓
物理分片（4-16 个物理分片）
    ↓
MySQL 主从（每个物理分片 1 主 N 从）
```

### 物理部署

```yaml
# docker-compose.yml
services:
  # 物理分片 0
  shard0-master:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root123
      MYSQL_DATABASE: shard0
    ports:
      - "3320:3306"
  
  shard0-slave:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root123
      MYSQL_DATABASE: shard0
    ports:
      - "3321:3306"
  
  # 物理分片 1
  shard1-master:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root123
      MYSQL_DATABASE: shard1
    ports:
      - "3322:3306"
  
  shard1-slave:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root123
      MYSQL_DATABASE: shard1
    ports:
      - "3323:3306"
  
  # ... 更多分片
```

---

## 限制和注意事项

### 1. 不支持的功能

- ❌ 跨分片 JOIN
- ❌ 跨分片事务
- ❌ 跨分片聚合（COUNT、SUM 等）
- ❌ 自动分片键提取

### 2. 应用层职责

- ✅ 提供分片键
- ✅ 跨分片查询聚合
- ✅ 分布式事务（Saga/TCC）
- ✅ 数据迁移工具

### 3. gormx 职责

- ✅ 虚拟分片计算
- ✅ 物理分片路由
- ✅ 连接池管理
- ✅ 读写分离

---

## 总结

**推荐方案**：虚拟分片 + 应用层路由

**核心优势**：
1. 虚拟分片解决扩容问题（数据迁移量 50%）
2. 应用层路由简单可控
3. 利用 DBResolver 的表名路由能力
4. 不需要 SQL 解析

**使用方式**：
```go
client.Shard("user_id", 12345).Create(&user)
client.Shard("user_id", 12345).Where("id = ?", 12345).First(&user)
```

**扩容方式**：
1. 部署新物理分片
2. 更新虚拟分片映射
3. 迁移数据（50%）
4. 切换流量
