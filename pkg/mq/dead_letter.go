package mq

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// DeadLetterQueue 死信队列
// 用于处理失败的消息
type DeadLetterQueue struct {
	subscriber *Subscriber
	publisher  *Publisher
	logger     watermill.LoggerAdapter
}

// DeadLetterHandler 死信消息处理器
type DeadLetterHandler func(ctx context.Context, msg *message.Message, reason string) error

// NewDeadLetterQueue 创建死信队列
func NewDeadLetterQueue(subscriber *Subscriber, publisher *Publisher, logger watermill.LoggerAdapter) *DeadLetterQueue {
	return &DeadLetterQueue{
		subscriber: subscriber,
		publisher:  publisher,
		logger:     logger,
	}
}

// Subscribe 订阅死信队列
func (dlq *DeadLetterQueue) Subscribe(ctx context.Context, topic string, handler DeadLetterHandler) error {
	deadLetterTopic := topic + ".dead-letter"

	return dlq.subscriber.Subscribe(ctx, deadLetterTopic, func(ctx context.Context, msg *message.Message) error {
		reason := msg.Metadata.Get("dead_letter_reason")
		if reason == "" {
			reason = "unknown"
		}

		dlq.logger.Info("processing dead letter message", watermill.LogFields{
			"message_uuid":   msg.UUID,
			"reason":         reason,
			"original_topic": msg.Metadata.Get("original_topic"),
			"failed_time":    msg.Metadata.Get("dead_letter_time"),
		})

		return handler(ctx, msg, reason)
	})
}

// Republish 重新发布死信消息到原队列
func (dlq *DeadLetterQueue) Republish(ctx context.Context, msg *message.Message, targetTopic string) error {
	// 清理死信相关的 metadata
	delete(msg.Metadata, "dead_letter_reason")
	delete(msg.Metadata, "dead_letter_time")
	delete(msg.Metadata, "retry_count")

	dlq.logger.Info("republishing dead letter message", watermill.LogFields{
		"message_uuid": msg.UUID,
		"target_topic": targetTopic,
	})

	return dlq.publisher.PublishWithMetadata(ctx, targetTopic, msg.Payload, msg.Metadata)
}

// Monitor 监控死信队列
// 定期统计死信消息数量并触发告警
type DeadLetterStats struct {
	Topic    string
	Count    int
	LastSeen time.Time
}

func (dlq *DeadLetterQueue) Monitor(ctx context.Context, topics []string, interval time.Duration, alertFunc func(stats []DeadLetterStats)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	stats := make(map[string]*DeadLetterStats)

	// 订阅所有死信队列
	for _, topic := range topics {
		deadLetterTopic := topic + ".dead-letter"
		topicName := topic // 捕获变量

		dlq.subscriber.Subscribe(ctx, deadLetterTopic, func(ctx context.Context, msg *message.Message) error {
			if stats[topicName] == nil {
				stats[topicName] = &DeadLetterStats{
					Topic: topicName,
				}
			}
			stats[topicName].Count++
			stats[topicName].LastSeen = time.Now()
			return nil
		})
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if len(stats) > 0 && alertFunc != nil {
				// 转换为切片
				statsList := make([]DeadLetterStats, 0, len(stats))
				for _, s := range stats {
					if s.Count > 0 {
						statsList = append(statsList, *s)
					}
				}

				if len(statsList) > 0 {
					alertFunc(statsList)
				}

				// 重置计数
				stats = make(map[string]*DeadLetterStats)
			}
		}
	}
}

// AnalyzeFailures 分析死信消息的失败原因
func (dlq *DeadLetterQueue) AnalyzeFailures(ctx context.Context, topic string, duration time.Duration) (map[string]int, error) {
	deadLetterTopic := topic + ".dead-letter"
	reasons := make(map[string]int)

	done := make(chan struct{})

	handler := func(ctx context.Context, msg *message.Message) error {
		reason := msg.Metadata.Get("dead_letter_reason")
		if reason == "" {
			reason = "unknown"
		}
		reasons[reason]++
		return nil
	}

	if err := dlq.subscriber.Subscribe(ctx, deadLetterTopic, handler); err != nil {
		return nil, err
	}

	// 等待指定时间
	go func() {
		time.Sleep(duration)
		close(done)
	}()

	<-done
	return reasons, nil
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries        int
	RetryDelay        time.Duration
	BackoffMultiplier float64
}

// DefaultRetryPolicy 默认重试策略
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:        3,
		RetryDelay:        time.Second,
		BackoffMultiplier: 2.0,
	}
}

// RetryWithPolicy 使用指定策略重试死信消息
func (dlq *DeadLetterQueue) RetryWithPolicy(ctx context.Context, msg *message.Message, policy RetryPolicy, handler Handler) error {
	for i := 0; i < policy.MaxRetries; i++ {
		dlq.logger.Info("retrying dead letter message", watermill.LogFields{
			"message_uuid": msg.UUID,
			"attempt":      i + 1,
			"max_retries":  policy.MaxRetries,
		})

		if err := handler(ctx, msg); err == nil {
			dlq.logger.Info("dead letter message retry succeeded", watermill.LogFields{
				"message_uuid": msg.UUID,
				"attempt":      i + 1,
			})
			return nil
		}

		if i < policy.MaxRetries-1 {
			delay := time.Duration(float64(policy.RetryDelay) * float64(i+1) * policy.BackoffMultiplier)
			dlq.logger.Info("dead letter message retry failed, waiting", watermill.LogFields{
				"message_uuid": msg.UUID,
				"attempt":      i + 1,
				"next_delay":   delay,
			})
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("all retry attempts failed")
}
