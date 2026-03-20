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
