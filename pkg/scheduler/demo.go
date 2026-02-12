package scheduler

import (
	"context"
	"fmt"
	"time"
)

func main() {
	fmt.Println("=== 简化版演示 ===\n")

	s := NewScheduler()

	// 推荐：使用 AddFunc（无需 context）
	s.AddFunc("@every 5s", "task1", func() error {
		fmt.Println("任务1: 每 5 秒执行")
		return nil
	})

	s.AddFunc("@every 10s", "task2", func() error {
		fmt.Println("任务2: 每 10 秒执行")
		// 模拟失败
		return fmt.Errorf("任务失败")
	})

	s.Start()

	time.Sleep(30 * time.Second)
	s.Stop()

	fmt.Println("演示完成")
}

func main1() {
	fmt.Println("=== 实际业务场景演示 ===\n")

	s := NewScheduler()

	// 1. 数据同步任务（每小时）
	s.AddJob("0 * * * *", "data-sync", func(ctx context.Context) error {
		fmt.Println("[数据同步] 开始同步...")
		time.Sleep(2 * time.Second) // 模拟耗时操作
		fmt.Println("[数据同步] 完成")
		return nil
	})

	// 2. 缓存清理（每天凌晨 3 点）
	s.AddJob("0 3 * * *", "cache-cleanup", func(ctx context.Context) error {
		fmt.Println("[缓存清理] 清理过期缓存...")
		return nil
	})

	// 3. 报表生成（每周一早上 8 点）
	s.AddJob("0 8 * * 1", "weekly-report", func(ctx context.Context) error {
		fmt.Println("[报表生成] 生成周报...")
		return nil
	})

	// 4. 健康检查（每 30 秒）
	s.AddJob("@every 30s", "health-check", func(ctx context.Context) error {
		fmt.Println("[健康检查] 检查服务状态...")
		// 模拟检查逻辑
		healthy := time.Now().Unix()%2 == 0
		if !healthy {
			return fmt.Errorf("服务异常")
		}
		return nil
	})

	// 5. 订单超时处理（每分钟）
	s.AddJob("*/1 * * * *", "order-timeout", func(ctx context.Context) error {
		fmt.Println("[订单超时] 处理超时订单...")
		// 查询超时订单并处理
		return nil
	})

	// 6. 消息队列消费监控（每 10 秒）
	s.AddJob("@every 10s", "mq-monitor", func(ctx context.Context) error {
		fmt.Println("[MQ 监控] 检查消息堆积...")
		return nil
	})

	s.Start()

	// 显示任务列表
	fmt.Println("已注册的任务:")
	for _, job := range s.ListJobs() {
		fmt.Printf("  - %-20s 下次执行: %s\n",
			job.Name,
			job.Next.Format("2006-01-02 15:04:05"))
	}

	// 运行 2 分钟
	fmt.Println("\n调度器运行中...")
	time.Sleep(2 * time.Minute)

	s.Stop()
	fmt.Println("演示完成")
}

func main2() {
	fmt.Println("=== 动态添加任务演示 ===\n")

	// 1. 标准格式（5 字段）
	s1 := NewScheduler()
	s1.Start() // 先启动

	fmt.Println("1. 标准格式（分钟级）")
	s1.AddJob("@every 5s", "task1", func(ctx context.Context) error {
		fmt.Println("  任务1: @every 5s (预定义)")
		return nil
	})

	// Start 后动态添加
	time.Sleep(2 * time.Second)
	fmt.Println("\n动态添加任务2...")
	s1.AddJob("@every 8s", "task2", func(ctx context.Context) error {
		fmt.Println("  任务2: @every 8s (动态添加)")
		return nil
	})

	time.Sleep(20 * time.Second)

	// 2. 秒级格式（6 字段）
	fmt.Println("\n2. 秒级格式")
	s2 := NewScheduler(WithSeconds())
	s2.Start()

	// 秒级 cron 表达式（6 字段）
	s2.AddJob("*/3 * * * * *", "task3", func(ctx context.Context) error {
		fmt.Println("  任务3: 每 3 秒 (6 字段)")
		return nil
	})

	// 预定义在秒级模式下也有效
	s2.AddJob("@every 7s", "task4", func(ctx context.Context) error {
		fmt.Println("  任务4: @every 7s (预定义)")
		return nil
	})

	time.Sleep(20 * time.Second)

	// 3. 预定义表达式（通用）
	fmt.Println("\n3. 预定义表达式")
	s3 := NewScheduler()
	s3.Start()

	s3.AddJob("@every 2s", "every-2s", func(ctx context.Context) error {
		fmt.Println("  @every 2s")
		return nil
	})

	s3.AddJob("@every 1m", "every-1m", func(ctx context.Context) error {
		fmt.Println("  @every 1m")
		return nil
	})

	// @daily, @hourly 等在实际场景中使用
	// s3.AddJob("@daily", "daily-task", func(ctx context.Context) error {
	// 	fmt.Println("  每天执行")
	// 	return nil
	// })

	time.Sleep(10 * time.Second)

	// 停止
	s1.Stop()
	s2.Stop()
	s3.Stop()

	fmt.Println("\n演示完成")
}
