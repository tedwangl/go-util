package mq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type (
	// DelayedMessage 延迟消息结构
	DelayedMessage struct {
		TargetTopic string            `json:"target_topic"`
		Payload     []byte            `json:"payload"`
		Metadata    map[string]string `json:"metadata"`
		DelayUntil  time.Time         `json:"delay_until"`
	}

	// DelayQueue 延迟队列
	// 注意：不同 MQ 的支持情况：
	// - RabbitMQ: 原生支持（通过 TTL + Dead Letter Exchange）
	// - Kafka: 不支持，需要应用层实现
	// - NATS: 不支持，需要应用层实现
	DelayQueue struct {
		publisher  *Publisher
		subscriber *Subscriber
		logger     watermill.LoggerAdapter
	}
)

// NewDelayQueue 创建延迟队列
func NewDelayQueue(publisher *Publisher, subscriber *Subscriber, logger watermill.LoggerAdapter) *DelayQueue {
	return &DelayQueue{
		publisher:  publisher,
		subscriber: subscriber,
		logger:     logger,
	}
}

// PublishDelayed 发布延迟消息
// delay: 延迟时间
func (dq *DelayQueue) PublishDelayed(ctx context.Context, topic string, payload []byte, delay time.Duration, metadata map[string]string) error {
	delayedMsg := DelayedMessage{
		TargetTopic: topic,
		Payload:     payload,
		Metadata:    metadata,
		DelayUntil:  time.Now().Add(delay),
	}

	// 序列化延迟消息
	data, err := json.Marshal(delayedMsg)
	if err != nil {
		return err
	}

	// 发送到延迟队列
	delayTopic := topic + ".delayed"
	return dq.publisher.Publish(ctx, delayTopic, data)
}

// StartProcessor 启动延迟消息处理器
// 定期检查延迟队列，将到期的消息发送到目标队列
func (dq *DelayQueue) StartProcessor(ctx context.Context, topics []string) error {
	for _, topic := range topics {
		delayTopic := topic + ".delayed"

		handler := func(ctx context.Context, msg *message.Message) error {
			// 反序列化延迟消息
			var delayedMsg DelayedMessage
			if err := json.Unmarshal(msg.Payload, &delayedMsg); err != nil {
				dq.logger.Error("failed to unmarshal delayed message", err, watermill.LogFields{
					"message_uuid": msg.UUID,
				})
				return nil // ACK，避免重复处理错误消息
			}

			// 检查是否到期
			now := time.Now()
			if now.Before(delayedMsg.DelayUntil) {
				// 未到期，等待后重新投递
				waitTime := delayedMsg.DelayUntil.Sub(now)
				dq.logger.Debug("message not ready, waiting", watermill.LogFields{
					"message_uuid": msg.UUID,
					"wait_time":    waitTime,
				})
				time.Sleep(waitTime)
			}

			// 到期，发送到目标队列
			dq.logger.Info("publishing delayed message to target topic", watermill.LogFields{
				"message_uuid": msg.UUID,
				"target_topic": delayedMsg.TargetTopic,
			})

			return dq.publisher.PublishWithMetadata(
				ctx,
				delayedMsg.TargetTopic,
				delayedMsg.Payload,
				delayedMsg.Metadata,
			)
		}

		if err := dq.subscriber.Subscribe(ctx, delayTopic, handler); err != nil {
			return err
		}
	}

	return nil
}
