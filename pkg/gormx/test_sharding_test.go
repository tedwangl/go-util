package gormx_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/tedwangl/go-util/pkg/gormx"
)

// TestIntegration_Scenario4_Sharding 场景 4：真正的分片测试
func TestIntegration_Scenario4_Sharding(t *testing.T) {
	// 先清理可能存在的表（所有分片）
	cleanupShards := []string{
		"root:root123@tcp(localhost:3315)/shard0?charset=utf8mb4&parseTime=True&loc=Local",
		"root:root123@tcp(localhost:3316)/shard0?charset=utf8mb4&parseTime=True&loc=Local",
		"root:root123@tcp(localhost:3317)/shard1?charset=utf8mb4&parseTime=True&loc=Local",
		"root:root123@tcp(localhost:3318)/shard1?charset=utf8mb4&parseTime=True&loc=Local",
	}
	for _, dsn := range cleanupShards {
		cleanupClient, _ := gormx.NewClient(gormx.NewConfig("mysql", dsn))
		if cleanupClient != nil {
			cleanupClient.DB.Exec("DROP TABLE IF EXISTS test_users")
			cleanupClient.DB.Exec("DROP TABLE IF EXISTS test_orders")
			cleanupClient.Close()
		}
	}

	// 配置分片（2 个分片，每个分片都有主从）
	cfg := gormx.NewConfig("mysql", "")
	cfg.WithSharding(gormx.ShardingConfig{
		Algorithm:  "mod",
		ShardCount: 2,
		Shards: []gormx.ShardNode{
			{
				ID:         0,
				Name:       "shard0",
				DSN:        "root:root123@tcp(localhost:3315)/shard0?charset=utf8mb4&parseTime=True&loc=Local",
				ReplicaDSN: "root:root123@tcp(localhost:3316)/shard0?charset=utf8mb4&parseTime=True&loc=Local",
			},
			{
				ID:         1,
				Name:       "shard1",
				DSN:        "root:root123@tcp(localhost:3317)/shard1?charset=utf8mb4&parseTime=True&loc=Local",
				ReplicaDSN: "root:root123@tcp(localhost:3318)/shard1?charset=utf8mb4&parseTime=True&loc=Local",
			},
		},
	})
	cfg.LogLevel = "info"

	client, err := gormx.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 在所有分片创建相同的表结构
	for i := 0; i < 2; i++ { // 改为 2 个分片
		if err := client.ShardByID(i).AutoMigrate(&TestUser{}); err != nil {
			t.Fatalf("Failed to migrate users in shard %d: %v", i, err)
		}
		if err := client.ShardByID(i).AutoMigrate(&TestOrder{}); err != nil {
			t.Fatalf("Failed to migrate orders in shard %d: %v", i, err)
		}
	}

	// 手动在从库也创建表（模拟主从同步）
	shard0SlaveClient, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3316)/shard0?charset=utf8mb4&parseTime=True&loc=Local",
	))
	shard0SlaveClient.DB.AutoMigrate(&TestUser{}, &TestOrder{})
	shard0SlaveClient.Close()

	shard1SlaveClient, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3318)/shard1?charset=utf8mb4&parseTime=True&loc=Local",
	))
	shard1SlaveClient.DB.AutoMigrate(&TestUser{}, &TestOrder{})
	shard1SlaveClient.Close()

	// 测试数据：插入 8 个用户，应该分布到 2 个分片
	testUsers := []struct {
		id      int64
		name    string
		shardID int
	}{
		{100, "用户100", 0}, // 100 % 2 = 0
		{101, "用户101", 1}, // 101 % 2 = 1
		{102, "用户102", 0}, // 102 % 2 = 0
		{103, "用户103", 1}, // 103 % 2 = 1
		{200, "用户200", 0}, // 200 % 2 = 0
		{201, "用户201", 1}, // 201 % 2 = 1
		{202, "用户202", 0}, // 202 % 2 = 0
		{203, "用户203", 1}, // 203 % 2 = 1
	}

	// 写入数据（自动路由到正确的分片）
	for _, tu := range testUsers {
		user := &TestUser{
			ID:    tu.id,
			Name:  tu.name,
			Email: fmt.Sprintf("user%d@example.com", tu.id),
		}

		// 使用分片键写入
		if err := client.Shard(tu.id).Create(user).Error; err != nil {
			t.Fatalf("Failed to create user %d: %v", tu.id, err)
		}
		t.Logf("✅ Created user in SHARD%d: ID=%d, Name=%s", tu.shardID, user.ID, user.Name)

		// 创建订单
		order := &TestOrder{
			UserID: tu.id,
			Amount: float64(tu.id) * 10.5,
		}
		if err := client.Shard(tu.id).Create(order).Error; err != nil {
			t.Fatalf("Failed to create order for user %d: %v", tu.id, err)
		}
		t.Logf("✅ Created order in SHARD%d: ID=%d, Amount=%.2f", tu.shardID, order.ID, order.Amount)
	}

	// 手动同步数据到从库（模拟主从同步）
	shard0SlaveClient2, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3316)/shard0?charset=utf8mb4&parseTime=True&loc=Local",
	))
	for _, tu := range testUsers {
		if tu.shardID == 0 {
			shard0SlaveClient2.DB.Create(&TestUser{
				ID:    tu.id,
				Name:  tu.name,
				Email: fmt.Sprintf("user%d@example.com", tu.id),
			})
			shard0SlaveClient2.DB.Create(&TestOrder{
				UserID: tu.id,
				Amount: float64(tu.id) * 10.5,
			})
		}
	}
	shard0SlaveClient2.Close()

	shard1SlaveClient2, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3318)/shard1?charset=utf8mb4&parseTime=True&loc=Local",
	))
	for _, tu := range testUsers {
		if tu.shardID == 1 {
			shard1SlaveClient2.DB.Create(&TestUser{
				ID:    tu.id,
				Name:  tu.name,
				Email: fmt.Sprintf("user%d@example.com", tu.id),
			})
			shard1SlaveClient2.DB.Create(&TestOrder{
				UserID: tu.id,
				Amount: float64(tu.id) * 10.5,
			})
		}
	}
	shard1SlaveClient2.Close()

	// 等待同步
	time.Sleep(100 * time.Millisecond)

	// 测试查询（应该走从库）
	for _, tu := range testUsers {
		var user TestUser
		if err := client.Shard(tu.id).Where("id = ?", tu.id).First(&user).Error; err != nil {
			t.Fatalf("Failed to find user %d: %v", tu.id, err)
		}
		if user.Name != tu.name {
			t.Errorf("User name mismatch: expected %s, got %s", tu.name, user.Name)
		}
		t.Logf("✅ Found user from SHARD%d SLAVE: ID=%d, Name=%s", tu.shardID, user.ID, user.Name)
	}

	// 测试跨分片查询（应用层聚合）
	var allUsers []TestUser
	for i := 0; i < 2; i++ {
		var users []TestUser
		if err := client.ShardByID(i).Find(&users).Error; err != nil {
			t.Fatalf("Failed to find users from shard %d: %v", i, err)
		}
		allUsers = append(allUsers, users...)
		t.Logf("📊 SHARD%d: %d users", i, len(users))
	}

	if len(allUsers) != len(testUsers) {
		t.Errorf("Total users mismatch: expected %d, got %d", len(testUsers), len(allUsers))
	}
	t.Logf("✅ Total users across all shards: %d", len(allUsers))

	// 测试连接池状态
	stats := client.Stats()
	t.Logf("📊 Connection Pool Stats:")
	t.Logf("   OpenConnections: %d", stats.OpenConnections)
	t.Logf("   InUse: %d", stats.InUse)
	t.Logf("   Idle: %d", stats.Idle)

	// 验证数据分布
	shardCounts := make(map[int]int)
	for _, tu := range testUsers {
		shardCounts[tu.shardID]++
	}
	t.Logf("📈 Data Distribution:")
	for i := 0; i < 2; i++ {
		t.Logf("   SHARD%d: %d users", i, shardCounts[i])
	}

	// 清理
	for i := 0; i < 2; i++ {
		client.ShardByID(i).Exec("DROP TABLE IF EXISTS test_users")
		client.ShardByID(i).Exec("DROP TABLE IF EXISTS test_orders")
	}

	// 清理从库
	shard0SlaveClient3, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3316)/shard0?charset=utf8mb4&parseTime=True&loc=Local",
	))
	shard0SlaveClient3.DB.Exec("DROP TABLE IF EXISTS test_users")
	shard0SlaveClient3.DB.Exec("DROP TABLE IF EXISTS test_orders")
	shard0SlaveClient3.Close()

	shard1SlaveClient3, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3318)/shard1?charset=utf8mb4&parseTime=True&loc=Local",
	))
	shard1SlaveClient3.DB.Exec("DROP TABLE IF EXISTS test_users")
	shard1SlaveClient3.DB.Exec("DROP TABLE IF EXISTS test_orders")
	shard1SlaveClient3.Close()
}
