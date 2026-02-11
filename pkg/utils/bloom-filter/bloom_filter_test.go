package bloomfilter

import (
	"testing"
)

func TestBloomFilter_Basic(t *testing.T) {
	bf := New(1000, 0.01)

	bf.AddString("hello")
	bf.AddString("world")

	if !bf.ContainsString("hello") {
		t.Error("Expected to find 'hello' in bloom filter")
	}

	if !bf.ContainsString("world") {
		t.Error("Expected to find 'world' in bloom filter")
	}

	if bf.ContainsString("notexist") {
		t.Error("Expected not to find 'notexist' in bloom filter")
	}
}

func TestBloomFilter_Bytes(t *testing.T) {
	bf := New(1000, 0.01)

	data := []byte("test data")
	bf.Add(data)

	if !bf.Contains(data) {
		t.Error("Expected to find data in bloom filter")
	}

	if bf.Contains([]byte("other data")) {
		t.Error("Expected not to find other data in bloom filter")
	}
}

func TestBloomFilter_TestAndAdd(t *testing.T) {
	bf := New(1000, 0.01)

	exists := bf.TestAndAddString("test")
	if exists {
		t.Error("Expected 'test' to not exist initially")
	}

	exists = bf.TestAndAddString("test")
	if !exists {
		t.Error("Expected 'test' to exist after adding")
	}
}

func TestBloomFilter_Clear(t *testing.T) {
	bf := New(1000, 0.01)

	bf.AddString("test")
	bf.Clear()

	if bf.ContainsString("test") {
		t.Error("Expected bloom filter to be empty after clear")
	}
}

func TestBloomFilter_New(t *testing.T) {
	bf := New(1000, 0.01)
	if bf == nil {
		t.Error("Expected bloom filter to be created")
	}

	if bf.filter == nil {
		t.Error("Expected internal filter to be initialized")
	}
}
