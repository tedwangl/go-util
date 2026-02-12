package mq

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Metrics 消息队列指标
type Metrics struct {
	PublishedCount   atomic.Int64
	PublishedErrors  atomic.Int64
	ConsumedCount    atomic.Int64
	ConsumedErrors   atomic.Int64
	AckCount         atomic.Int64
	NackCount        atomic.Int64
	ProcessingTimeMs atomic.Int64
}

// MetricsMiddleware 指标中间件
type MetricsMiddleware struct {
	metrics *Metrics
	logger  watermill.LoggerAdapter
}

// NewMetricsMiddleware 创建指标中间件
func NewMetricsMiddleware(logger watermill.LoggerAdapter) *MetricsMiddleware {
	return &MetricsMiddleware{
		metrics: &Metrics{},
		logger:  logger,
	}
}

// GetMetrics 获取指标
func (m *MetricsMiddleware) GetMetrics() *Metrics {
	return m.metrics
}

// WrapHandler 包装 handler，记录指标
func (m *MetricsMiddleware) WrapHandler(handler Handler) Handler {
	return func(ctx context.Context, msg *message.Message) error {
		m.metrics.ConsumedCount.Add(1)
		start := time.Now()

		err := handler(ctx, msg)

		elapsed := time.Since(start).Milliseconds()
		m.metrics.ProcessingTimeMs.Add(elapsed)

		if err != nil {
			m.metrics.ConsumedErrors.Add(1)
			m.metrics.NackCount.Add(1)
		} else {
			m.metrics.AckCount.Add(1)
		}

		return err
	}
}

// WrapPublisher 包装 Publisher，记录发布指标
func (m *MetricsMiddleware) WrapPublisher(pub *Publisher) *MetricsPublisher {
	return &MetricsPublisher{
		Publisher: pub,
		metrics:   m.metrics,
	}
}

// MetricsPublisher 带指标的发布者
type MetricsPublisher struct {
	*Publisher
	metrics *Metrics
}

// Publish 发布消息并记录指标
func (p *MetricsPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
	err := p.Publisher.Publish(ctx, topic, payload)
	p.metrics.PublishedCount.Add(1)
	if err != nil {
		p.metrics.PublishedErrors.Add(1)
	}
	return err
}

// PublishWithMetadata 发布带元数据的消息并记录指标
func (p *MetricsPublisher) PublishWithMetadata(ctx context.Context, topic string, payload []byte, metadata map[string]string) error {
	err := p.Publisher.PublishWithMetadata(ctx, topic, payload, metadata)
	p.metrics.PublishedCount.Add(1)
	if err != nil {
		p.metrics.PublishedErrors.Add(1)
	}
	return err
}

// PublishBatch 批量发布并记录指标
func (p *MetricsPublisher) PublishBatch(ctx context.Context, topic string, payloads [][]byte) error {
	err := p.Publisher.PublishBatch(ctx, topic, payloads)
	p.metrics.PublishedCount.Add(int64(len(payloads)))
	if err != nil {
		p.metrics.PublishedErrors.Add(1)
	}
	return err
}

// ResetMetrics 重置指标
func (m *Metrics) Reset() {
	m.PublishedCount.Store(0)
	m.PublishedErrors.Store(0)
	m.ConsumedCount.Store(0)
	m.ConsumedErrors.Store(0)
	m.AckCount.Store(0)
	m.NackCount.Store(0)
	m.ProcessingTimeMs.Store(0)
}

// GetSnapshot 获取指标快照
func (m *Metrics) GetSnapshot() MetricsSnapshot {
	return MetricsSnapshot{
		PublishedCount:   m.PublishedCount.Load(),
		PublishedErrors:  m.PublishedErrors.Load(),
		ConsumedCount:    m.ConsumedCount.Load(),
		ConsumedErrors:   m.ConsumedErrors.Load(),
		AckCount:         m.AckCount.Load(),
		NackCount:        m.NackCount.Load(),
		ProcessingTimeMs: m.ProcessingTimeMs.Load(),
	}
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	PublishedCount   int64
	PublishedErrors  int64
	ConsumedCount    int64
	ConsumedErrors   int64
	AckCount         int64
	NackCount        int64
	ProcessingTimeMs int64
}

// AvgProcessingTimeMs 平均处理时间（毫秒）
func (s MetricsSnapshot) AvgProcessingTimeMs() float64 {
	if s.ConsumedCount == 0 {
		return 0
	}
	return float64(s.ProcessingTimeMs) / float64(s.ConsumedCount)
}

// ErrorRate 错误率
func (s MetricsSnapshot) ErrorRate() float64 {
	if s.ConsumedCount == 0 {
		return 0
	}
	return float64(s.ConsumedErrors) / float64(s.ConsumedCount)
}

// PublishErrorRate 发布错误率
func (s MetricsSnapshot) PublishErrorRate() float64 {
	if s.PublishedCount == 0 {
		return 0
	}
	return float64(s.PublishedErrors) / float64(s.PublishedCount)
}
