# MQ Client 使用指南

## 简介

`Client` 是对 Watermill MQ 的高层封装，提供最简洁的 API，自动处理重试、死信队列等常见场景。

## 快速开始

### 1. 创建客户端

```go
import "github.com/tedwangl/go-util/pkg/mq"

// 配置
config := mq.Config{
    Broker: mq.BrokerConfig{
        Type: mq.BrokerTypeNATS,
        Addr: "nats://localhost:4222",
    },
}

// 创建客户端（一步到位）
client, err := mq.NewClient(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### 2. 发布消息

```go
ctx := context.Background()

// 简单发布
client.Publish(ctx, "topic", []byte("Hello"))

// 批量发布
payloads := [][]byte{
    []byte("Message 1"),
    []byte("Message 2"),
}
client.PublishBatch(ctx, "topic", payloads)

// 带元数据
metadata := map[string]string{"user_id": "123"}
client.PublishWithMetadata(ctx, "topic", []byte("data"), metadata)
```

### 3. 订阅消息

```go
// 自动带重试和死信队列
client.Subscribe(ctx, "topic", func(ctx context.Context, msg *message.Message) error {
    fmt.Printf("收到: %s\n", string(msg.Payload))
    return nil // 返回 nil = 成功，返回 error = 失败（会重试）
})
```

## 核心特性

### 自动重试

订阅消息时，默认自动重试 3 次：

```go
client.Subscribe(ctx, "orders", func(ctx context.Context, msg *message.Message) error {
    // 处理失败会自动重试
    if err := processOrder(msg); err != nil {
        return err // 触发重试
    }
    return nil
})
```

### 死信队列

重试失败后自动发送到死信队列：

```go
// 订阅死信队列
client.SubscribeDeadLetter(ctx, "orders", func(ctx context.Context, msg *message.Message, reason string) error {
    log.Printf("死信消息: %s, 原因: %s", string(msg.Payload), reason)
    
    // 可以选择重新发布
    // client.RepublishDeadLetter(ctx, msg, "orders")
    
    return nil
})
```

### 批量处理

```go
// 批量订阅：每 10 条或 2 秒处理一次
client.SubscribeBatch(ctx, "events", 10, 2*time.Second,
    func(ctx context.Context, msgs []*message.Message) error {
        for _, msg := range msgs {
            fmt.Printf("处理: %s\n", string(msg.Payload))
        }
        return nil
    })
```

## 自定义配置

### ACK 配置

```go
ackConfig := mq.AckConfig{
    MaxRetries:       5,                    // 最大重试次数
    RetryDelay:       time.Second,          // 重试间隔
    EnableDeadLetter: true,                 // 启用死信队列
    DeadLetterSuffix: ".dead-letter",       // 死信队列后缀
}

client, err := mq.NewClient(config, mq.WithAckConfig(ackConfig))
```

### 自定义日志

```go
logger := watermill.NewStdLogger(true, true)
client, err := mq.NewClient(config, mq.WithLogger(logger))
```

## 不同 MQ 配置

### Kafka

```go
config := mq.Config{
    Broker: mq.BrokerConfig{
        Type:  mq.BrokerTypeKafka,
        Addrs: []string{"localhost:9092"},
    },
    Kafka: mq.KafkaConfig{
        ConsumerGroup:     "my-group",
        RequiredAcks:      1,              // 等待 Leader 确认
        FetchMaxMessages:  100,            // 每次拉取 100 条
        PartitionStrategy: mq.PartitionStrategyHash, // Hash 分区
    },
}
```

### NATS

```go
config := mq.Config{
    Broker: mq.BrokerConfig{
        Type: mq.BrokerTypeNATS,
        Addr: "nats://localhost:4222",
    },
    NATS: mq.NATSConfig{
        QueueGroupPrefix: "my-service",
        EnableJetStream:  false,           // Core NATS（内存）
    },
}
```

### RabbitMQ

```go
config := mq.Config{
    Broker: mq.BrokerConfig{
        Type: mq.BrokerTypeRabbitMQ,
        Addr: "amqp://guest:guest@localhost:5672/",
    },
    RabbitMQ: mq.RabbitMQConfig{
        Durable:       true,               // 持久化
        DeliveryMode:  2,                  // 消息持久化
        PrefetchCount: 10,                 // QoS
    },
}
```

## 高级用法

### 原始订阅（不带重试）

```go
// 不需要重试时使用
client.SubscribeRaw(ctx, "topic", func(ctx context.Context, msg *message.Message) error {
    // 直接处理，不会重试
    return nil
})
```

### 访问底层对象

```go
// 获取底层 Publisher（高级用户）
publisher := client.GetPublisher()

// 获取底层 Subscriber（高级用户）
subscriber := client.GetSubscriber()

// 获取底层 Factory（高级用户）
factory := client.GetFactory()
```

### Kafka 分区

```go
// 指定分区键（保证顺序）
client.PublishWithPartitionKey(ctx, "orders", []byte("data"), "user_123")
```

## 完整示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/tedwangl/go-util/pkg/mq"
    "github.com/ThreeDotsLabs/watermill/message"
)

func main() {
    // 1. 配置
    config := mq.Config{
        Broker: mq.BrokerConfig{
            Type: mq.BrokerTypeNATS,
            Addr: "nats://localhost:4222",
        },
        NATS: mq.NATSConfig{
            QueueGroupPrefix: "my-service",
        },
    }

    // 2. 创建客户端
    client, err := mq.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // 3. 订阅消息
    go client.Subscribe(ctx, "orders", func(ctx context.Context, msg *message.Message) error {
        fmt.Printf("处理订单: %s\n", string(msg.Payload))
        return nil
    })

    // 4. 订阅死信队列
    go client.SubscribeDeadLetter(ctx, "orders", func(ctx context.Context, msg *message.Message, reason string) error {
        fmt.Printf("死信: %s, 原因: %s\n", string(msg.Payload), reason)
        return nil
    })

    time.Sleep(500 * time.Millisecond)

    // 5. 发布消息
    client.Publish(ctx, "orders", []byte("订单 #1001"))

    // 等待处理
    time.Sleep(2 * time.Second)
}
```

## API 对比

### 旧方式（Factory）

```go
// 需要多步操作
factory := mq.NewFactory(config, logger)
publisher, _ := factory.CreatePublisher()
subscriber, _ := factory.CreateSubscriber()
ackHandler := mq.NewAckHandler(ackConfig, publisher, logger)
dlq, _ := factory.CreateDeadLetterQueue()

// 手动包装 handler
wrappedHandler := ackHandler.WrapHandler(myHandler)
subscriber.Subscribe(ctx, "topic", wrappedHandler)
```

### 新方式（Client）

```go
// 一步到位
client, _ := mq.NewClient(config)

// 直接使用
client.Subscribe(ctx, "topic", myHandler)
```

## 注意事项

1. **默认重试**：`Subscribe()` 默认带重试，如果不需要重试使用 `SubscribeRaw()`
2. **死信队列**：失败消息自动发送到 `{topic}.dead-letter`
3. **资源关闭**：记得调用 `client.Close()` 释放资源
4. **并发安全**：Client 是并发安全的，可以在多个 goroutine 中使用

## 性能建议

1. **批量发布**：大量消息使用 `PublishBatch()` 提升性能
2. **批量订阅**：高吞吐场景使用 `SubscribeBatch()` 减少处理开销
3. **Kafka 分区**：使用 `PublishWithPartitionKey()` 保证顺序性
4. **调整 Prefetch**：根据消息处理速度调整 `FetchMaxMessages` / `PrefetchCount`
