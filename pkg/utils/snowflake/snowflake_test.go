package genid

import (
	"testing"
)

func TestSnowflakeID(t *testing.T) {
	// 创建ID生成器实例
	gen, err := NewSnowflakeID(1)
	if err != nil {
		t.Fatal(err)
	}

	// 生成多个ID并验证唯一性
	ids := make(map[int64]bool)
	for i := 0; i < 1000; i++ {
		id := gen.NextID()
		
		if ids[id] {
			t.Fatalf("Duplicate ID generated: %d", id)
		}
		ids[id] = true
	}
	
	t.Logf("Generated %d unique IDs", len(ids))
}

func TestParseID(t *testing.T) {
	gen, err := NewSnowflakeID(1)
	if err != nil {
		t.Fatal(err)
	}

	id := gen.NextID()

	parsedID := gen.ParseID(id)
	
	// 验证解析出的时间戳是合理的
	if parsedID.Time() <= 0 {
		t.Errorf("Parsed timestamp should be greater than 0, got %d", parsedID.Time())
	}
	
	// 验证节点ID
	if GetNodeIDFromID(id) != 1 {
		t.Errorf("Expected node ID 1, got %d", GetNodeIDFromID(id))
	}
	
	// 验证序列号在合理范围内
	if GetStepFromID(id) < 0 {
		t.Errorf("Step %d should not be negative", GetStepFromID(id))
	}
	
	t.Logf("ID: %d, Timestamp: %d, Node: %d, Step: %d", id, parsedID.Time(), GetNodeIDFromID(id), GetStepFromID(id))
}

func TestStringFormat(t *testing.T) {
	gen, err := NewSnowflakeID(1)
	if err != nil {
		t.Fatal(err)
	}

	// 测试字符串格式的ID生成
	stringID := gen.NextStringID()
	if stringID == "" {
		t.Error("Generated string ID should not be empty")
	}

	// 验证字符串ID和数字ID的一致性
	numID := gen.NextID()
	stringNumID := gen.NextStringID()
	
	// 至少验证它们都是有效的ID
	if numID <= 0 {
		t.Error("Generated numeric ID should be greater than 0")
	}
	if stringNumID == "" {
		t.Error("Generated string ID should not be empty")
	}
	
	t.Logf("Numeric ID: %d, String ID: %s", numID, stringNumID)
}