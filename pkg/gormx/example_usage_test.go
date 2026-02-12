package gormx_test

import (
	"time"

	"github.com/tedwangl/go-util/pkg/gormx"
	"gorm.io/gorm"
)

// Example_singleDatabase 场景 1：单库（最简单）
func Example_singleDatabase() {
	// 适用场景：开发环境、小型应用、TiDB/CockroachDB 等分布式数据库
	cfg := gormx.NewConfig(
		"mysql",
		"root:password@tcp(db.local:3306)/myapp?charset=utf8mb4",
	)

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	// 所有操作都走同一个连接池
	var users []User
	client.DB.Find(&users)
	client.DB.Create(&User{Name: "张三"})
}

// Example_masterSlave 场景 2：主从读写分离
func Example_masterSlave() {
	// 适用场景：读多写少、需要读写分离的应用
	cfg := gormx.NewConfig(
		"mysql",
		"root:password@tcp(master.db.local:3306)/myapp?charset=utf8mb4",
	)

	// 配置从库
	cfg.WithReplica("root:password@tcp(slave.db.local:3306)/myapp?charset=utf8mb4")

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	var users []User

	// SELECT → 从库
	client.DB.Find(&users)

	// INSERT/UPDATE/DELETE → 主库
	client.DB.Create(&User{Name: "张三"})
	client.DB.Model(&User{}).Where("id = ?", 1).Update("name", "李四")

	// 事务 → 全部走主库
	client.DB.Transaction(func(tx *gorm.DB) error {
		tx.Create(&User{Name: "王五"})
		tx.Find(&users) // 事务内的 SELECT 也走主库
		return nil
	})
}

// Example_multiDatabase 场景 3：多数据库（按表名分库）
func Example_multiDatabase() {
	// 适用场景：按表名路由到不同数据库（如按年份分库）
	cfg := gormx.NewConfig(
		"mysql",
		"", // 多数据库模式下可以留空，会自动使用第一个数据库的 DSN
	)

	// 配置多数据库（每个数据库必须指定表名）
	cfg.WithMultiDatabase([]gormx.DatabaseConfig{
		{
			Name:       "db_2024",
			Tables:     []string{"orders_2024*", "order_items_2024*"},
			DSN:        "root:password@tcp(db-2024.local:3306)/orders_2024?charset=utf8mb4",
			ReplicaDSN: "root:password@tcp(db-2024-slave.local:3306)/orders_2024?charset=utf8mb4",
		},
		{
			Name:       "db_2025",
			Tables:     []string{"orders_2025*", "order_items_2025*"},
			DSN:        "root:password@tcp(db-2025.local:3306)/orders_2025?charset=utf8mb4",
			ReplicaDSN: "root:password@tcp(db-2025-slave.local:3306)/orders_2025?charset=utf8mb4",
		},
	})

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	// 查询 2024 年订单 → db-2024-slave
	var orders2024 []Order
	client.DB.Table("orders_202401").Find(&orders2024)

	// 查询 2025 年订单 → db-2025-slave
	var orders2025 []Order
	client.DB.Table("orders_202501").Find(&orders2025)

	// 创建订单 → db-2024（主库）
	client.DB.Table("orders_202401").Create(&Order{Amount: 100})
}

// Example_multiDatabaseWithoutReplica 场景 4：多数据库（无从库）
func Example_multiDatabaseWithoutReplica() {
	// 适用场景：多数据库但不需要读写分离
	cfg := gormx.NewConfig(
		"mysql",
		"", // 留空即可
	)

	// 配置多数据库（不配置从库）
	cfg.WithMultiDatabase([]gormx.DatabaseConfig{
		{
			Name:   "db1",
			Tables: []string{"users_*", "profiles_*"},
			DSN:    "root:password@tcp(db1.local:3306)/db1?charset=utf8mb4",
		},
		{
			Name:   "db2",
			Tables: []string{"orders_*", "payments_*"},
			DSN:    "root:password@tcp(db2.local:3306)/db2?charset=utf8mb4",
		},
	})

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	// 所有表都必须匹配数据库规则
	var users []User
	client.DB.Table("users_001").Find(&users) // → db1

	var orders []Order
	client.DB.Table("orders_001").Find(&orders) // → db2
}

// Example_fluentAPI 场景 5：链式调用
func Example_fluentAPI() {
	cfg := gormx.NewConfig(
		"mysql",
		"root:password@tcp(master.db.local:3306)/myapp?charset=utf8mb4",
	)
	cfg.MaxOpenConns = 200

	// 链式配置主从
	cfg.WithReplica("root:password@tcp(slave.db.local:3306)/myapp?charset=utf8mb4")

	client, _ := gormx.NewClient(cfg)
	defer client.Close()
}

// Example_customConfig 场景 6：自定义配置
func Example_customConfig() {
	cfg := gormx.NewConfig(
		"mysql",
		"root:password@tcp(db.local:3306)/myapp?charset=utf8mb4",
	)

	// 自定义连接池参数
	cfg.MaxOpenConns = 200
	cfg.MaxIdleConns = 50
	cfg.LogLevel = "info"
	cfg.SlowThreshold = 500 * time.Millisecond

	client, _ := gormx.NewClient(cfg)
	defer client.Close()
}

// User 示例模型
type User struct {
	ID   int64
	Name string
}

// Order 示例模型
type Order struct {
	ID     int64
	Amount float64
}
