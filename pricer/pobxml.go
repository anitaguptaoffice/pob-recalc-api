package pricer

import (
	"encoding/xml"
	"fmt"
	"html"
	"log"
	"strconv"
	"strings"
)

// ---- POB XML structures for unmarshaling ----

type pobXML struct {
	XMLName xml.Name      `xml:"PathOfBuilding"`
	Build   pobBuild      `xml:"Build"`
	Items   pobItems      `xml:"Items"`
	Skills  pobSkillsWrap `xml:"Skills"`
}

type pobBuild struct {
	Level          int    `xml:"level,attr"`
	ClassName      string `xml:"className,attr"`
	AscendClass    string `xml:"ascendClassName,attr"`
}

type pobItems struct {
	ItemList []pobItemRaw `xml:"Item"`
	ItemSets []pobItemSet `xml:"ItemSet"`
}

type pobItemRaw struct {
	ID      int    `xml:"id,attr"`
	Content string `xml:",chardata"` // The POB text content between <Item> tags
}

type pobItemSet struct {
	ID    int       `xml:"id,attr"`
	Slots []pobSlot `xml:"Slot"`
}

type pobSlot struct {
	Name   string `xml:"name,attr"`
	ItemID int    `xml:"itemId,attr"`
}

type pobSkillsWrap struct {
	SkillSets []pobSkillSet `xml:"SkillSet"`
}

type pobSkillSet struct {
	ID     int        `xml:"id,attr"`
	Skills []pobSkill `xml:"Skill"`
}

type pobSkill struct {
	Slot    string   `xml:"slot,attr"`
	Enabled string   `xml:"enabled,attr"`
	Gems    []pobGem `xml:"Gem"`
}

type pobGem struct {
	NameSpec string `xml:"nameSpec,attr"`
	SkillID  string `xml:"skillId,attr"`
	Level    int    `xml:"level,attr"`
	Quality  int    `xml:"quality,attr"`
	Enabled  string `xml:"enabled,attr"`
}

// ParsePOBXML parses POB XML bytes and extracts items + gems.
func ParsePOBXML(xmlData []byte) (items []POBItem, gems []POBGem, buildInfo *pobBuild, err error) {
	var pob pobXML
	if err := xml.Unmarshal(xmlData, &pob); err != nil {
		return nil, nil, nil, fmt.Errorf("unmarshal POB XML: %w", err)
	}

	buildInfo = &pob.Build

	// Build item ID → slot mapping from the first ItemSet
	slotMap := make(map[int]string)
	if len(pob.Items.ItemSets) > 0 {
		for _, slot := range pob.Items.ItemSets[0].Slots {
			if slot.ItemID > 0 {
				slotMap[slot.ItemID] = slot.Name
			}
		}
	}

	// Parse each item
	for _, raw := range pob.Items.ItemList {
		item, parseErr := parsePOBItemText(raw.ID, raw.Content)
		if parseErr != nil {
			log.Printf("[pricer] Warning: failed to parse item %d: %v", raw.ID, parseErr)
			continue
		}
		// Assign slot from ItemSet
		if slot, ok := slotMap[raw.ID]; ok {
			item.Slot = slot
		}
		items = append(items, *item)
	}

	// Parse gems from Skills
	for _, skillSet := range pob.Skills.SkillSets {
		for _, skill := range skillSet.Skills {
			for _, g := range skill.Gems {
				gem := POBGem{
					NameSpec: html.UnescapeString(g.NameSpec),
					SkillID:  g.SkillID,
					Level:    g.Level,
					Quality:  g.Quality,
					Slot:     skill.Slot,
					Enabled:  g.Enabled == "true",
				}
				gems = append(gems, gem)
			}
		}
	}

	log.Printf("[pricer] Parsed POB XML: %d items, %d gems, level=%d class=%s/%s",
		len(items), len(gems), buildInfo.Level, buildInfo.ClassName, buildInfo.AscendClass)

	return items, gems, buildInfo, nil
}

// parsePOBItemText parses a POB item text block like:
//
//	Rarity: UNIQUE
//	Hyrri's Truth
//	Jade Amulet
//	...
func parsePOBItemText(id int, rawText string) (*POBItem, error) {
	// Clean up the text: unescape HTML entities and split lines
	text := html.UnescapeString(rawText)
	lines := strings.Split(strings.TrimSpace(text), "\n")

	// Filter out empty lines and ModRange XML tags
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "<ModRange") || strings.HasPrefix(trimmed, "</ModRange") {
			continue
		}
		cleanLines = append(cleanLines, trimmed)
	}

	if len(cleanLines) < 2 {
		return nil, fmt.Errorf("item text too short: %d lines", len(cleanLines))
	}

	item := &POBItem{ID: id}

	// First line should be "Rarity: XXXX"
	if strings.HasPrefix(cleanLines[0], "Rarity:") {
		item.Rarity = strings.TrimSpace(strings.TrimPrefix(cleanLines[0], "Rarity:"))
	} else {
		return nil, fmt.Errorf("first line is not Rarity: %s", cleanLines[0])
	}

	// For UNIQUE items: line 1 = name, line 2 = base type
	// For RARE items: line 1 = random name, line 2 = base type
	// For MAGIC/NORMAL: line 1 = base type (possibly with prefix/suffix)
	switch strings.ToUpper(item.Rarity) {
	case "UNIQUE", "RARE":
		if len(cleanLines) >= 3 {
			item.Name = cleanLines[1]
			item.BaseType = cleanLines[2]
		} else {
			item.Name = cleanLines[1]
		}
	default:
		// MAGIC, NORMAL — the "name" line is actually the base type
		item.BaseType = cleanLines[1]
	}

	// Skip metadata lines (Unique ID, Item Level, Quality, Sockets, LevelReq, etc.)
	// and collect mods starting after "Implicits: N" line
	inMods := false
	for i := 2; i < len(cleanLines); i++ {
		line := cleanLines[i]
		if strings.HasPrefix(line, "Implicits:") {
			inMods = true
			continue
		}
		if inMods {
			item.Mods = append(item.Mods, line)
		}
	}

	return item, nil
}

// DeduplicateGems removes duplicate gem entries (same nameSpec + level + quality).
// POB XML sometimes has gems listed in both the skills and items sections.
func DeduplicateGems(gems []POBGem) []POBGem {
	type gemKey struct {
		name    string
		level   int
		quality int
	}
	seen := make(map[gemKey]bool)
	var result []POBGem
	for _, g := range gems {
		key := gemKey{g.NameSpec, g.Level, g.Quality}
		if !seen[key] {
			seen[key] = true
			result = append(result, g)
		}
	}
	return result
}

// extractBaseTypeFromMagic attempts to extract the base type from a magic item name.
// Magic items have format like "Quicksilver Flask" or "of Adrenaline Quicksilver Flask".
// For flasks, we try to match known base types.
func extractBaseTypeFromMagic(name string) string {
	// Common flask base types
	flaskBases := []string{
		"Divine Life Flask", "Eternal Life Flask", "Hallowed Life Flask",
		"Sacred Life Flask", "Sanctified Life Flask",
		"Divine Mana Flask", "Eternal Mana Flask",
		"Quicksilver Flask", "Granite Flask", "Jade Flask",
		"Diamond Flask", "Basalt Flask", "Quartz Flask",
		"Amethyst Flask", "Ruby Flask", "Sapphire Flask", "Topaz Flask",
		"Bismuth Flask", "Stibnite Flask", "Sulphur Flask",
		"Silver Flask", "Aquamarine Flask", "Gold Flask",
	}
	for _, base := range flaskBases {
		if strings.Contains(name, base) {
			return base
		}
	}
	return name
}

func intPtr(v int) *int {
	return &v
}

func atoiDefault(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}
