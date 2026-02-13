# Temporal 工作流引擎

Temporal 是一个微服务编排平台，用于构建可靠的分布式应用程序。

## 快速开始

### 1. 启动 Temporal 服务

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f temporal

# 等待服务就绪（约 30 秒）
```

### 2. 访问 Web UI

- URL: http://localhost:8088
- 可以查看工作流执行历史、状态、事件等

### 3. 初始化 Go 模块

```bash
# 在 temporal 目录下
go mod init your-module
go mod tidy
```

### 4. 安装依赖

```bash
go get go.temporal.io/sdk@latest
```

### 5. 运行示例

```bash
# 终端 1: 启动 Worker
go run worker/main.go

# 终端 2: 启动工作流
go run starter/main.go
```

## 目录结构

```
temporal/
├── docker-compose.yml       # Docker 编排配置
├── workflows/               # 工作流定义
│   └── example_workflow.go
├── activities/              # 活动定义
│   └── example_activities.go
├── worker/                  # Worker 程序
│   └── main.go
├── starter/                 # 工作流启动器
│   └── main.go
└── config/                  # 配置文件
```

## 核心概念

### Workflow（工作流）
业务逻辑的编排，定义任务的执行顺序和依赖关系。

```go
func OrderWorkflow(ctx workflow.Context, input OrderInput) (*OrderResult, error) {
    // 步骤 1: 验证
    workflow.ExecuteActivity(ctx, ValidateOrder, input)
    
    // 步骤 2: 支付
    workflow.ExecuteActivity(ctx, ProcessPayment, input)
    
    // 步骤 3: 发货
    workflow.ExecuteActivity(ctx, ShipOrder, input)
    
    return &OrderResult{Status: "完成"}, nil
}
```

### Activity（活动）
实际执行的任务单元，可以是任何业务逻辑。

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

## 特性

### 1. 自动重试

```go
RetryPolicy: &workflow.RetryPolicy{
    InitialInterval:    time.Second,
    BackoffCoefficient: 2.0,
    MaximumInterval:    time.Minute,
    MaximumAttempts:    3,
}
```

### 2. 超时控制

```go
ActivityOptions{
    StartToCloseTimeout: 10 * time.Second,
    ScheduleToCloseTimeout: 1 * time.Minute,
}
```

### 3. 补偿操作（Saga 模式）

```go
err := workflow.ExecuteActivity(ctx, ProcessPayment, input).Get(ctx, &result)
if err != nil {
    // 补偿：退款
    workflow.ExecuteActivity(ctx, RefundPayment, input)
    return err
}
```

### 4. 信号和查询

```go
// 发送信号
workflow.GetSignalChannel(ctx, "approval").Receive(ctx, &approved)

// 查询状态
workflow.SetQueryHandler(ctx, "status", func() (string, error) {
    return currentStatus, nil
})
```

## 常用命令

### 查看工作流列表

```bash
docker-compose exec temporal-admin-tools tctl workflow list
```

### 查看工作流详情

```bash
docker-compose exec temporal-admin-tools tctl workflow show -w <workflow-id>
```

### 终止工作流

```bash
docker-compose exec temporal-admin-tools tctl workflow terminate -w <workflow-id>
```

### 查看任务队列

```bash
docker-compose exec temporal-admin-tools tctl taskqueue describe -t example-task-queue
```

## 使用场景

### 1. 订单处理流程
- 验证订单 → 处理支付 → 发货 → 发送通知
- 自动重试失败步骤
- 支付失败自动退款

### 2. 用户注册流程
- 创建账户 → 发送验证邮件 → 等待验证 → 激活账户
- 24 小时未验证自动清理

### 3. 数据同步任务
- 从源系统提取 → 转换 → 加载到目标系统
- 失败自动重试
- 记录完整执行历史

### 4. 审批流程
- 提交申请 → 等待审批 → 执行操作 → 通知结果
- 支持多级审批
- 超时自动处理

## 最佳实践

1. **工作流代码必须确定性**
   - 不要使用随机数、当前时间
   - 使用 `workflow.Now(ctx)` 代替 `time.Now()`
   - 使用 `workflow.Sleep()` 代替 `time.Sleep()`

2. **活动应该是幂等的**
   - 同样的输入多次执行应该产生同样的结果
   - 使用唯一 ID 避免重复操作

3. **合理设置超时**
   - 根据实际业务设置合理的超时时间
   - 避免无限等待

4. **使用版本控制**
   - 工作流代码变更时使用版本号
   - 保证旧版本工作流能继续执行

5. **监控和告警**
   - 监控工作流执行时间
   - 设置失败告警

## 与 Airflow 对比

| 特性 | Temporal | Airflow |
|------|----------|---------|
| 适用场景 | 微服务编排、业务流程 | 数据管道、批处理 |
| 状态管理 | 事件溯源 | 数据库 |
| 重试机制 | 自动、细粒度 | 配置化 |
| 长时间运行 | 支持（月/年） | 有限（天） |
| 编程语言 | Go/Python/Java/TS | Python |
| 学习曲线 | 较陡 | 中等 |

## 故障排查

### Worker 无法连接

```bash
# 检查 Temporal 服务状态
docker-compose ps

# 查看日志
docker-compose logs temporal
```

### 工作流卡住

- 检查 Worker 是否运行
- 查看活动是否超时
- 检查任务队列名称是否匹配

### 数据库连接失败

```bash
# 重启服务
docker-compose restart temporal
```

## 扩展阅读

- [官方文档](https://docs.temporal.io/)
- [Go SDK 文档](https://docs.temporal.io/develop/go)
- [示例代码](https://github.com/temporalio/samples-go)
- [最佳实践](https://docs.temporal.io/develop/go/best-practices)
