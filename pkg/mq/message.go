package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

// Message 消息结构
// 提供统一的消息格式和编解码
type Message struct {
	// ID 消息唯一标识
	ID string `json:"id"`
	// Type 消息类型（用于路由和处理）
	Type string `json:"type"`
	// Data 消息数据（业务数据）
	Data interface{} `json:"data"`
	// Metadata 元数据（Header）
	Metadata map[string]string `json:"metadata,omitempty"`
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
	// TraceID 链路追踪 ID
	TraceID string `json:"trace_id,omitempty"`
	// Source 消息来源
	Source string `json:"source,omitempty"`
}

// MessageOption 消息选项
type MessageOption func(*Message)

// WithType 设置消息类型
func WithType(msgType string) MessageOption {
	return func(m *Message) {
		m.Type = msgType
	}
}

// WithMetadata 设置元数据
func WithMetadata(metadata map[string]string) MessageOption {
	return func(m *Message) {
		if m.Metadata == nil {
			m.Metadata = make(map[string]string)
		}
		for k, v := range metadata {
			m.Metadata[k] = v
		}
	}
}

// WithTraceID 设置链路追踪 ID
func WithTraceID(traceID string) MessageOption {
	return func(m *Message) {
		m.TraceID = traceID
	}
}

// WithSource 设置消息来源
func WithSource(source string) MessageOption {
	return func(m *Message) {
		m.Source = source
	}
}

// NewMessage 创建消息
func NewMessage(data interface{}, opts ...MessageOption) *Message {
	msg := &Message{
		ID:        generateMessageID(),
		Data:      data,
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}

	for _, opt := range opts {
		opt(msg)
	}

	return msg
}

// Encode 编码消息为 JSON
func (m *Message) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// EncodeToWatermillMessage 编码为 Watermill 消息
func (m *Message) EncodeToWatermillMessage() (*message.Message, error) {
	payload, err := m.Encode()
	if err != nil {
		return nil, err
	}

	wmsg := message.NewMessage(m.ID, payload)

	// 设置元数据
	if m.Type != "" {
		wmsg.Metadata.Set("type", m.Type)
	}
	if m.TraceID != "" {
		wmsg.Metadata.Set("trace_id", m.TraceID)
	}
	if m.Source != "" {
		wmsg.Metadata.Set("source", m.Source)
	}
	for k, v := range m.Metadata {
		wmsg.Metadata.Set(k, v)
	}

	return wmsg, nil
}

// DecodeMessage 解码消息
func DecodeMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// DecodeFromWatermillMessage 从 Watermill 消息解码
func DecodeFromWatermillMessage(wmsg *message.Message) (*Message, error) {
	msg, err := DecodeMessage(wmsg.Payload)
	if err != nil {
		return nil, err
	}

	// 从 Watermill metadata 中恢复信息
	if msg.ID == "" {
		msg.ID = wmsg.UUID
	}
	if msgType := wmsg.Metadata.Get("type"); msgType != "" {
		msg.Type = msgType
	}
	if traceID := wmsg.Metadata.Get("trace_id"); traceID != "" {
		msg.TraceID = traceID
	}
	if source := wmsg.Metadata.Get("source"); source != "" {
		msg.Source = source
	}

	return msg, nil
}

// DecodeDataTo 解码 Data 字段到指定类型
func (m *Message) DecodeDataTo(v interface{}) error {
	// Data 可能是 map[string]interface{} 或其他类型
	// 先转为 JSON 再解码到目标类型
	data, err := json.Marshal(m.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// generateMessageID 生成消息 ID
func generateMessageID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// TypedHandler 类型化的消息处理器
// 自动解码消息并提供结构化访问
type TypedHandler func(*Message) error

// WrapTypedHandler 包装类型化 handler 为标准 handler
func WrapTypedHandler(handler TypedHandler) Handler {
	return func(ctx context.Context, wmsg *message.Message) error {
		msg, err := DecodeFromWatermillMessage(wmsg)
		if err != nil {
			return fmt.Errorf("failed to decode message: %w", err)
		}
		return handler(msg)
	}
}
