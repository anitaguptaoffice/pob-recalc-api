package zh2en

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/cn-poe-community/cn-poe-utils/go/api"
	"github.com/cn-poe-community/cn-poe-utils/go/data/poe"
)

func loadTestDataForJson(t *testing.T) *poe.Data {
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

func TestCrucibleModTranslation(t *testing.T) {
	data := loadTestDataForJson(t)
	basicTranslator := NewBasicTranslator(data)
	jsonTranslator := NewJsonTranslator(basicTranslator)

	itemJSON := `{
		"name": "恐慌 影弦",
		"typeLine": "脊弓",
		"baseType": "脊弓",
		"ilvl": 75,
		"crucibleMods": [
			"该装备附加 30 - 47 基础火焰伤害",
			"攻击速度减慢 6%",
			"暴击几率提高 +1.2%",
			"-500 命中值"
		],
		"frameType": 2,
		"inventoryId": "Weapon"
	}`

	var item api.Item
	if err := json.Unmarshal([]byte(itemJSON), &item); err != nil {
		t.Fatalf("无法反序列化物品: %v", err)
	}

	jsonTranslator.TransItem(&item)

	expected := "+1.2% to Critical Strike Chance"
	if item.CrucibleMods[2] != expected {
		t.Errorf("CrucibleMods[2] = %q, want %q", item.CrucibleMods[2], expected)
	}
}

func TestForbiddenJewelsTranslation(t *testing.T) {
	data := loadTestDataForJson(t)
	basicTranslator := NewBasicTranslator(data)
	jsonTranslator := NewJsonTranslator(basicTranslator)

	itemJSON := `{
		"verified": false,
		"w": 1,
		"h": 1,
		"icon": "https://poecdn.game.qq.com/gen/image/WzI1LDE0LHsiZiI6IjJESXRlbXMvSmV3ZWxzL1B1enpsZVBpZWNlSmV3ZWxfR3JlYXRUYW5nbGUiLCJ3IjoxLCJoIjoxLCJzY2FsZSI6MX1d/9035b9ffd4/PuzzlePieceJewel_GreatTangle.png",
		"league": "S22赛季",
		"id": "0df33223c46c6ac81c38fa4683e4f97b74f7f812c7fffa43153c844e6372d36a",
		"name": "禁断之肉",
		"typeLine": "钴蓝珠宝",
		"baseType": "钴蓝珠宝",
		"identified": true,
		"ilvl": 86,
		"corrupted": true,
		"properties": [
			{
				"name": "仅限",
				"values": [["1", 0]],
				"displayMode": 0
			}
		],
		"requirements": [
			{
				"name": "职业：",
				"values": [["女巫", 0]],
				"displayMode": 0,
				"type": 57
			}
		],
		"explicitMods": ["禁断之火上有匹配的词缀则配置 邪恶君王"],
		"descrText": "放置到一个天赋树的珠宝插槽中以产生效果。右键点击以移出插槽。",
		"flavourText": ["被纠缠之主包裹的肉体们\r", "在永无止尽的融合中哭喊着救命……"],
		"frameType": 3,
		"x": 56,
		"y": 0,
		"inventoryId": "PassiveJewels"
	}`

	var item api.Item
	if err := json.Unmarshal([]byte(itemJSON), &item); err != nil {
		t.Fatalf("无法反序列化物品: %v", err)
	}

	jsonTranslator.TransItem(&item)

	// 检查 Properties 翻译
	if item.Properties[0].Name != "Limited to" {
		t.Errorf("Properties[0].Name = %q, want %q", item.Properties[0].Name, "Limited to")
	}

	// 检查 Requirements 翻译
	if item.Requirements[0].Name != "Class:" {
		t.Errorf("Requirements[0].Name = %q, want %q", item.Requirements[0].Name, "Class:")
	}

	// 检查 Requirement Values 翻译
	if item.Requirements[0].Values[0].Value != "Witch" {
		t.Errorf("Requirements[0].Values[0].Value = %q, want %q", item.Requirements[0].Values[0].Value, "Witch")
	}
}
