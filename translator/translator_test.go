package translator

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cn-poe-community/cn-poe-utils/go/api"
)

func TestInit(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
}

func TestTranslateBasicItem(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	rarity := api.RarityUnique
	invId := "BodyArmour"

	// Build a minimal request with one item
	items := &api.GetItemsResult{
		Items: []*api.Item{
			{
				Name:     "冰息",
				BaseType: "精制环甲",
				TypeLine: "冰息 精制环甲",
				Rarity:   &rarity,
				Ilvl:     80,
				Identified: true,
				ExplicitMods: []string{
					"+110 最大生命",
					"30% 冰冷抗性",
				},
				InventoryId: &invId,
			},
		},
	}

	passiveSkills := &api.GetPassiveSkillsResult{
		Hashes: []int{},
	}

	req := &TranslateRequest{
		Items:         items,
		PassiveSkills: passiveSkills,
	}

	result, err := Translate(req)
	if err != nil {
		t.Fatalf("Translate() failed: %v", err)
	}

	if result.XML == "" {
		t.Fatal("Expected non-empty XML output")
	}

	// The XML should contain PathOfBuilding root element
	if !strings.Contains(result.XML, "<PathOfBuilding>") {
		t.Error("XML should contain <PathOfBuilding> root element")
	}

	t.Logf("Generated XML length: %d bytes", len(result.XML))
	maxLen := len(result.XML)
	if maxLen > 500 {
		maxLen = 500
	}
	t.Logf("XML preview:\n%s", result.XML[:maxLen])
}

func TestTranslateItemsJSON(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	req := map[string]interface{}{
		"items": map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"name":        "冰息",
					"baseType":    "精制环甲",
					"typeLine":    "冰息 精制环甲",
					"rarity":      "Unique",
					"ilvl":        80,
					"identified":  true,
					"inventoryId": "BodyArmour",
					"frameType":   3,
					"icon":        "",
					"w":           2,
					"h":           3,
					"explicitMods": []string{
						"+110 最大生命",
					},
				},
			},
		},
		"passiveSkills": map[string]interface{}{
			"hashes": []int{},
		},
	}

	jsonData, _ := json.Marshal(req)
	xmlStr, err := TranslateItemsJSON(jsonData)
	if err != nil {
		t.Fatalf("TranslateItemsJSON() failed: %v", err)
	}

	if xmlStr == "" {
		t.Fatal("Expected non-empty XML output")
	}

	t.Logf("XML from JSON: %d bytes", len(xmlStr))
}

// TestTranslateItem verifies the single-item translation path:
// api.Item JSON → jsonTr.TransItem() → pobxml.NewItem() → POB text
// This covers: api.Item struct fields, TransItem(), NewItem(), detectSlotFromItem()
func TestTranslateItem(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// A Chinese unique item JSON (as returned by CN trade API)
	itemJSON := []byte(`{
		"name": "猎首",
		"baseType": "皮革腰带",
		"typeLine": "猎首 皮革腰带",
		"ilvl": 80,
		"identified": true,
		"frameType": 3,
		"icon": "https://poe.game.qq.com/gen/image/WzI1LDE0LHsiZiI6IjJESXRlbXMvQmVsdHMvSGVhZGh1bnRlciIsInciOjIsImgiOjEsInNjYWxlIjoxfV0/abc/headhunter.png",
		"w": 2, "h": 1,
		"explicitMods": [
			"+25 敏捷",
			"+40 最大生命",
			"+60 最大生命",
			"对稀有怪物的击中伤害提高42%",
			"击杀稀有怪物时，你获得其词缀20秒"
		]
	}`)

	result, err := TranslateItem(itemJSON)
	if err != nil {
		t.Fatalf("TranslateItem() failed: %v", err)
	}

	if result.ItemText == "" {
		t.Fatal("Expected non-empty item text")
	}

	// The translated text should contain English mod text
	if !strings.Contains(result.ItemText, "Life") && !strings.Contains(result.ItemText, "Dexterity") {
		t.Errorf("Translated text should contain English mods, got:\n%s", result.ItemText)
	}

	// Slot should be detected from icon URL
	if result.Slot != "Belt" {
		t.Errorf("Expected slot 'Belt', got %q", result.Slot)
	}

	t.Logf("TranslateItem result: slot=%s, text length=%d", result.Slot, len(result.ItemText))
	t.Logf("Item text preview:\n%s", result.ItemText)
}

// TestConvertItem verifies the English item conversion path (no translation):
// api.Item JSON → pobxml.NewItem() → POB text
// This covers: api.Item struct, NewItem() with English data, detectSlotFromItem()
func TestConvertItem(t *testing.T) {
	// An English unique item JSON (as returned by international trade API)
	itemJSON := []byte(`{
		"name": "Headhunter",
		"baseType": "Leather Belt",
		"typeLine": "Headhunter Leather Belt",
		"ilvl": 80,
		"identified": true,
		"frameType": 3,
		"icon": "https://web.poecdn.com/image/Art/2DItems/Belts/Headhunter.png",
		"w": 2, "h": 1,
		"explicitMods": [
			"+25 to Dexterity",
			"+40 to maximum Life",
			"+60 to maximum Life",
			"42% increased Damage with Hits against Rare monsters",
			"When you Kill a Rare monster, you gain its Modifiers for 20 seconds"
		]
	}`)

	result, err := ConvertItem(itemJSON)
	if err != nil {
		t.Fatalf("ConvertItem() failed: %v", err)
	}

	if result.ItemText == "" {
		t.Fatal("Expected non-empty item text")
	}

	// Should contain the English name directly
	if !strings.Contains(result.ItemText, "Headhunter") {
		t.Errorf("Item text should contain 'Headhunter', got:\n%s", result.ItemText)
	}

	if result.Slot != "Belt" {
		t.Errorf("Expected slot 'Belt', got %q", result.Slot)
	}

	t.Logf("ConvertItem result: slot=%s, text length=%d", result.Slot, len(result.ItemText))
}

// TestTranslateItemRareWithMods tests a rare item with multiple mod types.
// This exercises more api.Item fields (implicitMods, craftedMods, fracturedMods).
func TestTranslateItemRareWithMods(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	itemJSON := []byte(`{
		"name": "幽魂之牙",
		"baseType": "翠玉护身符",
		"typeLine": "幽魂之牙 翠玉护身符",
		"ilvl": 85,
		"identified": true,
		"frameType": 2,
		"icon": "https://poe.game.qq.com/gen/image/WzI1LDE0LHsiZiI6IjJESXRlbXMvQW11bGV0cy9KYWRlQW11bGV0IiwidyI6MSwiaCI6MSwic2NhbGUiOjF9XQ/abc/jade.png",
		"w": 1, "h": 1,
		"implicitMods": ["+26 敏捷"],
		"explicitMods": [
			"+75 最大生命",
			"+30% 全域暴击伤害加成"
		]
	}`)

	result, err := TranslateItem(itemJSON)
	if err != nil {
		t.Fatalf("TranslateItem() failed: %v", err)
	}

	if result.ItemText == "" {
		t.Fatal("Expected non-empty item text")
	}

	if result.Slot != "Amulet" {
		t.Errorf("Expected slot 'Amulet', got %q", result.Slot)
	}

	t.Logf("Rare item: slot=%s, length=%d\n%s", result.Slot, len(result.ItemText), result.ItemText)
}
