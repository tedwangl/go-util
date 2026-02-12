package gormx_test

import (
	"fmt"
	"time"

	"github.com/tedwangl/go-util/pkg/gormx"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// Example_dbResolverBasic DBResolver 基本使用
func Example_dbResolverBasic() {
	// 1. 主从读写分离
	cfg := &gormx.Config{
		Driver: "mysql",
		DSN:    "root:password@tcp(master:3306)/db?charset=utf8mb4&parseTime=True",
		Replicas: []string{
			"root:password@tcp(slave1:3306)/db?charset=utf8mb4&parseTime=True",
			"root:password@tcp(slave2:3306)/db?charset=utf8mb4&parseTime=True",
		},
		MaxOpenConns: 100,
		MaxIdleConns: 10,
	}

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	var users []User

	// SELECT → 从库（自动负载均衡）
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

	// 强制走主库
	client.DB.Clauses(dbresolver.Write).Find(&users)

	// 强制走从库
	client.DB.Clauses(dbresolver.Read).Find(&users)
}

// Example_dbResolverMultiMaster 多主配置
func Example_dbResolverMultiMaster() {
	cfg := &gormx.Config{
		Driver: "mysql",
		DSN:    "root:password@tcp(master1:3306)/db?charset=utf8mb4&parseTime=True",
		Sources: []string{
			"root:password@tcp(master2:3306)/db?charset=utf8mb4&parseTime=True",
			"root:password@tcp(master3:3306)/db?charset=utf8mb4&parseTime=True",
		},
		Replicas: []string{
			"root:password@tcp(slave1:3306)/db?charset=utf8mb4&parseTime=True",
			"root:password@tcp(slave2:3306)/db?charset=utf8mb4&parseTime=True",
		},
	}

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	// 写操作会在 master1/master2/master3 之间负载均衡
	client.DB.Create(&User{Name: "张三"})
}

// Example_dbResolverSharding 分库分表
func Example_dbResolverSharding() {
	cfg := &gormx.Config{
		Driver: "mysql",
		DSN:    "root:password@tcp(default-master:3306)/db?charset=utf8mb4&parseTime=True",
		Replicas: []string{
			"root:password@tcp(default-slave:3306)/db?charset=utf8mb4&parseTime=True",
		},
		// 分片配置
		Shards: []gormx.ShardConfig{
			{
				Name:   "shard1",
				Tables: []string{"orders_*", "order_items_*"}, // 匹配 orders_xxx 表
				Sources: []string{
					"root:password@tcp(shard1-master:3306)/shard1?charset=utf8mb4&parseTime=True",
				},
				Replicas: []string{
					"root:password@tcp(shard1-slave:3306)/shard1?charset=utf8mb4&parseTime=True",
				},
			},
			{
				Name:   "shard2",
				Tables: []string{"users_*", "user_profiles_*"},
				Sources: []string{
					"root:password@tcp(shard2-master:3306)/shard2?charset=utf8mb4&parseTime=True",
				},
				Replicas: []string{
					"root:password@tcp(shard2-slave:3306)/shard2?charset=utf8mb4&parseTime=True",
				},
			},
		},
	}

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	// 自动路由到对应分片
	var orders []Order
	client.DB.Table("orders_202401").Find(&orders) // → shard1

	var users []User
	client.DB.Table("users_vip").Find(&users) // → shard2

	// 默认表走默认库
	var products []Product
	client.DB.Find(&products) // → default master/slave
}

// Example_dbResolverManualSharding 手动分片路由
func Example_dbResolverManualSharding() {
	cfg := &gormx.Config{
		Driver: "mysql",
		DSN:    "root:password@tcp(master:3306)/db?charset=utf8mb4&parseTime=True",
		Shards: []gormx.ShardConfig{
			{
				Name:   "shard1",
				Tables: []string{"orders"},
				Sources: []string{
					"root:password@tcp(shard1:3306)/shard1?charset=utf8mb4&parseTime=True",
				},
			},
		},
	}

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	// 根据用户 ID 计算分片
	userID := int64(12345)
	shardID := userID % 10 // 10 个分片

	tableName := fmt.Sprintf("orders_%d", shardID)

	var orders []Order
	client.DB.Table(tableName).Where("user_id = ?", userID).Find(&orders)
}

// Example_dbResolverLoadBalance 负载均衡策略
func Example_dbResolverLoadBalance() {
	// DBResolver 默认使用 RandomPolicy（随机负载均衡）
	// 也可以自定义策略

	cfg := &gormx.Config{
		Driver: "mysql",
		DSN:    "root:password@tcp(master:3306)/db?charset=utf8mb4&parseTime=True",
		Replicas: []string{
			"root:password@tcp(slave1:3306)/db?charset=utf8mb4&parseTime=True", // weight=1
			"root:password@tcp(slave2:3306)/db?charset=utf8mb4&parseTime=True", // weight=1
		},
	}

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	// 读操作会在 slave1 和 slave2 之间随机选择
	var users []User
	client.DB.Find(&users)
}

// Example_dbResolverConnectionPool 连接池配置
func Example_dbResolverConnectionPool() {
	cfg := &gormx.Config{
		Driver: "mysql",
		DSN:    "root:password@tcp(master:3306)/db?charset=utf8mb4&parseTime=True",
		Replicas: []string{
			"root:password@tcp(slave:3306)/db?charset=utf8mb4&parseTime=True",
		},
		MaxOpenConns: 100, // 主库最大连接数
		MaxIdleConns: 10,  // 主库最大空闲连接数
	}

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	// 可以为从库单独配置连接池
	client.DB.Use(dbresolver.Register(dbresolver.Config{}).
		SetMaxIdleConns(5).
		SetMaxOpenConns(50).
		SetConnMaxLifetime(time.Hour))
}

// Order 订单模型
type Order struct {
	ID     int64 `gorm:"primarykey"`
	UserID int64 `gorm:"index"`
	Amount float64
}

// Product 产品模型
type Product struct {
	ID   int64  `gorm:"primarykey"`
	Name string `gorm:"size:100"`
}
