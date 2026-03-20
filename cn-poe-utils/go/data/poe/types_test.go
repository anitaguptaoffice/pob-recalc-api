package poe

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDataSerialization(t *testing.T) {
	// 读取测试文件
	data, err := os.ReadFile("testdata/all.json")
	if err != nil {
		t.Fatalf("无法读取测试文件: %v", err)
	}

	// 反序列化到Data类型
	var result Data
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
}
