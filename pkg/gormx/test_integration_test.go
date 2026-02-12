package gormx_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/tedwangl/go-util/pkg/gormx"
	"gorm.io/gorm"
)

// TestUser 测试用户模型
type TestUser struct {
	ID        int64     `gorm:"primarykey"`
	Name      string    `gorm:"size:100"`
	Email     string    `gorm:"size:100"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// TestOrder 测试订单模型
type TestOrder struct {
	ID        int64 `gorm:"primarykey"`
	UserID    int64 `gorm:"index"`
	Amount    float64
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// TestIntegration_Scenario1_SingleDB 场景 1：单库测试
func TestIntegration_Scenario1_SingleDB(t *testing.T) {
	cfg := gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
	)
	cfg.LogLevel = "info"

	client, err := gormx.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 自动迁移
	if err := client.DB.AutoMigrate(&TestUser{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// 测试写入
	user := &TestUser{Name: "张三", Email: "zhangsan@example.com"}
	if err := client.DB.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	t.Logf("✅ Created user: ID=%d, Name=%s", user.ID, user.Name)

	// 测试查询
	var users []TestUser
	if err := client.DB.Find(&users).Error; err != nil {
		t.Fatalf("Failed to find users: %v", err)
	}
	t.Logf("✅ Found %d users", len(users))

	// 测试连接池状态
	stats := client.Stats()
	t.Logf("📊 Connection Pool Stats:")
	t.Logf("   OpenConnections: %d", stats.OpenConnections)
	t.Logf("   InUse: %d", stats.InUse)
	t.Logf("   Idle: %d", stats.Idle)

	// 清理
	client.DB.Exec("DROP TABLE test_users")
}

// TestIntegration_Scenario2_MasterSlave 场景 2：主从测试
func TestIntegration_Scenario2_MasterSlave(t *testing.T) {
	cfg := gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3307)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
	)
	cfg.WithReplica("root:root123@tcp(localhost:3308)/testdb?charset=utf8mb4&parseTime=True&loc=Local")
	cfg.LogLevel = "info"

	client, err := gormx.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 自动迁移（主库）
	if err := client.DB.AutoMigrate(&TestUser{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// 手动在从库也创建表（模拟主从同步）
	// 实际生产环境中，这由 MySQL 主从复制自动完成
	slaveClient, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3308)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
	))
	slaveClient.DB.AutoMigrate(&TestUser{})
	slaveClient.Close()

	// 测试写入（应该走主库）
	user := &TestUser{Name: "李四", Email: "lisi@example.com"}
	if err := client.DB.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	t.Logf("✅ Created user on MASTER: ID=%d, Name=%s", user.ID, user.Name)

	// 手动同步数据到从库（模拟主从同步）
	slaveClient2, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3308)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
	))
	slaveClient2.DB.Create(&TestUser{ID: user.ID, Name: user.Name, Email: user.Email})
	slaveClient2.Close()

	// 测试查询（应该走从库）
	var users []TestUser
	if err := client.DB.Find(&users).Error; err != nil {
		t.Fatalf("Failed to find users: %v", err)
	}
	t.Logf("✅ Found %d users from SLAVE", len(users))

	// 测试事务（应该全部走主库）
	err = client.DB.Transaction(func(tx *gorm.DB) error {
		user2 := &TestUser{Name: "王五", Email: "wangwu@example.com"}
		if err := tx.Create(user2).Error; err != nil {
			return err
		}
		t.Logf("✅ Created user in transaction on MASTER: ID=%d", user2.ID)

		// 事务内的查询也应该走主库
		var count int64
		if err := tx.Model(&TestUser{}).Count(&count).Error; err != nil {
			return err
		}
		t.Logf("✅ Count in transaction from MASTER: %d", count)
		return nil
	})
	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// 测试连接池状态
	stats := client.Stats()
	t.Logf("📊 Connection Pool Stats:")
	t.Logf("   OpenConnections: %d (应该有 2 个连接池：主+从)", stats.OpenConnections)
	t.Logf("   InUse: %d", stats.InUse)
	t.Logf("   Idle: %d", stats.Idle)

	// 清理
	client.DB.Exec("DROP TABLE test_users")
	slaveClient3, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3308)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
	))
	slaveClient3.DB.Exec("DROP TABLE test_users")
	slaveClient3.Close()
}

// TestIntegration_Scenario3_MultiDatabase 场景 3：多数据库测试（纯分库）
func TestIntegration_Scenario3_MultiDatabase(t *testing.T) {
	// 先清理可能存在的表（所有相关数据库）
	cleanupDBs := []string{
		"root:root123@tcp(localhost:3315)/shard0", // 数据库0主库
		"root:root123@tcp(localhost:3316)/shard0", // 数据库0从库
		"root:root123@tcp(localhost:3317)/shard1", // 数据库1主库
		"root:root123@tcp(localhost:3318)/shard1", // 数据库1从库
	}
	for _, dsn := range cleanupDBs {
		cleanupClient, _ := gormx.NewClient(gormx.NewConfig("mysql", dsn+"?charset=utf8mb4&parseTime=True&loc=Local"))
		if cleanupClient != nil {
			cleanupClient.DB.Exec("DROP TABLE IF EXISTS test_users_0")
			cleanupClient.DB.Exec("DROP TABLE IF EXISTS test_users_1")
			cleanupClient.DB.Exec("DROP TABLE IF EXISTS test_orders_0")
			cleanupClient.DB.Exec("DROP TABLE IF EXISTS test_orders_1")
			cleanupClient.Close()
		}
	}

	cfg := gormx.NewConfig(
		"mysql",
		"", // DSN 留空，会自动使用第一个数据库
	)
	cfg.WithMultiDatabase([]gormx.DatabaseConfig{
		{
			Name:       "db0",
			Tables:     []string{"test_users_0", "test_orders_0"},
			DSN:        "root:root123@tcp(localhost:3315)/shard0?charset=utf8mb4&parseTime=True&loc=Local",
			ReplicaDSN: "root:root123@tcp(localhost:3316)/shard0?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			Name:       "db1",
			Tables:     []string{"test_users_1", "test_orders_1"},
			DSN:        "root:root123@tcp(localhost:3317)/shard1?charset=utf8mb4&parseTime=True&loc=Local",
			ReplicaDSN: "root:root123@tcp(localhost:3318)/shard1?charset=utf8mb4&parseTime=True&loc=Local",
		},
	})
	cfg.LogLevel = "info"

	client, err := gormx.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 迁移表（主库）
	if err := client.DB.Table("test_users_0").AutoMigrate(&TestUser{}); err != nil {
		t.Fatalf("Failed to migrate users_0: %v", err)
	}
	if err := client.DB.Table("test_users_1").AutoMigrate(&TestUser{}); err != nil {
		t.Fatalf("Failed to migrate users_1: %v", err)
	}
	if err := client.DB.Table("test_orders_0").AutoMigrate(&TestOrder{}); err != nil {
		t.Fatalf("Failed to migrate orders_0: %v", err)
	}
	if err := client.DB.Table("test_orders_1").AutoMigrate(&TestOrder{}); err != nil {
		t.Fatalf("Failed to migrate orders_1: %v", err)
	}

	// 手动在从库也创建表（模拟主从同步）
	db0SlaveClient, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3316)/shard0?charset=utf8mb4&parseTime=True&loc=Local",
	))
	db0SlaveClient.DB.Table("test_users_0").AutoMigrate(&TestUser{})
	db0SlaveClient.DB.Table("test_orders_0").AutoMigrate(&TestOrder{})
	db0SlaveClient.Close()

	db1SlaveClient, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3318)/shard1?charset=utf8mb4&parseTime=True&loc=Local",
	))
	db1SlaveClient.DB.Table("test_users_1").AutoMigrate(&TestUser{})
	db1SlaveClient.DB.Table("test_orders_1").AutoMigrate(&TestOrder{})
	db1SlaveClient.Close()

	// 模拟应用层分库逻辑（简化测试，只插入4条数据）
	testData := []struct {
		userID int64
		dbID   int64
	}{
		{1, 1}, {2, 0}, {3, 1}, {4, 0},
	}

	for _, data := range testData {
		// 写入用户
		user := &TestUser{
			ID:    data.userID,
			Name:  fmt.Sprintf("用户%d", data.userID),
			Email: fmt.Sprintf("user%d@example.com", data.userID),
		}
		tableName := fmt.Sprintf("test_users_%d", data.dbID)
		if err := client.DB.Table(tableName).Create(user).Error; err != nil {
			t.Fatalf("Failed to create user in %s: %v", tableName, err)
		}
		t.Logf("✅ Created user in DB%d: ID=%d, Name=%s", data.dbID, user.ID, user.Name)

		// 写入订单
		order := &TestOrder{
			UserID: data.userID,
			Amount: float64(data.userID) * 10.5,
		}
		orderTable := fmt.Sprintf("test_orders_%d", data.dbID)
		if err := client.DB.Table(orderTable).Create(order).Error; err != nil {
			t.Fatalf("Failed to create order in %s: %v", orderTable, err)
		}
		t.Logf("✅ Created order in DB%d: ID=%d, Amount=%.2f", data.dbID, order.ID, order.Amount)
	}

	// 手动同步数据到从库（模拟主从同步）
	db0SlaveClient2, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3316)/shard0?charset=utf8mb4&parseTime=True&loc=Local",
	))
	for _, data := range testData {
		if data.dbID == 0 {
			db0SlaveClient2.DB.Table("test_users_0").Create(&TestUser{
				ID:    data.userID,
				Name:  fmt.Sprintf("用户%d", data.userID),
				Email: fmt.Sprintf("user%d@example.com", data.userID),
			})
			db0SlaveClient2.DB.Table("test_orders_0").Create(&TestOrder{
				UserID: data.userID,
				Amount: float64(data.userID) * 10.5,
			})
		}
	}
	db0SlaveClient2.Close()

	db1SlaveClient2, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3318)/shard1?charset=utf8mb4&parseTime=True&loc=Local",
	))
	for _, data := range testData {
		if data.dbID == 1 {
			db1SlaveClient2.DB.Table("test_users_1").Create(&TestUser{
				ID:    data.userID,
				Name:  fmt.Sprintf("用户%d", data.userID),
				Email: fmt.Sprintf("user%d@example.com", data.userID),
			})
			db1SlaveClient2.DB.Table("test_orders_1").Create(&TestOrder{
				UserID: data.userID,
				Amount: float64(data.userID) * 10.5,
			})
		}
	}
	db1SlaveClient2.Close()

	// 等待同步
	time.Sleep(100 * time.Millisecond)

	// 测试查询（应该走从库）
	var users0 []TestUser
	if err := client.DB.Table("test_users_0").Find(&users0).Error; err != nil {
		t.Fatalf("Failed to find users from db0: %v", err)
	}
	t.Logf("✅ Found %d users from DB0 SLAVE", len(users0))

	var users1 []TestUser
	if err := client.DB.Table("test_users_1").Find(&users1).Error; err != nil {
		t.Fatalf("Failed to find users from db1: %v", err)
	}
	t.Logf("✅ Found %d users from DB1 SLAVE", len(users1))

	// 测试连接池状态
	stats := client.Stats()
	t.Logf("📊 Connection Pool Stats:")
	t.Logf("   OpenConnections: %d (应该有 4 个连接池：2数据库*2)", stats.OpenConnections)
	t.Logf("   InUse: %d", stats.InUse)
	t.Logf("   Idle: %d", stats.Idle)

	// 验证数据分布
	t.Logf("📈 Data Distribution:")
	t.Logf("   DB0: %d users", len(users0))
	t.Logf("   DB1: %d users", len(users1))
	t.Logf("   Total: %d users (expected 4)", len(users0)+len(users1))

	// 验证连接池优化：应该是 4 个
	if stats.OpenConnections > 4 {
		t.Errorf("❌ 连接池数量异常！预期 4 个，实际 %d 个", stats.OpenConnections)
	} else {
		t.Logf("✅ 连接池数量正确！第一个数据库主库被复用")
	}

	// 清理主库
	db0MasterCleanup, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3315)/shard0?charset=utf8mb4&parseTime=True&loc=Local",
	))
	db0MasterCleanup.DB.Exec("DROP TABLE IF EXISTS test_users_0")
	db0MasterCleanup.DB.Exec("DROP TABLE IF EXISTS test_orders_0")
	db0MasterCleanup.Close()

	db1MasterCleanup, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3317)/shard1?charset=utf8mb4&parseTime=True&loc=Local",
	))
	db1MasterCleanup.DB.Exec("DROP TABLE IF EXISTS test_users_1")
	db1MasterCleanup.DB.Exec("DROP TABLE IF EXISTS test_orders_1")
	db1MasterCleanup.Close()

	// 清理从库
	db0SlaveClient3, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3316)/shard0?charset=utf8mb4&parseTime=True&loc=Local",
	))
	db0SlaveClient3.DB.Exec("DROP TABLE IF EXISTS test_users_0")
	db0SlaveClient3.DB.Exec("DROP TABLE IF EXISTS test_orders_0")
	db0SlaveClient3.Close()

	db1SlaveClient3, _ := gormx.NewClient(gormx.NewConfig(
		"mysql",
		"root:root123@tcp(localhost:3318)/shard1?charset=utf8mb4&parseTime=True&loc=Local",
	))
	db1SlaveClient3.DB.Exec("DROP TABLE IF EXISTS test_users_1")
	db1SlaveClient3.DB.Exec("DROP TABLE IF EXISTS test_orders_1")
	db1SlaveClient3.Close()
}
