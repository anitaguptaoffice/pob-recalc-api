package zh2en

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/cn-poe-community/cn-poe-utils/go/data/poe"
)

func loadTestData(t *testing.T) *poe.Data {
	data, err := os.ReadFile("../../data/poe/testdata/all.json")
	if err != nil {
		t.Fatalf("无法读取测试文件: %v", err)
	}

	var result poe.Data
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	return &result
}

func TestTransNameAndBaseType(t *testing.T) {
	data := loadTestData(t)
	translator := NewBasicTranslator(data)

	testCases := []struct {
		name         string
		zhName       string
		zhBaseType   string
		wantName     string
		wantBaseType string
	}{
		{
			name:         "安赛娜丝的安抚之语|丝绸手套",
			zhName:       "安赛娜丝的安抚之语",
			zhBaseType:   "丝绸手套",
			wantName:     "Asenath's Gentle Touch",
			wantBaseType: "Silk Gloves",
		},
		{
			name:         "漆黑天顶|丝绸手套",
			zhName:       "漆黑天顶",
			zhBaseType:   "丝绸手套",
			wantName:     "Black Zenith",
			wantBaseType: "Fingerless Silk Gloves",
		},
		{
			name:         "稀有物品|丝绸手套",
			zhName:       "abc",
			zhBaseType:   "丝绸手套",
			wantName:     "Item",
			wantBaseType: "Fingerless Silk Gloves",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := translator.TransNameAndBaseType(tc.zhName, tc.zhBaseType)
			if result == nil {
				t.Fatalf("TransNameAndBaseType(%q, %q) = nil, want non-nil", tc.zhName, tc.zhBaseType)
			}
			if result.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tc.wantName)
			}
			if result.BaseType != tc.wantBaseType {
				t.Errorf("BaseType = %q, want %q", result.BaseType, tc.wantBaseType)
			}
		})
	}
}

func TestTransMod(t *testing.T) {
	data := loadTestData(t)
	translator := NewBasicTranslator(data)

	testCases := []struct {
		zh string
		en string
	}{
		{
			zh: "增加 8 个天赋技能",
			en: "Adds 8 Passive Skills",
		},
		{
			zh: "所有 电球 宝石等级 +3",
			en: "+3 to Level of all Spark Gems",
		},
		{
			zh: "有一个传奇怪物出现在你面前：法术附加 {0} - {1} 基础物理伤害",
			en: "While a Unique Enemy is in your Presence, Adds {0} to {1} Physical Damage to Spells",
		},
		{
			zh: "有一个异界图鉴最终首领出现在你面前：冰霜净化的光环效果提高 {0}%",
			en: "While a Pinnacle Atlas Boss is in your Presence, Purity of Ice has {0}% increased Aura Effect",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.zh, func(t *testing.T) {
			result := translator.TransMod(tc.zh)
			if result == nil {
				t.Fatalf("TransMod(%q) = nil, want %q", tc.zh, tc.en)
			}
			if *result != tc.en {
				t.Errorf("TransMod(%q) = %q, want %q", tc.zh, *result, tc.en)
			}
		})
	}
}

func TestTransAscendant(t *testing.T) {
	data := loadTestData(t)
	translator := NewBasicTranslator(data)

	testCases := []struct {
		zh string
		en string
	}{
		{
			zh: "自然之怒",
			en: "Fury of Nature",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.zh, func(t *testing.T) {
			result := translator.TransAscendant(tc.zh)
			if result == nil {
				t.Fatalf("TransAscendant(%q) = nil, want %q", tc.zh, tc.en)
			}
			if *result != tc.en {
				t.Errorf("TransAscendant(%q) = %q, want %q", tc.zh, *result, tc.en)
			}
		})
	}
}

func TestFindBaseTypeFromTypeLine(t *testing.T) {
	data := loadTestData(t)
	translator := NewBasicTranslator(data)

	testCases := []struct {
		name         string
		typeLine     string
		itemName     string
		wantBaseType string
	}{
		{
			name:         "魔法物品带修饰词",
			typeLine:     "显著的幼龙之大型星团珠宝",
			itemName:     "",
			wantBaseType: "Large Cluster Jewel",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := translator.FindBaseTypeFromTypeLine(tc.typeLine, tc.itemName)
			if result == nil {
				t.Fatalf("FindBaseTypeFromTypeLine(%q, %q) = nil, want non-nil", tc.typeLine, tc.itemName)
			}
			if result.En != tc.wantBaseType {
				t.Errorf("BaseType.En = %q, want %q", result.En, tc.wantBaseType)
			}
		})
	}
}

func TestTransNameAndTypeLine(t *testing.T) {
	data := loadTestData(t)
	translator := NewBasicTranslator(data)

	testCases := []struct {
		name         string
		zhName       string
		zhTypeLine   string
		wantName     string
		wantTypeLine string
	}{
		{
			name:         "传奇物品 - 漆黑天顶",
			zhName:       "漆黑天顶",
			zhTypeLine:   "丝绸手套",
			wantName:     "Black Zenith",
			wantTypeLine: "Fingerless Silk Gloves",
		},
		{
			name:         "普通物品",
			zhName:       "",
			zhTypeLine:   "丝绸手套",
			wantName:     "",
			wantTypeLine: "Fingerless Silk Gloves",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := translator.TransNameAndTypeLine(tc.zhName, tc.zhTypeLine)
			if result == nil {
				t.Fatalf("TransNameAndTypeLine(%q, %q) = nil, want non-nil", tc.zhName, tc.zhTypeLine)
			}
			if result.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tc.wantName)
			}
			if result.TypeLine != tc.wantTypeLine {
				t.Errorf("TypeLine = %q, want %q", result.TypeLine, tc.wantTypeLine)
			}
		})
	}
}
