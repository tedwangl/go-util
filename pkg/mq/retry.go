package mq

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type RetryMiddleware struct {
	config    RetryConfig
	publisher *Publisher
	logger    watermill.LoggerAdapter
}

func NewRetryMiddleware(config RetryConfig, publisher *Publisher, logger watermill.LoggerAdapter) *RetryMiddleware {
	return &RetryMiddleware{
		config:    config,
		publisher: publisher,
		logger:    logger,
	}
}

func (r *RetryMiddleware) Handler(next Handler) Handler {
	return func(ctx context.Context, msg *message.Message) error {
		retryCount := 0
		if val, ok := msg.Metadata["retry_count"]; ok {
			fmt.Sscanf(val, "%d", &retryCount)
		}

		err := next(ctx, msg)
		if err == nil {
			return nil
		}

		if retryCount >= r.config.MaxRetries {
			r.logger.Error("max retries exceeded, sending to dead letter queue", err, watermill.LogFields{
				"message_uuid": msg.UUID,
				"retry_count":  retryCount,
			})
			return r.sendToDeadLetter(ctx, msg)
		}

		delay := r.calculateDelay(retryCount)
		msg.Metadata["retry_count"] = fmt.Sprintf("%d", retryCount+1)
		msg.Metadata["error"] = err.Error()

		r.logger.Info("retrying message", watermill.LogFields{
			"message_uuid": msg.UUID,
			"retry_count":  retryCount + 1,
			"delay":        delay,
		})

		time.Sleep(delay)
		return r.publisher.PublishWithMetadata(ctx, r.config.RetryTopic, msg.Payload, msg.Metadata)
	}
}

func (r *RetryMiddleware) calculateDelay(retryCount int) time.Duration {
	delay := time.Duration(float64(r.config.InitialDelay) * r.config.Multiplier)
	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}
	return delay
}

func (r *RetryMiddleware) sendToDeadLetter(ctx context.Context, msg *message.Message) error {
	if r.config.DeadLetterTopic == "" {
		return nil
	}
	msg.Metadata["dead_letter_reason"] = "max_retries_exceeded"
	return r.publisher.PublishWithMetadata(ctx, r.config.DeadLetterTopic, msg.Payload, msg.Metadata)
}
