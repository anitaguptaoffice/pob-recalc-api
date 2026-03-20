package api

import (
	"encoding/json"
	"os"
	"testing"
)

// TestValue_Serialization 测试 ValuePair 的序列化和反序列化功能
func TestValue_Serialization(t *testing.T) {
	// 测试序列化
	original := Value{Value: "Test Value", Index: 42}
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal ValuePair: %v", err)
	}

	// 验证序列化结果
	expected := `["Test Value",42]`
	if string(jsonData) != expected {
		t.Errorf("Expected %s, got %s", expected, string(jsonData))
	}

	// 测试反序列化
	var parsed Value
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal ValuePair: %v", err)
	}

	// 验证反序列化结果
	if parsed.Value != original.Value || parsed.Index != original.Index {
		t.Errorf("Expected %+v, got %+v", original, parsed)
	}
}

// TestParseItemsJSON 测试 testdata/items.json 是否能解析为 GetItemsResult
func TestParseItemsJSON(t *testing.T) {
	// 读取 JSON 文件
	data, err := os.ReadFile("testdata/items.json")
	if err != nil {
		t.Fatalf("Failed to read items.json: %v", err)
	}

	// 解析 JSON 到 GetItemsResult
	var result GetItemsResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal items.json: %v", err)
	}

	// 验证解析结果
	if len(result.Items) == 0 {
		t.Error("Expected at least one item in the result")
	}
}

// TestParsePassiveSkillsJSON 测试 testdata/passive_skills.json 是否能正确解析为 GetPassiveSkillsResult
func TestParsePassiveSkillsJSON(t *testing.T) {
	// 读取 JSON 文件
	data, err := os.ReadFile("testdata/passive_skills.json")
	if err != nil {
		t.Fatalf("Failed to read passive_skills.json: %v", err)
	}

	// 解析 JSON 到 GetPassiveSkillsResult
	var result GetPassiveSkillsResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal passive_skills.json: %v", err)
	}
}
