package gormx_test

import (
	"context"
	"time"

	"github.com/tedwangl/go-util/pkg/gormx"
	"gorm.io/gorm"
)

// User 示例模型
type User struct {
	ID        int64          `gorm:"primarykey"`
	Name      string         `gorm:"size:100;not null"`
	Email     string         `gorm:"size:100;uniqueIndex"`
	Status    string         `gorm:"size:20;default:active"`
	IsActive  bool           `gorm:"default:true"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// UserRepository 用户仓储
type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(client *gormx.Client) *UserRepository {
	return &UserRepository{db: client.DB}
}

// FindByID 根据 ID 查找用户
func (r *UserRepository) FindByID(id int64) (*User, error) {
	var user User
	err := r.db.First(&user, id).Error
	return &user, err
}

// FindByEmail 根据邮箱查找用户
func (r *UserRepository) FindByEmail(email string) (*User, error) {
	var user User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

// FindActiveUsers 查找激活用户（分页）
func (r *UserRepository) FindActiveUsers(page, pageSize int) (*gormx.PageResult, error) {
	var users []User
	return gormx.FindWithPage(
		r.db.Scopes(gormx.Active(), gormx.OrderByCreatedAt(true)),
		page,
		pageSize,
		&users,
	)
}

// SearchUsers 搜索用户
func (r *UserRepository) SearchUsers(keyword string, page, pageSize int) (*gormx.PageResult, error) {
	var users []User
	return gormx.FindWithPage(
		r.db.Scopes(
			gormx.Search("name", keyword),
			gormx.WithoutDeleted(),
			gormx.OrderByCreatedAt(true),
		),
		page,
		pageSize,
		&users,
	)
}

// Create 创建用户
func (r *UserRepository) Create(user *User) error {
	return r.db.Create(user).Error
}

// Update 更新用户
func (r *UserRepository) Update(id int64, updates map[string]any) error {
	return r.db.Model(&User{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 软删除用户
func (r *UserRepository) Delete(id int64) error {
	return gormx.SoftDelete(r.db, id, &User{})
}

// BatchCreate 批量创建用户
func (r *UserRepository) BatchCreate(users []User) error {
	return gormx.BatchCreate(r.db, users, 100)
}

// ExistsEmail 检查邮箱是否存在
func (r *UserRepository) ExistsEmail(email string) (bool, error) {
	return gormx.Exists(r.db.Model(&User{}).Where("email = ?", email))
}

// FindRecentUsers 查找最近注册的用户
func (r *UserRepository) FindRecentUsers(days int) ([]User, error) {
	var users []User
	start := time.Now().AddDate(0, 0, -days)
	err := r.db.Scopes(
		gormx.TimeRange("created_at", start, time.Time{}),
		gormx.OrderByCreatedAt(true),
	).Find(&users).Error
	return users, err
}

// TransferBalance 转账示例（事务）
func (r *UserRepository) TransferBalance(ctx context.Context, fromID, toID int64, amount float64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 扣除转出方余额
		if err := tx.Model(&User{}).Where("id = ?", fromID).
			Update("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
			return err
		}

		// 增加转入方余额
		if err := tx.Model(&User{}).Where("id = ?", toID).
			Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
			return err
		}

		return nil
	})
}

// Example_basicUsage 基本使用示例
func Example_basicUsage() {
	// 1. 创建客户端
	cfg := &gormx.Config{
		Driver:       "mysql",
		DSN:          "root:password@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local",
		MaxOpenConns: 100,
		MaxIdleConns: 10,
		LogLevel:     "info",
	}

	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	// 2. 自动迁移
	client.AutoMigrate(&User{})

	// 3. 创建仓储
	repo := NewUserRepository(client)

	// 4. 创建用户
	user := &User{
		Name:  "张三",
		Email: "zhangsan@example.com",
	}
	repo.Create(user)

	// 5. 查询用户
	repo.FindByEmail("zhangsan@example.com")

	// 6. 分页查询
	repo.FindActiveUsers(1, 10)

	// 7. 搜索用户
	repo.SearchUsers("张", 1, 10)

	// 8. 更新用户
	repo.Update(user.ID, map[string]any{
		"name":   "李四",
		"status": "inactive",
	})

	// 9. 删除用户
	repo.Delete(user.ID)
}

// Example_complexQuery 复杂查询示例
func Example_complexQuery() {
	cfg := gormx.DefaultConfig()
	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	var users []User

	// 复杂条件组合
	client.DB.Scopes(
		gormx.Active(),
		gormx.Status("premium"),
		gormx.TimeRange("created_at", time.Now().AddDate(0, -1, 0), time.Now()),
		gormx.Search("name", "张"),
		gormx.OrderByCreatedAt(true),
		gormx.Paginate(1, 20),
	).Find(&users)

	// 关联查询（直接用 GORM）
	client.DB.
		Preload("Orders").
		Preload("Profile").
		Where("status = ?", "active").
		Find(&users)

	// 聚合查询
	var count int64
	client.DB.Model(&User{}).
		Where("created_at > ?", time.Now().AddDate(0, -1, 0)).
		Count(&count)
}

// Example_transaction 事务示例
func Example_transaction() {
	cfg := gormx.DefaultConfig()
	client, _ := gormx.NewClient(cfg)
	defer client.Close()

	// 方式 1：使用封装的事务方法
	client.Transaction(func(tx *gorm.DB) error {
		// 创建用户
		user := &User{Name: "张三", Email: "zhangsan@example.com"}
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		// 创建订单
		// order := &Order{UserID: user.ID, Amount: 100}
		// if err := tx.Create(order).Error; err != nil {
		// 	return err
		// }

		return nil
	})

	// 方式 2：手动控制事务
	ctx := context.Background()
	tx := client.BeginTx(ctx)

	user := &User{Name: "李四", Email: "lisi@example.com"}
	if err := tx.Create(user).Error; err != nil {
		gormx.Rollback(tx)
		return
	}

	gormx.Commit(tx)
}
