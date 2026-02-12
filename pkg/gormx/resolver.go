package gormx

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// setupDBResolver 配置 DBResolver（主从读写分离 + 分库）
func (c *Client) setupDBResolver(cfg *Config) error {
	// 如果没有配置主从或分库，直接返回
	if len(cfg.Replicas) == 0 && len(cfg.Sources) == 0 && len(cfg.Shards) == 0 {
		return nil
	}

	// 1. 配置默认主从（全局）
	if len(cfg.Replicas) > 0 || len(cfg.Sources) > 0 {
		resolverCfg := dbresolver.Config{
			Policy: dbresolver.RandomPolicy{}, // 随机负载均衡
		}

		// 添加额外主库
		if len(cfg.Sources) > 0 {
			sources, err := c.createDialectors(cfg.Driver, cfg.Sources)
			if err != nil {
				return fmt.Errorf("failed to create source dialectors: %w", err)
			}
			resolverCfg.Sources = sources
		}

		// 添加从库
		if len(cfg.Replicas) > 0 {
			replicas, err := c.createDialectors(cfg.Driver, cfg.Replicas)
			if err != nil {
				return fmt.Errorf("failed to create replica dialectors: %w", err)
			}
			resolverCfg.Replicas = replicas
		}

		if err := c.DB.Use(dbresolver.Register(resolverCfg)); err != nil {
			return fmt.Errorf("failed to register dbresolver: %w", err)
		}
	}

	// 2. 配置分库
	for _, shard := range cfg.Shards {
		if len(shard.Sources) == 0 {
			return fmt.Errorf("shard %s must have at least one source", shard.Name)
		}

		shardCfg := dbresolver.Config{
			Policy: dbresolver.RandomPolicy{},
		}

		// 添加分片主库
		sources, err := c.createDialectors(cfg.Driver, shard.Sources)
		if err != nil {
			return fmt.Errorf("failed to create shard %s sources: %w", shard.Name, err)
		}
		shardCfg.Sources = sources

		// 添加分片从库
		if len(shard.Replicas) > 0 {
			replicas, err := c.createDialectors(cfg.Driver, shard.Replicas)
			if err != nil {
				return fmt.Errorf("failed to create shard %s replicas: %w", shard.Name, err)
			}
			shardCfg.Replicas = replicas
		}

		// 注册分片（按表名路由）
		if err := c.DB.Use(dbresolver.Register(shardCfg, shard.Tables...)); err != nil {
			return fmt.Errorf("failed to register shard %s: %w", shard.Name, err)
		}
	}

	return nil
}

// createDialectors 创建 Dialector 列表
func (c *Client) createDialectors(driver string, dsns []string) ([]gorm.Dialector, error) {
	dialectors := make([]gorm.Dialector, len(dsns))

	for i, dsn := range dsns {
		dialector, err := c.createDialector(driver, dsn)
		if err != nil {
			return nil, err
		}
		dialectors[i] = dialector
	}

	return dialectors, nil
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
