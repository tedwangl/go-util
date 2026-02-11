// Package bloomfilter 布隆过滤器实现
package bloomfilter

import (
	"github.com/bits-and-blooms/bloom/v3"
)

// BloomFilter 布隆过滤器
type BloomFilter struct {
	filter *bloom.BloomFilter
}

// New 创建布隆过滤器
// n: 预期元素数量
// p: 误判率
func New(n uint, p float64) *BloomFilter {
	return &BloomFilter{
		filter: bloom.NewWithEstimates(n, p),
	}
}

// Add 添加元素到布隆过滤器
func (b *BloomFilter) Add(data []byte) {
	b.filter.Add(data)
}

// AddString 添加字符串到布隆过滤器
func (b *BloomFilter) AddString(s string) {
	b.filter.AddString(s)
}

// Contains 检查元素是否可能存在于布隆过滤器中
func (b *BloomFilter) Contains(data []byte) bool {
	return b.filter.Test(data)
}

// ContainsString 检查字符串是否可能存在于布隆过滤器中
func (b *BloomFilter) ContainsString(s string) bool {
	return b.filter.TestString(s)
}

// TestAndAdd 测试并添加元素到布隆过滤器
func (b *BloomFilter) TestAndAdd(data []byte) bool {
	return b.filter.TestAndAdd(data)
}

// TestAndAddString 测试并添加字符串到布隆过滤器
func (b *BloomFilter) TestAndAddString(s string) bool {
	return b.filter.TestAndAddString(s)
}

// Clear 清空布隆过滤器
func (b *BloomFilter) Clear() {
	b.filter.ClearAll()
}
