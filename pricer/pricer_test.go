package pricer

import (
	"testing"

	"pob_api/translator"
)

// TestNewBuildCostCalculator verifies that the pricer can be initialized
// with the current cn-poe-utils poe.Data structure. This catches breaking
// changes in field names (Amulets, Belts, BaseType.En/Zh, Skill.En/Zh, etc.)
func TestNewBuildCostCalculator(t *testing.T) {
	if err := translator.Init(); err != nil {
		t.Fatalf("translator.Init() failed: %v", err)
	}

	calc, err := NewBuildCostCalculator()
	if err != nil {
		t.Fatalf("NewBuildCostCalculator() failed: %v", err)
	}

	// Verify maps are populated (not empty)
	if len(calc.uniqueEnToZh) == 0 {
		t.Error("uniqueEnToZh is empty — poe.Data.*.Uniques may have changed")
	}
	if len(calc.baseTypeEnToZh) == 0 {
		t.Error("baseTypeEnToZh is empty — poe.Data base type fields may have changed")
	}
	if len(calc.gemEnToZh) == 0 {
		t.Error("gemEnToZh is empty — poe.Data skill fields may have changed")
	}

	t.Logf("Reverse maps: %d uniques, %d base types, %d gems",
		len(calc.uniqueEnToZh), len(calc.baseTypeEnToZh), len(calc.gemEnToZh))

	// Spot-check a few well-known items that should always exist
	spotChecks := map[string]map[string]string{
		"uniqueEnToZh": {
			"Headhunter": "",  // just check key exists
			"Tabula Rasa": "", // classic unique
		},
		"baseTypeEnToZh": {
			"Jade Amulet":     "",
			"Leather Belt":    "",
			"Simple Robe":     "",
			"Cobalt Jewel":    "",
		},
		"gemEnToZh": {
			"Cyclone":      "",
			"Tornado Shot": "",
		},
	}

	for mapName, checks := range spotChecks {
		var m map[string]string
		switch mapName {
		case "uniqueEnToZh":
			m = calc.uniqueEnToZh
		case "baseTypeEnToZh":
			m = calc.baseTypeEnToZh
		case "gemEnToZh":
			m = calc.gemEnToZh
		}
		for key := range checks {
			if zh, ok := m[key]; !ok {
				t.Errorf("%s: missing expected key %q", mapName, key)
			} else if zh == "" {
				t.Errorf("%s[%q]: Chinese name is empty", mapName, key)
			} else {
				t.Logf("%s[%q] = %q ✓", mapName, key, zh)
			}
		}
	}
}

// TestParsePOBXMLItems verifies POB XML item parsing still works.
func TestParsePOBXMLItems(t *testing.T) {
	// Minimal POB XML matching the actual structure ParsePOBXML expects
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<PathOfBuilding>
<Build level="90" className="Ranger" ascendClassName="Deadeye">
</Build>
<Items>
<Item id="1">
Rarity: UNIQUE
Headhunter
Leather Belt
Unique ID: abc123
Item Level: 80
LevelReq: 40
+25 to Dexterity
+40 to maximum Life
</Item>
<ItemSet id="1">
<Slot name="Belt" itemId="1" />
</ItemSet>
</Items>
<Skills>
<SkillSet id="1">
<Skill mainActiveSkillCalcs="1" slot="" label="" enabled="true">
<Gem level="21" quality="20" skillId="Cyclone" nameSpec="Cyclone" enabled="true" />
</Skill>
</SkillSet>
</Skills>
<Tree activeSpec="1">
<Spec treeVersion="3_28" classId="4" ascendClassId="3" nodes="1,2,3">
<URL>https://example.com</URL>
</Spec>
</Tree>
</PathOfBuilding>`)

	items, gems, _, err := ParsePOBXML(xmlData)
	if err != nil {
		t.Fatalf("ParsePOBXML() failed: %v", err)
	}

	if len(items) == 0 {
		t.Fatal("Expected at least 1 item")
	}
	if items[0].Name != "Headhunter" {
		t.Errorf("Expected item name 'Headhunter', got %q", items[0].Name)
	}
	if items[0].BaseType != "Leather Belt" {
		t.Errorf("Expected base type 'Leather Belt', got %q", items[0].BaseType)
	}
	if items[0].Slot != "Belt" {
		t.Errorf("Expected slot 'Belt', got %q", items[0].Slot)
	}

	if len(gems) == 0 {
		t.Fatal("Expected at least 1 gem")
	}
	if gems[0].NameSpec != "Cyclone" {
		t.Errorf("Expected gem 'Cyclone', got %q", gems[0].NameSpec)
	}
}
