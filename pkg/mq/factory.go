package mq

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Factory MQ 工厂
// 负责创建各种 MQ 组件（Publisher、Subscriber、Router 等）
type Factory struct {
	config Config
	logger watermill.LoggerAdapter
}

// NewFactory 创建工厂实例
func NewFactory(config Config, logger watermill.LoggerAdapter) *Factory {
	return &Factory{
		config: config,
		logger: logger,
	}
}

// CreatePublisher 创建发布者
func (f *Factory) CreatePublisher() (*Publisher, error) {
	var pub message.Publisher
	var err error

	switch f.config.Broker.Type {
	case BrokerTypeKafka:
		pub, err = f.createKafkaPublisher()
	case BrokerTypeRabbitMQ:
		pub, err = f.createRabbitMQPublisher()
	case BrokerTypeNATS:
		pub, err = f.createNATSPublisher()
	default:
		return nil, fmt.Errorf("unsupported broker type: %s (supported: kafka, rabbitmq, nats)", f.config.Broker.Type)
	}

	if err != nil {
		return nil, err
	}

	return NewPublisher(pub, f.logger), nil
}

// CreateSubscriber 创建订阅者
func (f *Factory) CreateSubscriber() (*Subscriber, error) {
	var sub message.Subscriber
	var err error

	switch f.config.Broker.Type {
	case BrokerTypeKafka:
		sub, err = f.createKafkaSubscriber()
	case BrokerTypeRabbitMQ:
		sub, err = f.createRabbitMQSubscriber()
	case BrokerTypeNATS:
		sub, err = f.createNATSSubscriber()
	default:
		return nil, fmt.Errorf("unsupported broker type: %s (supported: kafka, rabbitmq, nats)", f.config.Broker.Type)
	}

	if err != nil {
		return nil, err
	}

	return NewSubscriber(sub, f.logger), nil
}

// CreateRouter 创建路由器
func (f *Factory) CreateRouter() (*Router, error) {
	sub, err := f.CreateSubscriber()
	if err != nil {
		return nil, err
	}
	return NewRouter(sub, f.logger), nil
}

// CreateDelayQueue 创建延迟队列
func (f *Factory) CreateDelayQueue() (*DelayQueue, error) {
	pub, err := f.CreatePublisher()
	if err != nil {
		return nil, err
	}
	sub, err := f.CreateSubscriber()
	if err != nil {
		return nil, err
	}
	return NewDelayQueue(pub, sub, f.logger), nil
}

// CreateDeadLetterQueue 创建死信队列
func (f *Factory) CreateDeadLetterQueue() (*DeadLetterQueue, error) {
	sub, err := f.CreateSubscriber()
	if err != nil {
		return nil, err
	}
	pub, err := f.CreatePublisher()
	if err != nil {
		return nil, err
	}
	return NewDeadLetterQueue(sub, pub, f.logger), nil
}

// CreateRetryMiddleware 创建重试中间件
func (f *Factory) CreateRetryMiddleware(config RetryConfig) (*RetryMiddleware, error) {
	pub, err := f.CreatePublisher()
	if err != nil {
		return nil, err
	}
	return NewRetryMiddleware(config, pub, f.logger), nil
}

// CreateAckHandler 创建 ACK 处理器
func (f *Factory) CreateAckHandler(config AckConfig) (*AckHandler, error) {
	pub, err := f.CreatePublisher()
	if err != nil {
		return nil, err
	}
	return NewAckHandler(config, pub, f.logger), nil
}

// CreateAll 创建所有组件
func (f *Factory) CreateAll(ctx context.Context) (*Publisher, *Subscriber, *Router, *DelayQueue, *DeadLetterQueue, error) {
	pub, err := f.CreatePublisher()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	sub, err := f.CreateSubscriber()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	router := NewRouter(sub, f.logger)
	delayQueue := NewDelayQueue(pub, sub, f.logger)
	deadLetterQueue := NewDeadLetterQueue(sub, pub, f.logger)

	return pub, sub, router, delayQueue, deadLetterQueue, nil
}

// Close 关闭所有资源
func (f *Factory) Close() error {
	// Watermill 的 Publisher 和 Subscriber 会在各自的 Close 方法中处理
	return nil
}

// ValidateConfig 验证配置
func (f *Factory) ValidateConfig() error {
	if f.config.Broker.Addr == "" {
		return fmt.Errorf("broker address is required")
	}

	switch f.config.Broker.Type {
	case BrokerTypeKafka:
		if f.config.Kafka.ConsumerGroup == "" {
			return fmt.Errorf("kafka consumer group is required")
		}
	case BrokerTypeRabbitMQ:
		if f.config.RabbitMQ.ExchangeName == "" {
			return fmt.Errorf("rabbitmq exchange name is required")
		}
	case BrokerTypeNATS:
		// NATS 配置可选
	default:
		return fmt.Errorf("unsupported broker type: %s", f.config.Broker.Type)
	}

	return nil
}
