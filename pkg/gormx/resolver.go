package gormx

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// setupDBResolver 配置 DBResolver（主从读写分离 + 多数据库）
//
// 工作原理：
// 1. 为每个 DSN 创建独立的连接池（gorm.Dialector）
// 2. 通过 dbresolver.Register() 注册到 GORM
// 3. DBResolver 拦截所有 SQL，根据规则路由到对应连接池
//
// 连接池数量计算：
// - 单库：1 个（主库）
// - 主从：2 个（主库 + 从库）
// - 多数据库：数据库数 * 2（每个数据库的主从）
func (c *Client) setupDBResolver(cfg *Config) error {
	// 场景 1：单库（无主从，无多数据库）
	if !cfg.HasReplica() && !cfg.HasMultiDatabase() {
		return nil // 不需要配置 DBResolver
	}

	// 场景 2：主从读写分离（无多数据库）
	if cfg.HasReplica() && !cfg.HasMultiDatabase() {
		return c.setupReplica(cfg.replica)
	}

	// 场景 3：多数据库（可能带主从）
	if cfg.HasMultiDatabase() {
		return c.setupMultiDatabase(cfg.multiDB)
	}

	return nil
}

// setupReplica 配置主从读写分离
func (c *Client) setupReplica(replica *ReplicaConfig) error {
	resolverCfg := dbresolver.Config{
		Policy: dbresolver.RandomPolicy{}, // 随机负载均衡
	}

	// 创建从库连接池
	replicaDialector, err := c.createDialector(c.config.Driver, replica.ReplicaDSN)
	if err != nil {
		return fmt.Errorf("failed to create replica dialector: %w", err)
	}
	resolverCfg.Replicas = []gorm.Dialector{replicaDialector}

	// 注册到 DBResolver
	if err := c.DB.Use(dbresolver.Register(resolverCfg)); err != nil {
		return fmt.Errorf("failed to register replica: %w", err)
	}

	return nil
}

// setupMultiDatabase 配置多数据库路由
// 关键：DBResolver 插件只能 Use 一次，但可以链式调用多个 Register()
func (c *Client) setupMultiDatabase(multiDB *MultiDatabaseConfig) error {
	// 构建链式 Register 调用
	var plugin gorm.Plugin

	// 注册所有数据库配置（带表名路由）
	for i, db := range multiDB.Databases {
		if db.DSN == "" {
			return fmt.Errorf("database %s must have dsn", db.Name)
		}
		if len(db.Tables) == 0 {
			return fmt.Errorf("database %s must specify tables", db.Name)
		}

		dbCfg := dbresolver.Config{
			Policy: dbresolver.RandomPolicy{},
		}

		// 第一个数据库的主库已作为主连接，不需要再注册 Sources
		if i == 0 {
			// 第一个数据库：只配置从库（如果有）
			if db.ReplicaDSN != "" {
				replica, err := c.createDialector(c.config.Driver, db.ReplicaDSN)
				if err != nil {
					return fmt.Errorf("failed to create database %s replica: %w", db.Name, err)
				}
				dbCfg.Replicas = []gorm.Dialector{replica}

				// 构建表名列表
				tables := make([]interface{}, len(db.Tables))
				for j, table := range db.Tables {
					tables[j] = table
				}

				if plugin == nil {
					plugin = dbresolver.Register(dbCfg, tables...)
				} else {
					plugin = plugin.(*dbresolver.DBResolver).Register(dbCfg, tables...)
				}
			}
			continue
		}

		// 其他数据库：正常注册主库和从库
		source, err := c.createDialector(c.config.Driver, db.DSN)
		if err != nil {
			return fmt.Errorf("failed to create database %s source: %w", db.Name, err)
		}
		dbCfg.Sources = []gorm.Dialector{source}

		if db.ReplicaDSN != "" {
			replica, err := c.createDialector(c.config.Driver, db.ReplicaDSN)
			if err != nil {
				return fmt.Errorf("failed to create database %s replica: %w", db.Name, err)
			}
			dbCfg.Replicas = []gorm.Dialector{replica}
		}

		// 构建表名列表
		tables := make([]interface{}, len(db.Tables))
		for j, table := range db.Tables {
			tables[j] = table
		}

		if plugin == nil {
			plugin = dbresolver.Register(dbCfg, tables...)
		} else {
			plugin = plugin.(*dbresolver.DBResolver).Register(dbCfg, tables...)
		}
	}

	// 一次性 Use 整个 DBResolver 插件
	if plugin != nil {
		if err := c.DB.Use(plugin); err != nil {
			return fmt.Errorf("failed to use dbresolver: %w", err)
		}
	}

	return nil
}

// createDialector 创建单个 Dialector
func (c *Client) createDialector(driver, dsn string) (gorm.Dialector, error) {
	switch driver {
	case "mysql":
		return c.createMySQLDialector(dsn), nil
	case "postgres":
		return c.createPostgresDialector(dsn), nil
	case "sqlite":
		return c.createSQLiteDialector(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}
}
