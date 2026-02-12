package mq

import (
	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
)

// createKafkaPublisher 创建 Kafka 发布者
func (f *Factory) createKafkaPublisher() (message.Publisher, error) {
	requiredAcks := f.config.Kafka.RequiredAcks
	if requiredAcks == 0 {
		requiredAcks = 1 // 默认等待 Leader 确认
	}

	// 转换为 Sarama 的 RequiredAcks
	var acks sarama.RequiredAcks
	switch requiredAcks {
	case 0:
		acks = sarama.NoResponse // 不等待确认
	case 1:
		acks = sarama.WaitForLocal // 等待 Leader 确认
	case -1:
		acks = sarama.WaitForAll // 等待所有副本确认
	default:
		acks = sarama.WaitForLocal
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.RequiredAcks = acks
	saramaConfig.Producer.Return.Successes = true // 必须开启才能获取发送结果

	// 分区策略
	switch f.config.Kafka.PartitionStrategy {
	case PartitionStrategyRoundRobin:
		saramaConfig.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	case PartitionStrategyHash:
		saramaConfig.Producer.Partitioner = sarama.NewHashPartitioner
	case PartitionStrategyRandom:
		saramaConfig.Producer.Partitioner = sarama.NewRandomPartitioner
	case PartitionStrategyManual:
		saramaConfig.Producer.Partitioner = sarama.NewManualPartitioner
	default:
		// 默认使用 Hash 分区（根据 Key 分区，保证相同 Key 进同一分区）
		saramaConfig.Producer.Partitioner = sarama.NewHashPartitioner
	}

	// 获取 Broker 地址列表
	brokers := f.getBrokerAddrs()

	return kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:               brokers,
			Marshaler:             kafka.DefaultMarshaler{},
			OverwriteSaramaConfig: saramaConfig,
		},
		f.logger,
	)
}

// createKafkaSubscriber 创建 Kafka 订阅者
func (f *Factory) createKafkaSubscriber() (message.Subscriber, error) {
	consumerGroup := f.config.Kafka.ConsumerGroup
	if consumerGroup == "" {
		consumerGroup = "go-util-group"
	}

	fetchMaxMessages := f.config.Kafka.FetchMaxMessages
	if fetchMaxMessages == 0 {
		fetchMaxMessages = 100
	}

	saramaConfig := kafka.DefaultSaramaSubscriberConfig()
	// 从最早的消息开始消费（用于测试和新消费者组）
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	// 控制每次拉取的最大消息数
	saramaConfig.Consumer.MaxProcessingTime = 1000 // 1 秒

	// 获取 Broker 地址列表
	brokers := f.getBrokerAddrs()

	return kafka.NewSubscriber(
		kafka.SubscriberConfig{
			Brokers:               brokers,
			ConsumerGroup:         consumerGroup,
			OverwriteSaramaConfig: saramaConfig,
		},
		f.logger,
	)
}

// getBrokerAddrs 获取 Broker 地址列表（支持单节点和集群）
func (f *Factory) getBrokerAddrs() []string {
	// 优先使用 Addrs（集群配置）
	if len(f.config.Broker.Addrs) > 0 {
		return f.config.Broker.Addrs
	}
	// 兼容旧配置 Addr（单节点）
	if f.config.Broker.Addr != "" {
		return []string{f.config.Broker.Addr}
	}
	// 默认 localhost
	return []string{"localhost:9092"}
}
