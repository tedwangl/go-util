package mq

import (
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
)

// createRabbitMQPublisher 创建 RabbitMQ 发布者
func (f *Factory) createRabbitMQPublisher() (message.Publisher, error) {
	config := f.getRabbitMQConfig()
	return amqp.NewPublisher(config, f.logger)
}

// createRabbitMQSubscriber 创建 RabbitMQ 订阅者
func (f *Factory) createRabbitMQSubscriber() (message.Subscriber, error) {
	config := f.getRabbitMQConfig()
	return amqp.NewSubscriber(config, f.logger)
}

// getRabbitMQConfig 获取 RabbitMQ 配置
func (f *Factory) getRabbitMQConfig() amqp.Config {
	exchangeName := f.config.RabbitMQ.ExchangeName
	if exchangeName == "" {
		exchangeName = "logs.direct"
	}

	exchangeType := f.config.RabbitMQ.ExchangeType
	if exchangeType == "" {
		exchangeType = "direct"
	}

	prefetchCount := f.config.RabbitMQ.PrefetchCount
	if prefetchCount == 0 {
		prefetchCount = 10
	}

	deliveryMode := f.config.RabbitMQ.DeliveryMode
	if deliveryMode == 0 {
		deliveryMode = 2 // 默认持久化
	}

	return amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: f.config.Broker.Addr,
		},
		Marshaler: amqp.DefaultMarshaler{},
		Exchange: amqp.ExchangeConfig{
			GenerateName: amqp.GenerateExchangeNameConstant(exchangeName),
			Type:         exchangeType,
			Durable:      f.config.RabbitMQ.Durable, // Exchange 持久化
		},
		Queue: amqp.QueueConfig{
			GenerateName: amqp.GenerateQueueNameTopicName,
			Durable:      f.config.RabbitMQ.Durable, // Queue 持久化
		},
		QueueBind: amqp.QueueBindConfig{
			GenerateRoutingKey: func(topic string) string {
				return topic
			},
		},
		Publish: amqp.PublishConfig{
			GenerateRoutingKey: func(topic string) string {
				return topic
			},
			// 消息持久化模式
			ConfirmDelivery: true, // 启用发布确认
		},
		Consume: amqp.ConsumeConfig{
			Qos: amqp.QosConfig{
				PrefetchCount: prefetchCount,
			},
		},
		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
	}
}
