package jsonx

import (
	"testing"
)

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestMarshal(t *testing.T) {
	p := Person{Name: "Alice", Age: 30}

	data, err := Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expected := `{"name":"Alice","age":30}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestUnmarshal(t *testing.T) {
	data := []byte(`{"name":"Bob","age":25}`)

	var p Person
	err := Unmarshal(data, &p)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if p.Name != "Bob" || p.Age != 25 {
		t.Errorf("Expected Name=Bob Age=25, got Name=%s Age=%d", p.Name, p.Age)
	}
}

func TestMarshalString(t *testing.T) {
	p := Person{Name: "Charlie", Age: 35}

	str, err := MarshalString(p)
	if err != nil {
		t.Fatalf("MarshalString failed: %v", err)
	}

	expected := `{"name":"Charlie","age":35}`
	if str != expected {
		t.Errorf("Expected %s, got %s", expected, str)
	}
}

func TestUnmarshalFromString(t *testing.T) {
	str := `{"name":"David","age":40}`

	var p Person
	err := UnmarshalFromString(str, &p)
	if err != nil {
		t.Fatalf("UnmarshalFromString failed: %v", err)
	}

	if p.Name != "David" || p.Age != 40 {
		t.Errorf("Expected Name=David Age=40, got Name=%s Age=%d", p.Name, p.Age)
	}
}

func TestMarshalToBytes(t *testing.T) {
	p := Person{Name: "Eve", Age: 28}

	data, err := MarshalToBytes(p)
	if err != nil {
		t.Fatalf("MarshalToBytes failed: %v", err)
	}

	expected := `{"name":"Eve","age":28}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestUnmarshalFromBytes(t *testing.T) {
	data := []byte(`{"name":"Frank","age":45}`)

	var p Person
	err := UnmarshalFromBytes(data, &p)
	if err != nil {
		t.Fatalf("UnmarshalFromBytes failed: %v", err)
	}

	if p.Name != "Frank" || p.Age != 45 {
		t.Errorf("Expected Name=Frank Age=45, got Name=%s Age=%d", p.Name, p.Age)
	}
}

func TestRoundTrip(t *testing.T) {
	original := Person{Name: "Grace", Age: 32}

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Person
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if original != decoded {
		t.Errorf("Round trip failed: expected %+v, got %+v", original, decoded)
	}
}

func TestRoundTripString(t *testing.T) {
	original := Person{Name: "Henry", Age: 38}

	str, err := MarshalString(original)
	if err != nil {
		t.Fatalf("MarshalString failed: %v", err)
	}

	var decoded Person
	err = UnmarshalFromString(str, &decoded)
	if err != nil {
		t.Fatalf("UnmarshalFromString failed: %v", err)
	}

	if original != decoded {
		t.Errorf("String round trip failed: expected %+v, got %+v", original, decoded)
	}
}
