package genid

import (
	"github.com/bwmarrin/snowflake"
)

// SnowflakeID 生成器结构体
type SnowflakeID struct {
	node *snowflake.Node
}

// NewSnowflakeID 创建一个新的雪花ID生成器
func NewSnowflakeID(nodeID int64) (*SnowflakeID, error) {
	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		return nil, err
	}
	return &SnowflakeID{node: node}, nil
}

// NextID 生成下一个ID
func (s *SnowflakeID) NextID() int64 {
	return s.node.Generate().Int64()
}

// NextStringID 生成下一个ID（字符串格式）
func (s *SnowflakeID) NextStringID() string {
	return s.node.Generate().String()
}

// ParseID 解析ID为snowflake结构体
func (s *SnowflakeID) ParseID(id int64) *snowflake.ID {
	sfID := snowflake.ParseInt64(id)
	return &sfID
}

// GetTimestampFromID 从ID中提取时间戳
func GetTimestampFromID(id int64) int64 {
	sfID := snowflake.ParseInt64(id)
	return sfID.Time()
}

// GetTimeFromID 从ID中提取时间
func GetTimeFromID(id int64) int64 {
	sfID := snowflake.ParseInt64(id)
	return sfID.Time()
}

// GetNodeIDFromID 从ID中提取节点ID
func GetNodeIDFromID(id int64) int64 {
	sfID := snowflake.ParseInt64(id)
	return sfID.Node()
}

// GetStepFromID 从ID中提取序列号
func GetStepFromID(id int64) int64 {
	sfID := snowflake.ParseInt64(id)
	return sfID.Step()
}
