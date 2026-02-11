package consistenthash

import (
	"github.com/stathat/consistent"
)

// ConsistentHash 一致性哈希结构体
type ConsistentHash struct {
	c *consistent.Consistent
}

// NewConsistentHash 创建一个新的一致性哈希实例
func NewConsistentHash() *ConsistentHash {
	return &ConsistentHash{
		c: consistent.New(),
	}
}

// Add 添加节点
func (ch *ConsistentHash) Add(node string) {
	ch.c.Add(node)
}

// Remove 移除节点
func (ch *ConsistentHash) Remove(node string) {
	ch.c.Remove(node)
}

// Get 获取给定key对应的节点
func (ch *ConsistentHash) Get(key string) (string, error) {
	return ch.c.Get(key)
}

// Set 设置节点列表（替换当前节点列表）
func (ch *ConsistentHash) Set(nodes []string) {
	ch.c.Set(nodes)
}

// GetTwo 获取给定key对应的两个不同节点
func (ch *ConsistentHash) GetTwo(key string) (string, string, error) {
	return ch.c.GetTwo(key)
}

// GetN 获取给定key对应的N个不同节点
func (ch *ConsistentHash) GetN(key string, n int) ([]string, error) {
	return ch.c.GetN(key, n)
}

// Members 返回所有节点
func (ch *ConsistentHash) Members() []string {
	return ch.c.Members()
}
