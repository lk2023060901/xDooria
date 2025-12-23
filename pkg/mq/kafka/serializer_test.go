package kafka

import (
	"testing"
)

type testMessage struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

func TestJSONSerializer(t *testing.T) {
	s := JSONSerializer

	// 测试序列化
	msg := &testMessage{
		ID:      "123",
		Content: "hello",
	}

	data, err := s.Serialize(msg)
	if err != nil {
		t.Fatalf("serialize error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty data")
	}

	// 测试反序列化
	var result testMessage
	err = s.Deserialize(data, &result)
	if err != nil {
		t.Fatalf("deserialize error: %v", err)
	}

	if result.ID != msg.ID {
		t.Errorf("expected ID %s, got %s", msg.ID, result.ID)
	}
	if result.Content != msg.Content {
		t.Errorf("expected Content %s, got %s", msg.Content, result.Content)
	}
}

func TestRawSerializer(t *testing.T) {
	s := RawSerializer

	// 测试序列化
	data := []byte("raw data")
	result, err := s.Serialize(data)
	if err != nil {
		t.Fatalf("serialize error: %v", err)
	}

	if string(result) != string(data) {
		t.Errorf("expected %s, got %s", string(data), string(result))
	}

	// 测试反序列化
	var output []byte
	err = s.Deserialize(data, &output)
	if err != nil {
		t.Fatalf("deserialize error: %v", err)
	}

	if string(output) != string(data) {
		t.Errorf("expected %s, got %s", string(data), string(output))
	}
}

func TestJSONSerializer_InvalidData(t *testing.T) {
	s := JSONSerializer

	// 测试无效 JSON 反序列化
	invalidData := []byte("not valid json")
	var result testMessage
	err := s.Deserialize(invalidData, &result)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestJSONSerializer_NilInput(t *testing.T) {
	s := JSONSerializer

	// 测试 nil 输入序列化
	data, err := s.Serialize(nil)
	if err != nil {
		t.Fatalf("serialize nil error: %v", err)
	}

	// JSON null
	if string(data) != "null" {
		t.Errorf("expected 'null', got %s", string(data))
	}
}

func TestRawSerializer_String(t *testing.T) {
	s := RawSerializer

	// 测试字符串序列化
	str := "hello world"
	data, err := s.Serialize(str)
	if err != nil {
		t.Fatalf("serialize string error: %v", err)
	}

	if string(data) != str {
		t.Errorf("expected %s, got %s", str, string(data))
	}

	// 测试字符串反序列化
	var output string
	err = s.Deserialize(data, &output)
	if err != nil {
		t.Fatalf("deserialize string error: %v", err)
	}

	if output != str {
		t.Errorf("expected %s, got %s", str, output)
	}
}

func TestJSONSerializer_ContentType(t *testing.T) {
	s := JSONSerializer
	if s.ContentType() != "application/json" {
		t.Errorf("expected content type application/json, got %s", s.ContentType())
	}
}

func TestRawSerializer_ContentType(t *testing.T) {
	s := RawSerializer
	if s.ContentType() != "application/octet-stream" {
		t.Errorf("expected content type application/octet-stream, got %s", s.ContentType())
	}
}

func TestProtoSerializer_ContentType(t *testing.T) {
	s := ProtoSerializer
	if s.ContentType() != "application/protobuf" {
		t.Errorf("expected content type application/protobuf, got %s", s.ContentType())
	}
}
