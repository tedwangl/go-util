package mq

import "time"

const (
	BrokerTypeKafka    BrokerType = "kafka"
	BrokerTypeRabbitMQ BrokerType = "rabbitmq"
	BrokerTypeNATS     BrokerType = "nats"
)

// PartitionStrategy Kafka 分区策略
type PartitionStrategy string

const (
	// PartitionStrategyHash 根据消息 Key 的 Hash 值分区（默认）
	// 相同 Key 的消息会进入同一分区，保证顺序性
	PartitionStrategyHash PartitionStrategy = "hash"

	// PartitionStrategyRoundRobin 轮询分区
	// 消息均匀分布到所有分区，无顺序保证
	PartitionStrategyRoundRobin PartitionStrategy = "round_robin"

	// PartitionStrategyRandom 随机分区
	// 消息随机分配到分区
	PartitionStrategyRandom PartitionStrategy = "random"

	// PartitionStrategyManual 手动指定分区
	// 需要在消息 metadata 中设置 "partition" 字段
	PartitionStrategyManual PartitionStrategy = "manual"
)

type (
	// BrokerType MQ 类型
	BrokerType string

	Config struct {
		Broker   BrokerConfig   `json:",optional"`
		Kafka    KafkaConfig    `json:",optional"`
		RabbitMQ RabbitMQConfig `json:",optional"`
		NATS     NATSConfig     `json:",optional"`
	}

	BrokerConfig struct {
		Type BrokerType `json:",default=kafka,options=[kafka,rabbitmq,nats]"`
		// Addr 单节点地址（兼容旧配置）
		Addr string `json:",optional"`
		// Addrs 集群地址列表（优先使用）
		// Kafka: ["broker1:9092", "broker2:9092", "broker3:9092"]
		// NATS: ["nats://node1:4222", "nats://node2:4222"]
		Addrs []string `json:",optional"`
	}

	KafkaConfig struct {
		ConsumerGroup string `json:",default=go-util-group"`
		// FetchMaxMessages 每次拉取的最大消息数
		FetchMaxMessages int `json:",default=100"`
		// RequiredAcks 发布确认级别
		// 0: NoResponse - 不等待确认（最快，可能丢消息）
		// 1: WaitForLocal - 等待 Leader 确认（默认）
		// -1: WaitForAll - 等待所有副本确认（最安全，最慢）
		RequiredAcks int `json:",default=1"`
		// PartitionStrategy 分区策略
		PartitionStrategy PartitionStrategy `json:",default=hash"`
		// Kafka 默认持久化，无需额外配置
	}

	RabbitMQConfig struct {
		ExchangeName string `json:",default=logs.direct"`
		ExchangeType string `json:",default=direct,options=[direct,topic,fanout,headers]"`
		// Durable 持久化（Exchange 和 Queue）
		Durable bool `json:",default=true"`
		// PrefetchCount QoS 预取数量
		PrefetchCount int `json:",default=10"`
		// DeliveryMode 消息持久化模式
		// 1: Non-persistent - 不持久化（内存）
		// 2: Persistent - 持久化到磁盘（默认）
		DeliveryMode uint8 `json:",default=2"`
	}

	NATSConfig struct {
		QueueGroupPrefix string `json:",optional"`
		// EnableJetStream 是否启用 JetStream（持久化）
		EnableJetStream bool `json:",default=false"`
		// StreamName JetStream 流名称
		StreamName string `json:",default=EVENTS"`
		// MaxAckPending 最大未确认消息数（JetStream）
		MaxAckPending int `json:",default=100"`
	}

	RetryConfig struct {
		MaxRetries      int
		InitialDelay    time.Duration
		MaxDelay        time.Duration
		Multiplier      float64
		RetryTopic      string
		DeadLetterTopic string
	}
)
