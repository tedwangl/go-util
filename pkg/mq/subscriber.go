package mq

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type (
	Subscriber struct {
		subscriber message.Subscriber
		logger     watermill.LoggerAdapter
	}
	Handler func(ctx context.Context, msg *message.Message) error
)

func NewSubscriber(sub message.Subscriber, logger watermill.LoggerAdapter) *Subscriber {
	return &Subscriber{
		subscriber: sub,
		logger:     logger,
	}
}

func (s *Subscriber) Subscribe(ctx context.Context, topic string, handler Handler) error {
	messages, err := s.subscriber.Subscribe(ctx, topic)
	if err != nil {
		return err
	}

	go s.processMessages(ctx, topic, messages, handler)
	return nil
}

func (s *Subscriber) processMessages(ctx context.Context, topic string, messages <-chan *message.Message, handler Handler) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				return
			}

			// 添加 topic 信息到 metadata（用于死信队列）
			if msg.Metadata.Get("topic") == "" {
				msg.Metadata.Set("topic", topic)
			}

			if err := handler(ctx, msg); err != nil {
				s.logger.Error("handler failed", err, watermill.LogFields{
					"message_uuid": msg.UUID,
					"topic":        topic,
				})
				msg.Nack()
			} else {
				msg.Ack()
			}
		}
	}
}

func (s *Subscriber) Close() error {
	if closer, ok := s.subscriber.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

type BatchHandler func(ctx context.Context, msgs []*message.Message) error

func (s *Subscriber) SubscribeBatch(ctx context.Context, topic string, batchSize int, timeout time.Duration, handler BatchHandler) error {
	messages, err := s.subscriber.Subscribe(ctx, topic)
	if err != nil {
		return err
	}

	go s.processBatchMessages(ctx, messages, batchSize, timeout, handler)
	return nil
}

func (s *Subscriber) processBatchMessages(ctx context.Context, messages <-chan *message.Message, batchSize int, timeout time.Duration, handler BatchHandler) {
	batch := make([]*message.Message, 0, batchSize)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// 收集窗口：收到消息后等待一小段时间收集更多消息
	collectWindow := 50 * time.Millisecond
	collectTimer := time.NewTimer(collectWindow)
	collectTimer.Stop() // 初始停止

	flushBatch := func() {
		if len(batch) > 0 {
			s.handleBatch(ctx, batch, handler)
			batch = make([]*message.Message, 0, batchSize)
		}
		timer.Reset(timeout)
		collectTimer.Stop()
	}

	collecting := false

	for {
		select {
		case <-ctx.Done():
			flushBatch()
			return

		case msg, ok := <-messages:
			if !ok {
				flushBatch()
				return
			}

			batch = append(batch, msg)

			// 达到批量大小，立即处理
			if len(batch) >= batchSize {
				flushBatch()
				collecting = false
			} else if !collecting {
				// 开始收集窗口
				collecting = true
				collectTimer.Reset(collectWindow)
			}

		case <-collectTimer.C:
			// 收集窗口结束
			flushBatch()
			collecting = false

		case <-timer.C:
			// 超时，处理当前批次
			flushBatch()
			collecting = false
		}
	}
}

func (s *Subscriber) handleBatch(ctx context.Context, batch []*message.Message, handler BatchHandler) {
	if err := handler(ctx, batch); err != nil {
		s.logger.Error("batch handler failed", err, watermill.LogFields{
			"batch_size": len(batch),
		})
		for _, msg := range batch {
			msg.Nack()
		}
	} else {
		for _, msg := range batch {
			msg.Ack()
		}
	}
}
