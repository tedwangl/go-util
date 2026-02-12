package mq

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Publisher struct {
	publisher message.Publisher
	logger    watermill.LoggerAdapter
}

func NewPublisher(pub message.Publisher, logger watermill.LoggerAdapter) *Publisher {
	return &Publisher{
		publisher: pub,
		logger:    logger,
	}
}

func (p *Publisher) Publish(ctx context.Context, topic string, payload []byte) error {
	msg := message.NewMessage(watermill.NewUUID(), payload)
	// 传递 context 中的 trace 信息
	if traceID := ctx.Value("trace_id"); traceID != nil {
		msg.Metadata.Set("trace_id", traceID.(string))
	}
	return p.publisher.Publish(topic, msg)
}

func (p *Publisher) PublishWithMetadata(ctx context.Context, topic string, payload []byte, metadata map[string]string) error {
	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata = metadata
	// 传递 context 中的 trace 信息
	if traceID := ctx.Value("trace_id"); traceID != nil {
		if msg.Metadata.Get("trace_id") == "" {
			msg.Metadata.Set("trace_id", traceID.(string))
		}
	}
	return p.publisher.Publish(topic, msg)
}

// PublishWithPartitionKey 发布消息并指定分区键（仅 Kafka 有效）
// partitionKey: 分区键，相同 Key 的消息会进入同一分区
func (p *Publisher) PublishWithPartitionKey(ctx context.Context, topic string, payload []byte, partitionKey string) error {
	msg := message.NewMessage(watermill.NewUUID(), payload)
	// Kafka 使用 metadata 中的 "partition_key" 作为分区键
	msg.Metadata.Set("partition_key", partitionKey)
	// 传递 context 中的 trace 信息
	if traceID := ctx.Value("trace_id"); traceID != nil {
		msg.Metadata.Set("trace_id", traceID.(string))
	}
	return p.publisher.Publish(topic, msg)
}

// PublishToPartition 发布消息到指定分区（仅 Kafka 有效，需要 PartitionStrategyManual）
// partition: 分区编号（从 0 开始）
func (p *Publisher) PublishToPartition(ctx context.Context, topic string, payload []byte, partition int32) error {
	msg := message.NewMessage(watermill.NewUUID(), payload)
	// Kafka 使用 metadata 中的 "partition" 指定分区
	msg.Metadata.Set("partition", string(rune(partition)))
	// 传递 context 中的 trace 信息
	if traceID := ctx.Value("trace_id"); traceID != nil {
		msg.Metadata.Set("trace_id", traceID.(string))
	}
	return p.publisher.Publish(topic, msg)
}

func (p *Publisher) Close() error {
	if closer, ok := p.publisher.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// PublishBatch 批量发布消息
// 注意：
// - Kafka: 支持真正的批量发送（性能优化）
// - NATS: 逐条发送（但速度很快）
// - RabbitMQ: 逐条发送
func (p *Publisher) PublishBatch(ctx context.Context, topic string, payloads [][]byte) error {
	if len(payloads) == 0 {
		return nil
	}

	// 构造消息批次
	messages := make([]*message.Message, len(payloads))
	for i, payload := range payloads {
		messages[i] = message.NewMessage(watermill.NewUUID(), payload)
	}

	// Watermill 会根据底层 MQ 自动优化批量发送
	// Kafka: 批量发送到同一个 partition
	// NATS/RabbitMQ: 逐条发送
	for _, msg := range messages {
		if err := p.publisher.Publish(topic, msg); err != nil {
			return fmt.Errorf("failed to publish message %s: %w", msg.UUID, err)
		}
	}

	return nil
}

// PublishBatchWithMetadata 批量发布带 metadata 的消息
func (p *Publisher) PublishBatchWithMetadata(ctx context.Context, topic string, items []BatchItem) error {
	if len(items) == 0 {
		return nil
	}

	for _, item := range items {
		msg := message.NewMessage(watermill.NewUUID(), item.Payload)
		msg.Metadata = item.Metadata

		if err := p.publisher.Publish(topic, msg); err != nil {
			return fmt.Errorf("failed to publish message %s: %w", msg.UUID, err)
		}
	}

	return nil
}

// BatchItem 批量发布项
type BatchItem struct {
	Payload  []byte
	Metadata map[string]string
}
