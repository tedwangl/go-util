package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// OrderWorkflowInput 订单工作流输入
type OrderWorkflowInput struct {
	OrderID    string
	CustomerID string
	Amount     float64
}

// OrderWorkflowResult 订单工作流结果
type OrderWorkflowResult struct {
	OrderID   string
	Status    string
	Timestamp time.Time
}

// OrderWorkflow 订单处理工作流示例
func OrderWorkflow(ctx workflow.Context, input OrderWorkflowInput) (*OrderWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("订单工作流开始", "OrderID", input.OrderID)

	// 配置活动选项
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 步骤 1: 验证订单
	var validateResult bool
	err := workflow.ExecuteActivity(ctx, "ValidateOrder", input).Get(ctx, &validateResult)
	if err != nil {
		logger.Error("订单验证失败", "error", err)
		return nil, err
	}
	if !validateResult {
		return &OrderWorkflowResult{
			OrderID:   input.OrderID,
			Status:    "验证失败",
			Timestamp: workflow.Now(ctx),
		}, nil
	}

	// 步骤 2: 处理支付
	var paymentResult string
	err = workflow.ExecuteActivity(ctx, "ProcessPayment", input).Get(ctx, &paymentResult)
	if err != nil {
		logger.Error("支付处理失败", "error", err)
		// 补偿操作：取消订单
		workflow.ExecuteActivity(ctx, "CancelOrder", input.OrderID)
		return nil, err
	}

	// 步骤 3: 发货
	var shipmentResult string
	err = workflow.ExecuteActivity(ctx, "ShipOrder", input).Get(ctx, &shipmentResult)
	if err != nil {
		logger.Error("发货失败", "error", err)
		// 补偿操作：退款
		workflow.ExecuteActivity(ctx, "RefundPayment", input.OrderID)
		return nil, err
	}

	// 步骤 4: 发送通知
	workflow.ExecuteActivity(ctx, "SendNotification", input.CustomerID, "订单已发货")

	logger.Info("订单工作流完成", "OrderID", input.OrderID)

	return &OrderWorkflowResult{
		OrderID:   input.OrderID,
		Status:    "已完成",
		Timestamp: workflow.Now(ctx),
	}, nil
}

// SimpleWorkflow 简单工作流示例
func SimpleWorkflow(ctx workflow.Context, name string) (string, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("工作流开始", "name", name)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result string
	err := workflow.ExecuteActivity(ctx, "SayHello", name).Get(ctx, &result)
	if err != nil {
		return "", err
	}

	return result, nil
}
