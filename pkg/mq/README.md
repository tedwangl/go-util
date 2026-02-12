# Watermill MQ 封装

基于 Watermill 的统一消息队列封装，支持 Kafka、NATS、RabbitMQ。

## 目录结构

```
pkg/mq/
├── config.go           # 配置定义（含枚举类型）
├── factory.go          # 工厂主入口（含配置验证）
├── kafka.go            # Kafka 实现
├── nats.go             # NATS 实现
├── rabbitmq.go         # RabbitMQ 实现
├── publisher.go        # 发布者（含 TraceID 传递）
├── subscriber.go       # 订阅者
├── router.go           # 路由器
├── ack_handler.go      # ACK 处理和重试
├── dead_letter.go      # 死信队列
├── delay.go            # 延迟队列
├── retry.go            # 重试中间件
└── metrics.go          # 指标监控
```

## 核心组件

### 1. Factory（工厂）
统一的组件创建入口，支持创建：
- Publisher（发布者）
- Subscriber（订阅者）
- Router（路由器）
- DelayQueue（延迟队列）
- DeadLetterQueue（死信队列）
- RetryMiddleware（重试中间件）
- AckHandler（ACK 处理器）
- ValidateConfig（配置验证）
- Close（资源关闭）

### 2. Publisher（发布者）
- `Publish()` - 发布单条消息
- `PublishWithMetadata()` - 发布带元数据的消息
- `PublishBatch()` - 批量发布消息
- `PublishBatchWithMetadata()` - 批量发布带元数据的消息

### 3. Subscriber（订阅者）
- `Subscribe()` - 订阅消息（单条处理）
- `SubscribeBatch()` - 批量订阅消息

### 4. Router（路由器）
- `AddHandler()` - 添加主题处理器
- `Run()` - 运行指定主题
- `RunAll()` - 运行所有主题

### 5. AckHandler（ACK 处理器）
- 自动重试失败的消息
- 超过重试次数自动发送到死信队列
- 避免 Kafka offset 卡住和 NATS 无限重试

### 6. DeadLetterQueue（死信队列）
- `Subscribe()` - 订阅死信队列
- `Republish()` - 重新发布死信消息
- `Monitor()` - 监控死信队列
- `AnalyzeFailures()` - 分析失败原因
- `RetryWithPolicy()` - 使用策略重试

### 7. DelayQueue（延迟队列）
- `PublishDelayed()` - 发布延迟消息
- `StartProcessor()` - 启动延迟消息处理器

### 8. RetryMiddleware（重试中间件）
- 支持指数退避重试
- 超过重试次数发送到死信队列

### 9. MetricsMiddleware（指标中间件）
- 发布/消费计数
- 错误率统计
- 平均处理时间
- ACK/NACK 统计

## 快速开始

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/ThreeDotsLabs/watermill"
    "github.com/ThreeDotsLabs/watermill/message"
    "your-project/pkg/mq"
)

func main() {
    // 1. 创建配置（使用枚举类型）
    config := mq.Config{
        Broker: mq.BrokerConfig{
            Type: mq.BrokerTypeNATS,
            Addr: "nats://localhost:4222",
        },
    }
    
    // 2. 创建工厂
    logger := watermill.NewStdLogger(false, false)
    factory := mq.NewFactory(config, logger)
    
    // 3. 验证配置
    if err := factory.ValidateConfig(); err != nil {
        log.Fatal(err)
    }
    
    // 4. 创建发布者和订阅者
    pub, _ := factory.CreatePublisher()
    sub, _ := factory.CreateSubscriber()
    defer pub.Close()
    defer sub.Close()
    
    // 5. 订阅消息
    ctx := context.Background()
    sub.Subscribe(ctx, "demo", func(ctx context.Context, msg *message.Message) error {
        log.Printf("收到: %s", msg.Payload)
        return nil
    })
    
    // 6. 发布消息（支持 TraceID 传递）
    ctx = context.WithValue(ctx, "trace_id", "trace-123")
    pub.Publish(ctx, "demo", []byte("Hello World"))
    
    time.Sleep(time.Second)
}
```

## 配置说明

### Kafka 配置
```go
config := mq.Config{
    Broker: mq.BrokerConfig{
        Type: mq.BrokerTypeKafka,
        Addr: "localhost:9092",
    },
    Kafka: mq.KafkaConfig{
        ConsumerGroup:    "my-group",
        FetchMaxMessages: 100,
        RequiredAcks:     1, // 0=不等待, 1=Leader确认, -1=所有副本确认
    },
}
```

### NATS 配置
```go
config := mq.Config{
    Broker: mq.BrokerConfig{
        Type: mq.BrokerTypeNATS,
        Addr: "nats://localhost:4222",
    },
    NATS: mq.NATSConfig{
        QueueGroupPrefix: "my-group",  // 负载均衡模式
        EnableJetStream:  false,       // 是否启用持久化
    },
}
```

### RabbitMQ 配置
```go
config := mq.Config{
    Broker: mq.BrokerConfig{
        Type: mq.BrokerTypeRabbitMQ,
        Addr: "amqp://guest:guest@localhost:5672/",
    },
    RabbitMQ: mq.RabbitMQConfig{
        ExchangeName:  "logs.direct",
        ExchangeType:  "direct",
        Durable:       true,  // 持久化
        PrefetchCount: 10,    // QoS
        DeliveryMode:  2,     // 消息持久化
    },
}
```

## 高级功能

### 1. 重试和死信队列
```go
// 创建 ACK 处理器
ackConfig := mq.DefaultAckConfig()
ackHandler, _ := factory.CreateAckHandler(ackConfig)

// 包装处理器
handler := ackHandler.WrapHandler(func(ctx context.Context, msg *message.Message) error {
    // 处理逻辑
    return nil
})

sub.Subscribe(ctx, "demo", handler)

// 订阅死信队列
dlq, _ := factory.CreateDeadLetterQueue()
dlq.Subscribe(ctx, "demo", func(ctx context.Context, msg *message.Message, reason string) error {
    log.Printf("死信消息: %s, 原因: %s", msg.Payload, reason)
    return nil
})
```

### 2. 批量发布
```go
payloads := [][]byte{
    []byte("msg1"),
    []byte("msg2"),
    []byte("msg3"),
}
pub.PublishBatch(ctx, "demo", payloads)
```

### 3. 批量订阅
```go
sub.SubscribeBatch(ctx, "demo", 10, time.Second, func(ctx context.Context, msgs []*message.Message) error {
    log.Printf("批次: %d 条消息", len(msgs))
    return nil
})
```

### 4. 延迟队列
```go
delayQueue, _ := factory.CreateDelayQueue()

// 发布延迟消息
delayQueue.PublishDelayed(ctx, "demo", []byte("delayed"), 5*time.Second, nil)

// 启动处理器
delayQueue.StartProcessor(ctx, []string{"demo"})
```

### 5. 路由器
```go
router, _ := factory.CreateRouter()

router.AddHandler("topic1", handler1)
router.AddHandler("topic2", handler2)

router.RunAll(ctx)
```

### 6. 指标监控
```go
// 创建指标中间件
metricsMiddleware := mq.NewMetricsMiddleware(logger)

// 包装 Publisher
metricsPub := metricsMiddleware.WrapPublisher(pub)
metricsPub.Publish(ctx, "demo", []byte("test"))

// 包装 Handler
handler := metricsMiddleware.WrapHandler(func(ctx context.Context, msg *message.Message) error {
    // 处理逻辑
    return nil
})
sub.Subscribe(ctx, "demo", handler)

// 获取指标
snapshot := metricsMiddleware.GetMetrics().GetSnapshot()
log.Printf("发布: %d, 消费: %d, 错误率: %.2f%%", 
    snapshot.PublishedCount, 
    snapshot.ConsumedCount, 
    snapshot.ErrorRate()*100)
```

### 7. TraceID 传递
```go
// 在 context 中设置 trace_id
ctx := context.WithValue(context.Background(), "trace_id", "trace-123")

// 发布消息时自动传递
pub.Publish(ctx, "demo", []byte("test"))

// 订阅时可以获取
sub.Subscribe(ctx, "demo", func(ctx context.Context, msg *message.Message) error {
    traceID := msg.Metadata.Get("trace_id")
    log.Printf("TraceID: %s", traceID)
    return nil
})
```

## MQ 特性对比

| 特性 | Kafka | NATS | RabbitMQ |
|------|-------|------|----------|
| 持久化 | ✅ 默认 | ⚠️ JetStream | ✅ 配置 |
| 消息顺序 | ✅ Partition | ❌ | ⚠️ Queue |
| ACK 机制 | Offset | 消息级 | 消息级 |
| 批量拉取 | ✅ | ❌ | ✅ |
| 延迟队列 | ❌ | ❌ | ✅ |
| 通配符 | ❌ | ✅ | ⚠️ Topic |
| 性能 | 高吞吐 | 低延迟 | 中等 |

## 注意事项

### Kafka
- ACK 是 Offset 级别，乱序 ACK 会导致消息丢失
- 建议按顺序处理消息
- 批量发布性能提升明显

### NATS
- Core NATS 无持久化，重启丢失消息
- JetStream 提供持久化，但性能略降
- 支持通配符订阅（`*` 和 `>`）
- 批量发布提升约 20-30%

### RabbitMQ
- 需要配置 Durable 和 DeliveryMode 实现持久化
- 支持原生延迟队列（TTL + DLX）
- Watermill 不支持原生批量 ACK

## 示例代码

完整示例请参考：
- `examples/mq/complete_demo.go` - 完整功能演示
- `examples/mq/nats/` - NATS 专项示例
- `examples/mq/kafka/` - Kafka 专项示例
- `examples/mq/ack_best_practice.go` - ACK 最佳实践
- `examples/mq/batch_publish_demo.go` - 批量发布
- `examples/mq/batch_ack_demo.go` - 批量订阅
- `examples/mq/delay_queue_demo.go` - 延迟队列
- `examples/mq/dead_letter_demo.go` - 死信队列
