package xml

import (
	"fmt"
	"strings"

	"github.com/cn-poe-community/cn-poe-utils/go/api"
	"github.com/cn-poe-community/cn-poe-utils/go/data/pob"
)

var itemNameMap = map[string]string{
	"Doppelgänger Guise": "Doppelganger Guise",
	"Mjölner":            "Mjolner",
}

const pobBaseTypeEnergyBlade = "Energy Blade One Handed"

var baseTypeMap = map[string]string{
	"Maelström Staff": "Maelstrom Staff",
	"Energy Blade":    pobBaseTypeEnergyBlade,
}

// Item 物品
type Item struct {
	ID   int
	JSON *api.Item
}

func NewItem(id int, json *api.Item) *Item {
	return &Item{
		ID:   id,
		JSON: json,
	}
}

// viewModel 返回物品的视图模型
func (i *Item) viewModel() map[string]any {
	model := make(map[string]any)
	json := i.JSON

	model["rarity"] = toPobRarity(json.FrameType)
	model["name"] = json.Name
	model["baseType"] = json.BaseType
	model["typeLine"] = json.TypeLine

	if newName, ok := itemNameMap[json.Name]; ok {
		model["name"] = newName
	}
	if newBaseType, ok := baseTypeMap[json.BaseType]; ok {
		model["baseType"] = newBaseType
	}

	if json.Name == "" {
		typeLine := json.TypeLine
		for baseType, mappedType := range baseTypeMap {
			if strings.Contains(typeLine, baseType) {
				model["typeLine"] = strings.Replace(typeLine, baseType, mappedType, 1)
				break
			}
		}
	}

	propMap := make(map[string]*api.Property)
	if json.Properties != nil {
		for i := range json.Properties {
			prop := &json.Properties[i]
			propMap[prop.Name] = prop
		}
	}

	if qualityText, ok := propMap["Quality"]; ok && len(qualityText.Values) > 0 {
		q := 0
		fmt.Sscanf(qualityText.Values[0].Value, "%d", &q)
		model["quality"] = q
	}
	if evasionRating, ok := propMap["Evasion Rating"]; ok && len(evasionRating.Values) > 0 {
		model["evasionRating"] = evasionRating.Values[0].Value
	}
	if energyShield, ok := propMap["Energy Shield"]; ok && len(energyShield.Values) > 0 {
		model["energyShield"] = energyShield.Values[0].Value
	}
	if armour, ok := propMap["Armour"]; ok && len(armour.Values) > 0 {
		model["armour"] = armour.Values[0].Value
	}
	if ward, ok := propMap["Ward"]; ok && len(ward.Values) > 0 {
		model["ward"] = ward.Values[0].Value
	}
	if radius, ok := propMap["Radius"]; ok && len(radius.Values) > 0 {
		model["radius"] = radius.Values[0].Value
	}
	if limitedTo, ok := propMap["Limited to"]; ok && len(limitedTo.Values) > 0 {
		model["limitedTo"] = limitedTo.Values[0].Value
	}

	requireMap := make(map[string]*api.Requirement)
	if json.Requirements != nil {
		for i := range json.Requirements {
			req := &json.Requirements[i]
			requireMap[req.Name] = req
		}
	}
	if requireClass, ok := requireMap["Class:"]; ok && len(requireClass.Values) > 0 {
		model["requireClass"] = requireClass.Values[0].Value
	}

	model["enchantMods"] = flattenMods(json.EnchantMods)
	model["implicitMods"] = flattenMods(json.ImplicitMods)
	model["explicitMods"] = flattenMods(json.ExplicitMods)
	model["craftedMods"] = flattenMods(json.CraftedMods)
	model["fracturedMods"] = flattenMods(json.FracturedMods)
	model["crucibleMods"] = flattenMods(json.CrucibleMods)
	model["mutatedMods"] = flattenMods(json.MutatedMods)

	abyssalSocketCount := 0
	if json.Sockets != nil {
		var builder []string
		for i := 0; i < len(json.Sockets); i++ {
			if i > 0 {
				if json.Sockets[i].Group == json.Sockets[i-1].Group {
					builder = append(builder, "-")
				} else {
					builder = append(builder, " ")
				}
			}
			color := json.Sockets[i].SColour
			builder = append(builder, string(color))
			if color == "A" {
				abyssalSocketCount++
			}
		}
		model["sockets"] = strings.Join(builder, "")
	}

	if model["baseType"] == pobBaseTypeEnergyBlade {
		model["implicitMods"] = nil
		if abyssalSocketCount > 0 {
			model["explicitMods"] = []string{fmt.Sprintf("Has %d Abyssal Sockets", abyssalSocketCount)}
		}
	}

	implicitCount := 0
	if enchantMods, ok := model["enchantMods"].([]string); ok {
		implicitCount += len(enchantMods)
	}
	if implicitMods, ok := model["implicitMods"].([]string); ok {
		implicitCount += len(implicitMods)
	}
	model["implicitCount"] = implicitCount

	if json.Influences != nil {
		if json.Influences.Shaper != nil && *json.Influences.Shaper {
			model["shaper"] = true
		}
		if json.Influences.Elder != nil && *json.Influences.Elder {
			model["elder"] = true
		}
		if json.Influences.Warlord != nil && *json.Influences.Warlord {
			model["warlord"] = true
		}
		if json.Influences.Hunter != nil && *json.Influences.Hunter {
			model["hunter"] = true
		}
		if json.Influences.Crusader != nil && *json.Influences.Crusader {
			model["crusader"] = true
		}
		if json.Influences.Redeemer != nil && *json.Influences.Redeemer {
			model["redeemer"] = true
		}
	}

	if json.ID != nil {
		model["id"] = *json.ID
	}
	if json.Searing != nil && *json.Searing {
		model["searing"] = true
	}
	if json.Tangled != nil && *json.Tangled {
		model["tangled"] = true
	}
	model["ilvl"] = json.Ilvl
	if json.Corrupted != nil && *json.Corrupted {
		model["corrupted"] = true
	}

	return model
}

func flattenMods(mods []string) []string {
	if mods == nil {
		return nil
	}
	var result []string
	for _, mod := range mods {
		result = append(result, strings.Split(mod, "\n")...)
	}
	return result
}

func toPobRarity(frameType int) string {
	if rarity, ok := pob.DefaultData.RarityMap[frameType]; ok {
		return rarity
	}
	return "Normal"
}

// String 返回XML字符串
func (i *Item) String() string {
	builder := []string{}
	model := i.viewModel()

	builder = append(builder, fmt.Sprintf(`<Item id="%d">`, i.ID))
	builder = append(builder, fmt.Sprintf("Rarity: %s", model["rarity"]))
	if name, ok := model["name"].(string); ok && name != "" {
		builder = append(builder, name)
		builder = append(builder, model["baseType"].(string))
	} else {
		builder = append(builder, model["typeLine"].(string))
	}
	if evasionRating, ok := model["evasionRating"].(string); ok {
		builder = append(builder, fmt.Sprintf("Evasion: %s", evasionRating))
	}
	if energyShield, ok := model["energyShield"].(string); ok {
		builder = append(builder, fmt.Sprintf("Energy Shield: %s", energyShield))
	}
	if armour, ok := model["armour"].(string); ok {
		builder = append(builder, fmt.Sprintf("Armour: %s", armour))
	}
	if ward, ok := model["ward"].(string); ok {
		builder = append(builder, fmt.Sprintf("Ward: %s", ward))
	}
	if id, ok := model["id"].(string); ok {
		builder = append(builder, fmt.Sprintf("Unique ID: %s", id))
	}
	if model["shaper"] != nil {
		builder = append(builder, "Shaper Item")
	}
	if model["elder"] != nil {
		builder = append(builder, "Elder Item")
	}
	if model["warlord"] != nil {
		builder = append(builder, "Warlord Item")
	}
	if model["hunter"] != nil {
		builder = append(builder, "Hunter Item")
	}
	if model["crusader"] != nil {
		builder = append(builder, "Crusader Item")
	}
	if model["redeemer"] != nil {
		builder = append(builder, "Redeemer Item")
	}
	if model["searing"] != nil {
		builder = append(builder, "Searing Exarch Item")
	}
	if model["tangled"] != nil {
		builder = append(builder, "Eater of Worlds Item")
	}
	builder = append(builder, fmt.Sprintf("Item Level: %d", model["ilvl"]))
	if quality, ok := model["quality"].(int); ok {
		builder = append(builder, fmt.Sprintf("Quality: %d", quality))
	}
	if sockets, ok := model["sockets"].(string); ok {
		builder = append(builder, fmt.Sprintf("Sockets: %s", sockets))
	}
	if radius, ok := model["radius"].(string); ok {
		builder = append(builder, fmt.Sprintf("Radius: %s", radius))
	}
	if limitedTo, ok := model["limitedTo"].(string); ok {
		builder = append(builder, fmt.Sprintf("Limited to: %s", limitedTo))
	}
	if requireClass, ok := model["requireClass"].(string); ok {
		builder = append(builder, fmt.Sprintf("Requires Class %s", requireClass))
	}
	builder = append(builder, fmt.Sprintf("Implicits: %d", model["implicitCount"]))

	if enchantMods, ok := model["enchantMods"].([]string); ok {
		for _, mod := range enchantMods {
			builder = append(builder, fmt.Sprintf("{crafted}%s", mod))
		}
	}
	if implicitMods, ok := model["implicitMods"].([]string); ok {
		for _, mod := range implicitMods {
			builder = append(builder, mod)
		}
	}
	if explicitMods, ok := model["explicitMods"].([]string); ok {
		for _, mod := range explicitMods {
			builder = append(builder, mod)
		}
	}
	if mutatedMods, ok := model["mutatedMods"].([]string); ok {
		for _, mod := range mutatedMods {
			builder = append(builder, fmt.Sprintf("{mutated}%s", mod))
		}
	}
	if fracturedMods, ok := model["fracturedMods"].([]string); ok {
		for _, mod := range fracturedMods {
			builder = append(builder, fmt.Sprintf("{fractured}%s", mod))
		}
	}
	if craftedMods, ok := model["craftedMods"].([]string); ok {
		for _, mod := range craftedMods {
			builder = append(builder, fmt.Sprintf("{crafted}%s", mod))
		}
	}
	if crucibleMods, ok := model["crucibleMods"].([]string); ok {
		for _, mod := range crucibleMods {
			builder = append(builder, fmt.Sprintf("{crucible}%s", mod))
		}
	}
	if model["corrupted"] != nil {
		builder = append(builder, "Corrupted")
	}
	builder = append(builder, "</Item>")

	return strings.Join(builder, "\n")
}

// Items 物品集合
type Items struct {
	ItemList []*Item
	ItemSet  *ItemSet
}

func NewItems() *Items {
	return &Items{
		ItemList: []*Item{},
		ItemSet:  NewItemSet(),
	}
}

// String 返回XML字符串
func (i *Items) String() string {
	var itemsView []string
	for _, item := range i.ItemList {
		itemsView = append(itemsView, item.String())
	}
	return fmt.Sprintf(`<Items>
%s
%s
</Items>`, strings.Join(itemsView, "\n"), i.ItemSet.String())
}
