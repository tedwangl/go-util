package gormx_test

import (
	"fmt"

	"github.com/tedwangl/go-util/pkg/gormx"
)

// Example_sharding 场景：真正的分片（按分片键路由）
func Example_sharding() {
	// 配置分片
	cfg := gormx.NewConfig("mysql", "") // DSN 留空，使用第一个分片

	cfg.WithSharding(gormx.ShardingConfig{
		Algorithm:  "mod", // 取模算法
		ShardCount: 4,     // 4 个分片
		Shards: []gormx.ShardNode{
			{
				ID:         0,
				Name:       "shard0",
				DSN:        "root:password@tcp(shard0:3306)/db?charset=utf8mb4",
				ReplicaDSN: "root:password@tcp(shard0-slave:3306)/db?charset=utf8mb4",
			},
			{
				ID:         1,
				Name:       "shard1",
				DSN:        "root:password@tcp(shard1:3306)/db?charset=utf8mb4",
				ReplicaDSN: "root:password@tcp(shard1-slave:3306)/db?charset=utf8mb4",
			},
			{
				ID:         2,
				Name:       "shard2",
				DSN:        "root:password@tcp(shard2:3306)/db?charset=utf8mb4",
				ReplicaDSN: "root:password@tcp(shard2-slave:3306)/db?charset=utf8mb4",
			},
			{
				ID:         3,
				Name:       "shard3",
				DSN:        "root:password@tcp(shard3:3306)/db?charset=utf8mb4",
				ReplicaDSN: "root:password@tcp(shard3-slave:3306)/db?charset=utf8mb4",
			},
		},
	})

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	// 使用方式 1：通过分片键自动计算分片 ID
	userID := int64(12345)
	user := &User{ID: userID, Name: "张三"}

	// 写入（自动路由到 shard1，因为 12345 % 4 = 1）
	client.Shard(userID).Model(&User{}).Create(user)

	// 查询（自动路由到 shard1）
	var result User
	client.Shard(userID).Model(&User{}).Where("id = ?", userID).First(&result)

	// 使用方式 2：直接指定分片 ID（用于已知分片 ID 的场景）
	client.ShardByID(1).Model(&User{}).Where("id = ?", userID).First(&result)

	// 使用方式 3：使用 Table 方法
	client.Shard(userID).Table("users").Where("id = ?", userID).First(&result)

	// 跨分片查询（应用层聚合）
	userIDs := []int64{100, 200, 300, 400}
	var allUsers []User
	for _, uid := range userIDs {
		var users []User
		client.Shard(uid).Model(&User{}).Where("id = ?", uid).Find(&users)
		allUsers = append(allUsers, users...)
	}

	fmt.Printf("Found %d users across shards\n", len(allUsers))
}

// Example_shardingHash 场景：使用哈希算法分片
func Example_shardingHash() {
	cfg := gormx.NewConfig("mysql", "")

	cfg.WithSharding(gormx.ShardingConfig{
		Algorithm:  "hash", // 哈希算法（支持字符串分片键）
		ShardCount: 4,
		Shards: []gormx.ShardNode{
			{ID: 0, DSN: "root:password@tcp(shard0:3306)/db"},
			{ID: 1, DSN: "root:password@tcp(shard1:3306)/db"},
			{ID: 2, DSN: "root:password@tcp(shard2:3306)/db"},
			{ID: 3, DSN: "root:password@tcp(shard3:3306)/db"},
		},
	})

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	// 使用字符串作为分片键
	tenantID := "tenant_abc"
	order := &Order{ID: 1, Amount: 100}

	client.Shard(tenantID).Model(&Order{}).Create(order)
	client.Shard(tenantID).Model(&Order{}).Where("id = ?", 1).First(order)
}

// Example_shardingWithoutReplica 场景：分片但不配置从库
func Example_shardingWithoutReplica() {
	cfg := gormx.NewConfig("mysql", "")

	cfg.WithSharding(gormx.ShardingConfig{
		Algorithm:  "mod",
		ShardCount: 2,
		Shards: []gormx.ShardNode{
			{ID: 0, DSN: "root:password@tcp(shard0:3306)/db"}, // 无从库
			{ID: 1, DSN: "root:password@tcp(shard1:3306)/db"}, // 无从库
		},
	})

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	userID := int64(12345)
	client.Shard(userID).Model(&User{}).Create(&User{ID: userID, Name: "李四"})
}
