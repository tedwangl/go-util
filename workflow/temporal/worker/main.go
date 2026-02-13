package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/tedwangl/go-util/workflow/temporal/activities"
	"github.com/tedwangl/go-util/workflow/temporal/workflows"
)

func main() {
	// 创建 Temporal 客户端
	c, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		log.Fatalln("无法创建 Temporal 客户端", err)
	}
	defer c.Close()

	// 创建 Worker
	w := worker.New(c, "example-task-queue", worker.Options{})

	// 注册工作流
	w.RegisterWorkflow(workflows.OrderWorkflow)
	w.RegisterWorkflow(workflows.SimpleWorkflow)

	// 注册活动
	act := &activities.Activities{}
	w.RegisterActivity(act.SayHello)
	w.RegisterActivity(act.ValidateOrder)
	w.RegisterActivity(act.ProcessPayment)
	w.RegisterActivity(act.ShipOrder)
	w.RegisterActivity(act.SendNotification)
	w.RegisterActivity(act.CancelOrder)
	w.RegisterActivity(act.RefundPayment)

	// 启动 Worker
	log.Println("Worker 启动中...")
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("无法启动 Worker", err)
	}
}
