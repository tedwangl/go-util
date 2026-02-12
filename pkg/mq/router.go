package mq

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Router struct {
	subscriber *Subscriber
	handlers   map[string]Handler
	logger     watermill.LoggerAdapter
}

func NewRouter(subscriber *Subscriber, logger watermill.LoggerAdapter) *Router {
	return &Router{
		subscriber: subscriber,
		handlers:   make(map[string]Handler),
		logger:     logger,
	}
}

func (r *Router) AddHandler(topic string, handler Handler) {
	r.handlers[topic] = handler
}

func (r *Router) AddHandlerFunc(topic string, handler func(ctx context.Context, msg *message.Message) error) {
	r.handlers[topic] = handler
}

func (r *Router) Run(ctx context.Context, topics ...string) error {
	for _, topic := range topics {
		handler, exists := r.handlers[topic]
		if !exists {
			return fmt.Errorf("no handler registered for topic: %s", topic)
		}

		if err := r.subscriber.Subscribe(ctx, topic, handler); err != nil {
			return fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
		}
	}

	<-ctx.Done()
	return nil
}

func (r *Router) RunAll(ctx context.Context) error {
	topics := make([]string, 0, len(r.handlers))
	for topic := range r.handlers {
		topics = append(topics, topic)
	}
	return r.Run(ctx, topics...)
}

func (r *Router) Close() error {
	return r.subscriber.Close()
}
