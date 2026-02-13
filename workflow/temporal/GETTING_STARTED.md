# Temporal 快速上手指南

## ✅ 服务已启动

Temporal 服务已经成功启动并运行！

### 访问地址

- **Web UI**: http://localhost:8088
- **gRPC 端口**: localhost:7233
- **PostgreSQL**: localhost:5433

## 📋 已完成的工作

1. ✅ Temporal Server 运行中
2. ✅ PostgreSQL 数据库运行中
3. ✅ Web UI 可访问
4. ✅ Go SDK 依赖已安装

## 🚀 下一步：运行示例

### 步骤 1: 启动 Worker

打开一个新终端：

```bash
cd workflow/temporal
go run worker/main.go
```

你会看到：
```
Worker 启动中...
```

### 步骤 2: 运行工作流

打开另一个终端：

```bash
cd workflow/temporal
go run starter/main.go
```

你会看到工作流执行结果。

### 步骤 3: 在 Web UI 查看

访问 http://localhost:8088，你可以看到：
- 工作流执行历史
- 每个步骤的详细信息
- 事件时间线
- 输入输出数据

## 📚 示例说明

### 简单工作流 (SimpleWorkflow)

最基础的示例，演示如何：
- 定义工作流
- 执行活动
- 返回结果

### 订单工作流 (OrderWorkflow)

复杂业务流程示例，演示：
- 多步骤编排
- 错误处理
- 补偿操作（Saga 模式）
- 自动重试

## 🔧 常用命令

### 查看运行中的容器

```bash
docker-compose ps
```

### 查看日志

```bash
docker-compose logs -f temporal
```

### 停止服务

```bash
docker-compose down
```

### 清理所有数据

```bash
docker-compose down -v
```

## 💡 核心概念

### Workflow（工作流）
业务逻辑的编排，定义任务执行顺序。

```go
func OrderWorkflow(ctx workflow.Context, input OrderInput) error {
    // 步骤 1
    workflow.ExecuteActivity(ctx, ValidateOrder, input)
    
    // 步骤 2
    workflow.ExecuteActivity(ctx, ProcessPayment, input)
    
    // 步骤 3
    workflow.ExecuteActivity(ctx, ShipOrder, input)
    
    return nil
}
```

### Activity（活动）
实际执行的任务单元。

```go
func ProcessPayment(ctx context.Context, input OrderInput) (string, error) {
    // 调用支付网关
    paymentID := callPaymentGateway(input)
    return paymentID, nil
}
```

### Worker（工作者）
执行工作流和活动的进程。

```go
w := worker.New(client, "task-queue", worker.Options{})
w.RegisterWorkflow(OrderWorkflow)
w.RegisterActivity(ProcessPayment)
w.Run(worker.InterruptCh())
```

## 🎯 实际应用场景

### 1. 订单处理
```
验证订单 → 处理支付 → 发货 → 发送通知
```

### 2. 用户注册
```
创建账户 → 发送验证邮件 → 等待验证 → 激活账户
```

### 3. 数据同步
```
提取数据 → 转换 → 加载 → 验证
```

## 🐛 故障排查

### Worker 无法连接

检查 Temporal 服务是否运行：
```bash
docker-compose ps
```

### 工作流卡住

1. 检查 Worker 是否运行
2. 查看 Web UI 中的错误信息
3. 检查任务队列名称是否匹配

### 端口冲突

如果端口被占用，修改 `docker-compose.yml` 中的端口映射。

## 📖 学习资源

- [官方文档](https://docs.temporal.io/)
- [Go SDK 文档](https://docs.temporal.io/develop/go)
- [示例代码](https://github.com/temporalio/samples-go)
- [最佳实践](https://docs.temporal.io/develop/go/best-practices)

## 🎉 恭喜！

你已经成功启动了 Temporal 工作流引擎。现在可以开始构建可靠的分布式应用了！
