package activities

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
)

// Activities 活动集合
type Activities struct{}

// SayHello 简单的问候活动
func (a *Activities) SayHello(ctx context.Context, name string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("执行 SayHello 活动", "name", name)

	time.Sleep(1 * time.Second) // 模拟处理时间

	return fmt.Sprintf("Hello, %s!", name), nil
}

// ValidateOrder 验证订单
func (a *Activities) ValidateOrder(ctx context.Context, input interface{}) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("验证订单")

	// 模拟验证逻辑
	time.Sleep(500 * time.Millisecond)

	// 这里可以添加实际的验证逻辑
	// 例如：检查库存、验证客户信息等

	return true, nil
}

// ProcessPayment 处理支付
func (a *Activities) ProcessPayment(ctx context.Context, input interface{}) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("处理支付")

	// 模拟支付处理
	time.Sleep(2 * time.Second)

	// 这里可以调用支付网关 API
	paymentID := fmt.Sprintf("PAY-%d", time.Now().Unix())

	logger.Info("支付成功", "paymentID", paymentID)
	return paymentID, nil
}

// ShipOrder 发货
func (a *Activities) ShipOrder(ctx context.Context, input interface{}) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("处理发货")

	// 模拟发货处理
	time.Sleep(1 * time.Second)

	// 这里可以调用物流系统 API
	trackingNumber := fmt.Sprintf("TRACK-%d", time.Now().Unix())

	logger.Info("发货成功", "trackingNumber", trackingNumber)
	return trackingNumber, nil
}

// SendNotification 发送通知
func (a *Activities) SendNotification(ctx context.Context, customerID, message string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("发送通知", "customerID", customerID, "message", message)

	// 模拟发送通知
	time.Sleep(300 * time.Millisecond)

	// 这里可以调用邮件/短信服务

	return nil
}

// CancelOrder 取消订单（补偿操作）
func (a *Activities) CancelOrder(ctx context.Context, orderID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("取消订单", "orderID", orderID)

	time.Sleep(500 * time.Millisecond)

	return nil
}

// RefundPayment 退款（补偿操作）
func (a *Activities) RefundPayment(ctx context.Context, orderID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("处理退款", "orderID", orderID)

	time.Sleep(1 * time.Second)

	return nil
}
