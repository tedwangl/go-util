package mq

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// AckConfig ACK 处理配置
type AckConfig struct {
	// 最大重试次数（0 表示不重试）
	MaxRetries int
	// 重试间隔
	RetryDelay time.Duration
	// 是否启用死信队列
	EnableDeadLetter bool
	// 死信队列主题后缀
	DeadLetterSuffix string
}

// DefaultAckConfig 默认 ACK 配置
func DefaultAckConfig() AckConfig {
	return AckConfig{
		MaxRetries:       3,
		RetryDelay:       time.Second,
		EnableDeadLetter: true,
		DeadLetterSuffix: ".dead-letter",
	}
}

// AckHandler 带重试和死信队列的消息处理器
type AckHandler struct {
	config    AckConfig
	publisher *Publisher
	logger    watermill.LoggerAdapter
}

// NewAckHandler 创建 ACK 处理器
func NewAckHandler(config AckConfig, publisher *Publisher, logger watermill.LoggerAdapter) *AckHandler {
	return &AckHandler{
		config:    config,
		publisher: publisher,
		logger:    logger,
	}
}

// WrapHandler 包装用户的 handler，添加重试和死信队列逻辑
func (h *AckHandler) WrapHandler(handler Handler) Handler {
	return func(ctx context.Context, msg *message.Message) error {
		var lastErr error

		// 应用层重试（不依赖 MQ 的 Nack 机制）
		for attempt := 0; attempt <= h.config.MaxRetries; attempt++ {
			// 设置当前重试次数到 metadata
			msg.Metadata.Set("retry_count", fmt.Sprintf("%d", attempt))

			// 执行用户的处理逻辑
			err := handler(ctx, msg)
			if err == nil {
				// 处理成功
				return nil
			}

			lastErr = err

			// 如果还有重试机会，等待后继续
			if attempt < h.config.MaxRetries {
				h.logger.Info("message processing failed, will retry", watermill.LogFields{
					"message_uuid": msg.UUID,
					"retry_count":  attempt,
					"error":        err.Error(),
				})
				time.Sleep(h.config.RetryDelay)
			}
		}

		// 所有重试都失败了
		h.logger.Error("message processing failed after max retries", lastErr, watermill.LogFields{
			"message_uuid": msg.UUID,
			"retry_count":  h.config.MaxRetries,
		})

		// 如果启用死信队列，发送到死信队列
		if h.config.EnableDeadLetter && h.publisher != nil {
			if dlErr := h.sendToDeadLetter(ctx, msg, lastErr); dlErr != nil {
				h.logger.Error("failed to send to dead letter queue", dlErr, watermill.LogFields{
					"message_uuid": msg.UUID,
				})
			}
		}

		// 返回 nil，让消息被 ACK（不再重试）
		// 这样可以避免 Kafka 的 offset 卡住和 NATS 的无限重试
		return nil
	}
}

// getRetryCount 获取消息的重试次数
func (h *AckHandler) getRetryCount(msg *message.Message) int {
	if countStr := msg.Metadata.Get("retry_count"); countStr != "" {
		var count int
		fmt.Sscanf(countStr, "%d", &count)
		return count
	}
	return 0
}

// setRetryCount 设置消息的重试次数
func (h *AckHandler) setRetryCount(msg *message.Message, count int) {
	msg.Metadata.Set("retry_count", fmt.Sprintf("%d", count))
}

// sendToDeadLetter 发送消息到死信队列
func (h *AckHandler) sendToDeadLetter(ctx context.Context, msg *message.Message, originalErr error) error {
	// 构造死信队列主题名
	originalTopic := msg.Metadata.Get("topic")
	if originalTopic == "" {
		originalTopic = "unknown"
	}
	deadLetterTopic := originalTopic + h.config.DeadLetterSuffix

	// 添加失败信息到 metadata
	msg.Metadata.Set("dead_letter_reason", originalErr.Error())
	msg.Metadata.Set("dead_letter_time", time.Now().Format(time.RFC3339))
	msg.Metadata.Set("original_topic", originalTopic)

	// 发送到死信队列
	return h.publisher.PublishWithMetadata(ctx, deadLetterTopic, msg.Payload, msg.Metadata)
}

// SafeHandler 安全的消息处理器（捕获 panic）
type SafeHandler struct {
	logger watermill.LoggerAdapter
}

// NewSafeHandler 创建安全处理器
func NewSafeHandler(logger watermill.LoggerAdapter) *SafeHandler {
	return &SafeHandler{
		logger: logger,
	}
}

// WrapHandler 包装 handler，捕获 panic
func (s *SafeHandler) WrapHandler(handler Handler) Handler {
	return func(ctx context.Context, msg *message.Message) (err error) {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("handler panicked", fmt.Errorf("%v", r), watermill.LogFields{
					"message_uuid": msg.UUID,
				})
				err = fmt.Errorf("handler panicked: %v", r)
			}
		}()

		return handler(ctx, msg)
	}
}
