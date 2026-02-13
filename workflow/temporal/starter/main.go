package main

import (
	"context"
	"log"

	"go.temporal.io/sdk/client"

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

	// 示例 1: 启动简单工作流
	runSimpleWorkflow(c)

	// 示例 2: 启动订单工作流
	runOrderWorkflow(c)
}

func runSimpleWorkflow(c client.Client) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        "simple-workflow-1",
		TaskQueue: "example-task-queue",
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, workflows.SimpleWorkflow, "World")
	if err != nil {
		log.Fatalln("无法启动工作流", err)
	}

	log.Println("启动工作流", "WorkflowID", we.GetID(), "RunID", we.GetRunID())

	// 等待工作流完成
	var result string
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("工作流执行失败", err)
	}

	log.Println("工作流结果:", result)
}

func runOrderWorkflow(c client.Client) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        "order-workflow-001",
		TaskQueue: "example-task-queue",
	}

	input := workflows.OrderWorkflowInput{
		OrderID:    "ORD-12345",
		CustomerID: "CUST-001",
		Amount:     99.99,
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, workflows.OrderWorkflow, input)
	if err != nil {
		log.Fatalln("无法启动订单工作流", err)
	}

	log.Println("启动订单工作流", "WorkflowID", we.GetID(), "RunID", we.GetRunID())

	// 等待工作流完成
	var result workflows.OrderWorkflowResult
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("订单工作流执行失败", err)
	}

	log.Printf("订单工作流结果: OrderID=%s, Status=%s, Timestamp=%s\n",
		result.OrderID, result.Status, result.Timestamp)
}
