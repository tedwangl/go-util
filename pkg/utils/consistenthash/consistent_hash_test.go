package consistenthash

import (
	"testing"
)

func TestConsistentHash(t *testing.T) {
	ch := NewConsistentHash()

	// 添加节点
	nodes := []string{"server1", "server2", "server3"}
	for _, node := range nodes {
		ch.Add(node)
	}

	// 测试获取节点
	key := "user123"
	node, err := ch.Get(key)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}

	if node == "" {
		t.Error("Expected non-empty node")
	}

	t.Logf("Key '%s' mapped to node '%s'", key, node)
}

func TestGetTwo(t *testing.T) {
	ch := NewConsistentHash()

	// 添加节点
	nodes := []string{"server1", "server2", "server3"}
	for _, node := range nodes {
		ch.Add(node)
	}

	// 测试获取两个节点
	key := "user456"
	node1, node2, err := ch.GetTwo(key)
	if err != nil {
		t.Errorf("GetTwo failed: %v", err)
	}

	if node1 == "" || node2 == "" {
		t.Error("Expected non-empty nodes")
	}

	if node1 == node2 {
		t.Error("Expected different nodes")
	}

	t.Logf("Key '%s' mapped to nodes '%s' and '%s'", key, node1, node2)
}

func TestGetN(t *testing.T) {
	ch := NewConsistentHash()

	// 添加节点
	nodes := []string{"server1", "server2", "server3", "server4", "server5"}
	for _, node := range nodes {
		ch.Add(node)
	}

	// 测试获取N个节点
	key := "user789"
	resultNodes, err := ch.GetN(key, 3)
	if err != nil {
		t.Errorf("GetN failed: %v", err)
	}

	if len(resultNodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(resultNodes))
	}

	t.Logf("Key '%s' mapped to nodes %v", key, resultNodes)
}

func TestRemoveAndAdd(t *testing.T) {
	ch := NewConsistentHash()

	// 添加节点
	ch.Add("server1")
	ch.Add("server2")

	// 获取初始映射
	key := "test_key"
	initialNode, err := ch.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// 移除一个节点
	ch.Remove("server2")

	// 获取新的映射
	newNode, err := ch.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// 添加回节点
	ch.Add("server2")

	t.Logf("Initial mapping for '%s': %s, After removal: %s", key, initialNode, newNode)
}
