package mq

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Client 简化的 MQ 客户端
// 提供最简洁的 API，自动处理重试、死信队列等
type Client struct {
	config     Config
	factory    *Factory
	publisher  *Publisher
	subscriber *Subscriber
	ackHandler *AckHandler
	dlq        *DeadLetterQueue
	logger     watermill.LoggerAdapter
}

// ClientOption 客户端选项
type ClientOption func(*Client)

// WithLogger 设置日志器
func WithLogger(logger watermill.LoggerAdapter) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithAckConfig 设置 ACK 配置
func WithAckConfig(config AckConfig) ClientOption {
	return func(c *Client) {
		c.ackHandler = NewAckHandler(config, c.publisher, c.logger)
	}
}

// NewClient 创建 MQ 客户端
// 一步到位：配置加载、创建发布者、订阅者、ACK 处理器、死信队列
func NewClient(config Config, opts ...ClientOption) (*Client, error) {
	// 默认日志器
	logger := watermill.NewStdLogger(false, false)

	// 创建工厂
	factory := NewFactory(config, logger)

	// 创建发布者和订阅者
	publisher, err := factory.CreatePublisher()
	if err != nil {
		return nil, err
	}

	subscriber, err := factory.CreateSubscriber()
	if err != nil {
		return nil, err
	}

	// 创建死信队列
	dlq, err := factory.CreateDeadLetterQueue()
	if err != nil {
		return nil, err
	}

	client := &Client{
		config:     config,
		factory:    factory,
		publisher:  publisher,
		subscriber: subscriber,
		dlq:        dlq,
		logger:     logger,
	}

	// 应用选项
	for _, opt := range opts {
		opt(client)
	}

	// 如果没有设置 ACK 处理器，使用默认配置
	if client.ackHandler == nil {
		client.ackHandler = NewAckHandler(DefaultAckConfig(), publisher, logger)
	}

	return client, nil
}

// Publish 发布消息
func (c *Client) Publish(ctx context.Context, topic string, payload []byte) error {
	return c.publisher.Publish(ctx, topic, payload)
}

// PublishWithMetadata 发布带元数据的消息
func (c *Client) PublishWithMetadata(ctx context.Context, topic string, payload []byte, metadata map[string]string) error {
	return c.publisher.PublishWithMetadata(ctx, topic, payload, metadata)
}

// PublishMessage 发布结构化消息
func (c *Client) PublishMessage(ctx context.Context, topic string, msg *Message) error {
	wmsg, err := msg.EncodeToWatermillMessage()
	if err != nil {
		return err
	}
	return c.publisher.publisher.Publish(topic, wmsg)
}

// PublishJSON 发布 JSON 消息（自动编码）
func (c *Client) PublishJSON(ctx context.Context, topic string, data interface{}, opts ...MessageOption) error {
	msg := NewMessage(data, opts...)
	return c.PublishMessage(ctx, topic, msg)
}

// PublishBatch 批量发布消息
func (c *Client) PublishBatch(ctx context.Context, topic string, payloads [][]byte) error {
	return c.publisher.PublishBatch(ctx, topic, payloads)
}

// PublishWithPartitionKey 发布消息并指定分区键（仅 Kafka）
func (c *Client) PublishWithPartitionKey(ctx context.Context, topic string, payload []byte, partitionKey string) error {
	return c.publisher.PublishWithPartitionKey(ctx, topic, payload, partitionKey)
}

// Subscribe 订阅消息（自动带重试和死信队列）
func (c *Client) Subscribe(ctx context.Context, topic string, handler Handler) error {
	// 包装 handler，添加重试和死信队列逻辑
	wrappedHandler := c.ackHandler.WrapHandler(handler)
	return c.subscriber.Subscribe(ctx, topic, wrappedHandler)
}

// SubscribeRaw 订阅消息（不带重试，直接处理）
func (c *Client) SubscribeRaw(ctx context.Context, topic string, handler Handler) error {
	return c.subscriber.Subscribe(ctx, topic, handler)
}

// SubscribeTyped 订阅结构化消息（自动解码）
func (c *Client) SubscribeTyped(ctx context.Context, topic string, handler TypedHandler) error {
	wrappedHandler := c.ackHandler.WrapHandler(WrapTypedHandler(handler))
	return c.subscriber.Subscribe(ctx, topic, wrappedHandler)
}

// SubscribeBatch 批量订阅消息
func (c *Client) SubscribeBatch(ctx context.Context, topic string, batchSize int, timeout time.Duration, handler BatchHandler) error {
	return c.subscriber.SubscribeBatch(ctx, topic, batchSize, timeout, handler)
}

// SubscribeDeadLetter 订阅死信队列
func (c *Client) SubscribeDeadLetter(ctx context.Context, topic string, handler DeadLetterHandler) error {
	return c.dlq.Subscribe(ctx, topic, handler)
}

// RepublishDeadLetter 重新发布死信消息
func (c *Client) RepublishDeadLetter(ctx context.Context, msg *message.Message, targetTopic string) error {
	return c.dlq.Republish(ctx, msg, targetTopic)
}

// Close 关闭客户端
func (c *Client) Close() error {
	if err := c.publisher.Close(); err != nil {
		return err
	}
	if err := c.subscriber.Close(); err != nil {
		return err
	}
	return nil
}

// GetPublisher 获取底层 Publisher（高级用户）
func (c *Client) GetPublisher() *Publisher {
	return c.publisher
}

// GetSubscriber 获取底层 Subscriber（高级用户）
func (c *Client) GetSubscriber() *Subscriber {
	return c.subscriber
}

// GetFactory 获取底层 Factory（高级用户）
func (c *Client) GetFactory() *Factory {
	return c.factory
}
