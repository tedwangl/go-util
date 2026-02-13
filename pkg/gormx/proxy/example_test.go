package proxy_test

import (
	"time"

	"github.com/tedwangl/go-util/pkg/gormx/proxy"
	"gorm.io/gorm"
)

// User 示例模型
type User struct {
	ID        int64          `gorm:"primarykey"`
	Name      string         `gorm:"size:100"`
	Email     string         `gorm:"size:100;uniqueIndex"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Order 分片表示例
type Order struct {
	ID        int64     `gorm:"primarykey"`
	UserID    int64     `gorm:"index"`
	Amount    float64   `gorm:"type:decimal(10,2)"`
	Status    string    `gorm:"size:20"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// Example_singleMode 单库模式
func Example_singleMode() {
	cfg := &proxy.Config{
		Mode: "single",
		ProxySQL: &proxy.ProxySQLConfig{
			Addr:     "127.0.0.1:6033",
			Username: "root",
			Password: "root123",
			Database: "testdb",
		},
		MaxOpenConns: 100,
		LogLevel:     "info",
	}

	client, _ := proxy.NewClient(cfg)
	defer client.Close()

	// 自动迁移
	client.AutoMigrate(&User{})

	// 正常使用
	client.Create(&User{Name: "张三", Email: "zhangsan@example.com"})

	var users []User
	client.Find(&users)
}

// Example_masterSlaveMode 主从模式（通过 ProxySQL）
func Example_masterSlaveMode() {
	cfg := &proxy.Config{
		Mode: "master-slave",
		ProxySQL: &proxy.ProxySQLConfig{
			Addr:     "127.0.0.1:6033", // ProxySQL 自动路由
			Username: "root",
			Password: "root123",
			Database: "testdb",
		},
		MaxOpenConns: 100,
		LogLevel:     "info",
	}

	client, _ := proxy.NewClient(cfg)
	defer client.Close()

	var users []User

	// SELECT → 从库（ProxySQL 自动路由）
	client.Find(&users)

	// INSERT → 主库（ProxySQL 自动路由）
	client.Create(&User{Name: "李四", Email: "lisi@example.com"})

	// UPDATE → 主库（ProxySQL 自动路由）
	client.Model(&User{}).Where("id = ?", 1).Update("name", "王五")

	// 事务 → 全部走主库（ProxySQL 自动路由）
	client.Transaction(func(tx *gorm.DB) error {
		tx.Create(&User{Name: "赵六", Email: "zhaoliu@example.com"})
		tx.Find(&users) // 事务内的 SELECT 也走主库
		return nil
	})
}

// Example_shardingMode 分片模式
func Example_shardingMode() {
	cfg := &proxy.Config{
		Mode: "sharding",
		ProxySQL: &proxy.ProxySQLConfig{
			Addr:     "127.0.0.1:6033",
			Username: "root",
			Password: "root123",
			Database: "testdb",
		},
		Sharding: &proxy.ShardingConfig{
			Enable:     true,
			ShardCount: 4,
			Algorithm:  "mod",
			Tables: []*proxy.ShardingTable{
				{
					TableName:          "orders",
					ShardingKey:        "user_id",
					ShardCount:         4,
					Algorithm:          "mod",
					GeneratePrimaryKey: true,
				},
			},
		},
		MaxOpenConns: 100,
		LogLevel:     "info",
	}

	client, _ := proxy.NewClient(cfg)
	defer client.Close()

	// 自动迁移（会创建 orders_0, orders_1, orders_2, orders_3）
	client.AutoMigrate(&Order{})

	// 插入数据（自动路由到对应分片）
	client.Create(&Order{
		UserID: 1001, // 根据 user_id 分片
		Amount: 99.99,
		Status: "pending",
	})

	client.Create(&Order{
		UserID: 1002,
		Amount: 199.99,
		Status: "paid",
	})

	// 查询（需要指定分片键）
	var orders []Order
	client.Where("user_id = ?", 1001).Find(&orders)

	// 更新
	client.Model(&Order{}).
		Where("user_id = ?", 1001).
		Update("status", "completed")

	// 删除
	client.Where("user_id = ? AND id = ?", 1001, 1).Delete(&Order{})
}

// Example_shardingWithMasterSlave 分片 + 主从
func Example_shardingWithMasterSlave() {
	// ProxySQL 处理主从读写分离
	// go-gorm/sharding 处理分库分表
	// 两者完美配合

	cfg := &proxy.Config{
		Mode: "sharding",
		ProxySQL: &proxy.ProxySQLConfig{
			Addr:     "127.0.0.1:6033", // ProxySQL 自动主从路由
			Username: "root",
			Password: "root123",
			Database: "testdb",
		},
		Sharding: &proxy.ShardingConfig{
			Enable:     true,
			ShardCount: 8,
			Algorithm:  "mod",
			Tables: []*proxy.ShardingTable{
				{
					TableName:          "orders",
					ShardingKey:        "user_id",
					ShardCount:         8,
					GeneratePrimaryKey: true,
				},
				{
					TableName:          "order_items",
					ShardingKey:        "order_id",
					ShardCount:         8,
					GeneratePrimaryKey: true,
				},
			},
		},
		MaxOpenConns: 200,
		LogLevel:     "warn",
	}

	client, _ := proxy.NewClient(cfg)
	defer client.Close()

	// 写操作 → 主库 + 分片路由
	client.Create(&Order{UserID: 1001, Amount: 99.99})

	// 读操作 → 从库 + 分片路由
	var orders []Order
	client.Where("user_id = ?", 1001).Find(&orders)

	// 事务 → 主库 + 分片路由
	client.Transaction(func(tx *gorm.DB) error {
		tx.Create(&Order{UserID: 1001, Amount: 199.99})
		tx.Where("user_id = ?", 1001).Find(&orders)
		return nil
	})
}
