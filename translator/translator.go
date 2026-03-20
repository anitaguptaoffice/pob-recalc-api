// Package translator provides Chinese-to-English translation for POE items
// using cn-poe-utils translation data and building utilities.
package translator

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cn-poe-community/cn-poe-utils/go/api"
	"github.com/cn-poe-community/cn-poe-utils/go/building"
	pobxml "github.com/cn-poe-community/cn-poe-utils/go/building/xml"
	"github.com/cn-poe-community/cn-poe-utils/go/data/poe"
	"github.com/cn-poe-community/cn-poe-utils/go/translator/zh2en"
)

//go:embed all.json
var allJSON []byte

var (
	initOnce        sync.Once
	globalPoeData   poe.Data
	globalBasicTr   *zh2en.BasicTranslator
	globalJSONTr    *zh2en.JsonTranslator
	initErr         error
)

// Init initializes the translation engine. It's safe to call multiple times;
// actual initialization happens only once. Returns any initialization error.
func Init() error {
	initOnce.Do(func() {
		if err := json.Unmarshal(allJSON, &globalPoeData); err != nil {
			initErr = fmt.Errorf("failed to parse translation data: %w", err)
			return
		}
		globalBasicTr = zh2en.NewBasicTranslator(&globalPoeData)
		globalJSONTr = zh2en.NewJsonTranslator(globalBasicTr)
	})
	return initErr
}

// TranslateRequest represents the input for translation: items and passive skills
// from the POE (Chinese) API.
type TranslateRequest struct {
	Items         *api.GetItemsResult         `json:"items"`
	PassiveSkills *api.GetPassiveSkillsResult `json:"passiveSkills"`
}

// TranslateResult contains the translated POB XML string.
type TranslateResult struct {
	XML string `json:"xml"`
}

// Translate takes Chinese POE API data (items + passive skills) and returns
// a POB-compatible XML string with all text translated to English.
func Translate(req *TranslateRequest) (*TranslateResult, error) {
	if err := Init(); err != nil {
		return nil, fmt.Errorf("translator not initialized: %w", err)
	}

	if req.Items == nil {
		return nil, fmt.Errorf("items is required")
	}
	if req.PassiveSkills == nil {
		return nil, fmt.Errorf("passiveSkills is required")
	}

	// Step 1: Translate items and passive skills (in-place modification)
	globalJSONTr.TransItems(req.Items)
	globalJSONTr.TransPassiveSkills(req.PassiveSkills)

	// Step 2: Transform to POB XML
	pob := building.Transform(req.Items, req.PassiveSkills, &building.TransformOptions{})
	xmlStr := pob.String()

	return &TranslateResult{XML: xmlStr}, nil
}

// TranslateItemsJSON takes raw JSON bytes of the translate request and returns
// the POB XML string. This is a convenience wrapper for HTTP handlers.
func TranslateItemsJSON(jsonData []byte) (string, error) {
	var req TranslateRequest
	if err := json.Unmarshal(jsonData, &req); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	result, err := Translate(&req)
	if err != nil {
		return "", err
	}

	return result.XML, nil
}

// TranslateItemResult contains the translated item text and detected slot name.
type TranslateItemResult struct {
	ItemText string `json:"item_text"`
	Slot     string `json:"slot"`
}

// TranslateItem translates a single Chinese item JSON into English POB item text.
// It accepts the raw item JSON from the POE trade API (the "item" field from a fetch result).
// Returns the POB-format item text and the detected equipment slot name.
func TranslateItem(itemJSON []byte) (*TranslateItemResult, error) {
	if err := Init(); err != nil {
		return nil, fmt.Errorf("translator not initialized: %w", err)
	}

	var item api.Item
	if err := json.Unmarshal(itemJSON, &item); err != nil {
		return nil, fmt.Errorf("invalid item JSON: %w", err)
	}

	// Translate the item in-place
	globalJSONTr.TransItem(&item)

	// Generate POB item text using the XML item builder
	xmlItem := pobxml.NewItem(1, &item)
	itemStr := xmlItem.String()

	// Strip the XML wrapper tags: <Item id="1"> ... </Item>
	// The item text is between the first and last lines
	lines := splitLines(itemStr)
	if len(lines) >= 3 {
		// Remove first line (<Item id="1">) and last line (</Item>)
		itemStr = joinLines(lines[1 : len(lines)-1])
	}

	// Detect slot from item category
	slot := detectSlotFromItem(&item)

	return &TranslateItemResult{
		ItemText: itemStr,
		Slot:     slot,
	}, nil
}

// detectSlotFromItem tries to determine the POB equipment slot from the item's icon URL
// or other properties. This is a best-effort detection for trade items which lack inventoryId.
func detectSlotFromItem(item *api.Item) string {
	icon := item.Icon

	// Detect from icon URL patterns (trade API items have descriptive icon URLs)
	iconSlotMap := map[string]string{
		"/Helmets/":     "Helmet",
		"/BodyArmours/": "Body Armour",
		"/Gloves/":      "Gloves",
		"/Boots/":       "Boots",
		"/Shields/":     "Weapon 2",
		"/Amulets/":     "Amulet",
		"/Rings/":       "Ring 1",
		"/Belts/":       "Belt",
		"/Quivers/":     "Weapon 2",
		"/Flasks/":      "Flask 1",
		"/Jewels/":      "Jewel 1",
		// Weapons → Weapon 1
		"/OneHandWeapons/":  "Weapon 1",
		"/TwoHandWeapons/":  "Weapon 1",
		"/Daggers/":         "Weapon 1",
		"/Claws/":           "Weapon 1",
		"/Wands/":           "Weapon 1",
		"/Sceptres/":        "Weapon 1",
	}

	for pattern, slot := range iconSlotMap {
		if containsStr(icon, pattern) {
			return slot
		}
	}

	// Fallback: try baseType patterns
	baseType := item.BaseType
	baseTypeSlotMap := map[string]string{
		"Ring":   "Ring 1",
		"Amulet": "Amulet",
		"Belt":   "Belt",
		"Quiver": "Weapon 2",
		"Shield": "Weapon 2",
		"Flask":  "Flask 1",
		"Jewel":  "Jewel 1",
	}
	for pattern, slot := range baseTypeSlotMap {
		if containsStr(baseType, pattern) {
			return slot
		}
	}

	return "Weapon 1" // fallback
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}
