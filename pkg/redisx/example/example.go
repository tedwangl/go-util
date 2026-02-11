package main

import (
	"context"
	"fmt"
	"github.com/tedwangl/go-util/pkg/redisx/cache"
	"github.com/tedwangl/go-util/pkg/redisx/lock"
	"time"

	"github.com/tedwangl/go-util/pkg/redisx/client"
	"github.com/tedwangl/go-util/pkg/redisx/config"
)

func main() {
	// 示例1: 单节点模式
	singleExample()

	// 示例2: 哨兵模式
	// sentinelExample()

	// 示例3: 集群模式
	// clusterExample()

	// 示例4: 多主多从模式
	// multiMasterExample()
}

// 单节点模式示例
func singleExample() {
	fmt.Println("=== 单节点模式示例 ===")

	// 1. 创建配置
	cfg := &config.Config{
		Mode:     "single",
		Password: "",
		DB:       0,

		Single: &config.SingleConfig{
			Addr: "localhost:6379",
		},
	}

	// 2. 创建客户端
	cli, err := client.NewClient(cfg)
	if err != nil {
		fmt.Printf("创建客户端失败: %v\n", err)
		return
	}

	// 3. 测试基础操作
	ctx := context.Background()

	// 设置键值
	cmd := cli.Set(ctx, "test:key", "test:value", 10*time.Second)
	if err := cmd.Err(); err != nil {
		fmt.Printf("设置键值失败: %v\n", err)
		return
	}
	fmt.Println("设置键值成功")

	// 获取键值
	getCmd, err := cli.Get(ctx, "test:key")
	if err != nil {
		fmt.Printf("获取键值失败: %v\n", err)
		return
	}
	value, err := getCmd.Result()
	if err != nil {
		fmt.Printf("获取键值结果失败: %v\n", err)
		return
	}
	fmt.Printf("获取键值成功: %s\n", value)

	// 4. 测试缓存模块
	serverCache := cache.NewServerCache(cli, "server")

	// 设置服务器配置
	err = serverCache.SetConfig(ctx, "max_connections", "1000", 24*time.Hour)
	if err != nil {
		fmt.Printf("设置配置失败: %v\n", err)
		return
	}
	fmt.Println("设置配置成功")

	// 获取服务器配置
	configValue, err := serverCache.GetConfig(ctx, "max_connections")
	if err != nil {
		fmt.Printf("获取配置失败: %v\n", err)
		return
	}
	fmt.Printf("获取配置成功: %s\n", configValue)

	userCache := cache.NewUserCache(cli, "user")

	// 设置用户信息
	userInfo := map[string]interface{}{
		"id":   "123",
		"name": "张三",
		"age":  30,
	}
	err = userCache.SetUserInfo(ctx, "123", userInfo, 24*time.Hour)
	if err != nil {
		fmt.Printf("设置用户信息失败: %v\n", err)
		return
	}
	fmt.Println("设置用户信息成功")

	// 获取用户信息
	info, err := userCache.GetUserInfo(ctx, "123")
	if err != nil {
		fmt.Printf("获取用户信息失败: %v\n", err)
		return
	}
	fmt.Printf("获取用户信息成功: %v\n", info)

	// 5. 测试锁模块
	singleLock := lock.NewSingleLock(cli, "test:lock", nil)

	// 获取锁
	err = singleLock.Acquire(ctx)
	if err != nil {
		fmt.Printf("获取锁失败: %v\n", err)
		return
	}
	fmt.Println("获取锁成功")

	// 检查锁状态
	locked, err := singleLock.IsLocked(ctx)
	if err != nil {
		fmt.Printf("检查锁状态失败: %v\n", err)
		return
	}
	fmt.Printf("锁状态: %v\n", locked)

	// 模拟业务操作
	time.Sleep(2 * time.Second)

	// 释放锁
	err = singleLock.Release(ctx)
	if err != nil {
		fmt.Printf("释放锁失败: %v\n", err)
		return
	}
	fmt.Println("释放锁成功")

	// 再次检查锁状态
	locked, err = singleLock.IsLocked(ctx)
	if err != nil {
		fmt.Printf("检查锁状态失败: %v\n", err)
		return
	}
	fmt.Printf("锁状态: %v\n", locked)

	fmt.Println("单节点模式示例完成")
}

// 哨兵模式示例
func sentinelExample() {
	fmt.Println("=== 哨兵模式示例 ===")

	// 1. 创建配置
	cfg := &config.Config{
		Mode:     "sentinel",
		Password: "",
		DB:       0,

		Sentinel: &config.SentinelConfig{
			MasterName:    "mymaster",
			SentinelAddrs: []string{"localhost:26379", "localhost:26380", "localhost:26381"},
		},
	}

	// 2. 创建客户端
	cli, err := client.NewClient(cfg)
	if err != nil {
		fmt.Printf("创建客户端失败: %v\n", err)
		return
	}

	// 3. 测试操作
	ctx := context.Background()

	// 设置键值
	cmd := cli.Set(ctx, "sentinel:key", "sentinel:value", 10*time.Second)
	if err := cmd.Err(); err != nil {
		fmt.Printf("设置键值失败: %v\n", err)
		return
	}
	fmt.Println("设置键值成功")

	// 获取键值
	getCmd, err := cli.Get(ctx, "sentinel:key")
	if err != nil {
		fmt.Printf("获取键值失败: %v\n", err)
		return
	}
	value, err := getCmd.Result()
	if err != nil {
		fmt.Printf("获取键值结果失败: %v\n", err)
		return
	}
	fmt.Printf("获取键值成功: %s\n", value)

	fmt.Println("哨兵模式示例完成")
}

// 集群模式示例
func clusterExample() {
	fmt.Println("=== 集群模式示例 ===")

	// 1. 创建配置
	cfg := &config.Config{
		Mode:     "cluster",
		Password: "",

		Cluster: &config.ClusterConfig{
			Addrs: []string{"localhost:7000", "localhost:7001", "localhost:7002", "localhost:7003", "localhost:7004", "localhost:7005"},
		},
	}

	// 2. 创建客户端
	cli, err := client.NewClient(cfg)
	if err != nil {
		fmt.Printf("创建客户端失败: %v\n", err)
		return
	}

	// 3. 测试操作
	ctx := context.Background()

	// 设置键值
	cmd := cli.Set(ctx, "cluster:key", "cluster:value", 10*time.Second)
	if err := cmd.Err(); err != nil {
		fmt.Printf("设置键值失败: %v\n", err)
		return
	}
	fmt.Println("设置键值成功")

	// 获取键值
	getCmd, err := cli.Get(ctx, "cluster:key")
	if err != nil {
		fmt.Printf("获取键值失败: %v\n", err)
		return
	}
	value, err := getCmd.Result()
	if err != nil {
		fmt.Printf("获取键值结果失败: %v\n", err)
		return
	}
	fmt.Printf("获取键值成功: %s\n", value)

	fmt.Println("集群模式示例完成")
}

// 多主多从模式示例
func multiMasterExample() {
	fmt.Println("=== 多主多从模式示例 ===")

	// 1. 创建配置
	cfg := &config.Config{
		Mode:     "multi-master",
		Password: "",
		DB:       0,

		MultiMaster: &config.MultiMasterConfig{
			Masters: []config.MasterConfig{
				{
					Addr:   "localhost:6379",
					Slaves: []string{"localhost:6381"},
				},
				{
					Addr:   "localhost:6380",
					Slaves: []string{"localhost:6382"},
				},
			},
		},
	}

	// 2. 创建客户端
	cli, err := client.NewClient(cfg)
	if err != nil {
		fmt.Printf("创建客户端失败: %v\n", err)
		return
	}

	// 3. 测试操作
	ctx := context.Background()

	// 设置键值
	cmd := cli.Set(ctx, "multi:key", "multi:value", 10*time.Second)
	if err := cmd.Err(); err != nil {
		fmt.Printf("设置键值失败: %v\n", err)
		return
	}
	fmt.Println("设置键值成功")

	// 获取键值
	getCmd, err := cli.Get(ctx, "multi:key")
	if err != nil {
		fmt.Printf("获取键值失败: %v\n", err)
		return
	}
	value, err := getCmd.Result()
	if err != nil {
		fmt.Printf("获取键值结果失败: %v\n", err)
		return
	}
	fmt.Printf("获取键值成功: %s\n", value)

	fmt.Println("多主多从模式示例完成")
}
