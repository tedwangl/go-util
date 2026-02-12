package mq

import (
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
)

// createNATSPublisher 创建 NATS 发布者
func (f *Factory) createNATSPublisher() (message.Publisher, error) {
	enableJetStream := f.config.NATS.EnableJetStream
	url := f.getNATSURL()

	config := nats.PublisherConfig{
		URL: url,
		JetStream: nats.JetStreamConfig{
			Disabled: !enableJetStream, // 根据配置启用/禁用 JetStream
		},
	}

	return nats.NewPublisher(config, f.logger)
}

// createNATSSubscriber 创建 NATS 订阅者
func (f *Factory) createNATSSubscriber() (message.Subscriber, error) {
	enableJetStream := f.config.NATS.EnableJetStream
	url := f.getNATSURL()

	config := nats.SubscriberConfig{
		URL:              url,
		QueueGroupPrefix: f.config.NATS.QueueGroupPrefix,
		JetStream: nats.JetStreamConfig{
			Disabled: !enableJetStream, // 根据配置启用/禁用 JetStream
		},
	}

	return nats.NewSubscriber(config, f.logger)
}

// getNATSURL 获取 NATS 连接地址（支持单节点和集群）
func (f *Factory) getNATSURL() string {
	// 优先使用 Addrs（集群配置）
	// NATS 集群格式: "nats://node1:4222,nats://node2:4222,nats://node3:4222"
	if len(f.config.Broker.Addrs) > 0 {
		url := ""
		for i, addr := range f.config.Broker.Addrs {
			if i > 0 {
				url += ","
			}
			url += addr
		}
		return url
	}
	// 兼容旧配置 Addr（单节点）
	if f.config.Broker.Addr != "" {
		return f.config.Broker.Addr
	}
	// 默认 localhost
	return "nats://localhost:4222"
}
