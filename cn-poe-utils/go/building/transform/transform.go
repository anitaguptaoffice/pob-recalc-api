package transform

import (
	"fmt"

	"github.com/cn-poe-community/cn-poe-utils/go/api"
	"github.com/cn-poe-community/cn-poe-utils/go/building/xml"
)

// TransformOptions 转换选项
type TransformOptions struct {
	SkipWeapon2 bool
}

// Transformer 转换器
type Transformer struct {
	itemsData         *api.GetItemsResult
	passiveSkillsData *api.GetPassiveSkillsResult
	building          *xml.PathOfBuilding
	itemIdGenerator   int
	options           *TransformOptions
}

func NewTransformer(
	itemsData *api.GetItemsResult,
	passiveSkillsData *api.GetPassiveSkillsResult,
	options *TransformOptions,
) *Transformer {
	return &Transformer{
		itemsData:         itemsData,
		passiveSkillsData: passiveSkillsData,
		itemIdGenerator:   1,
		options:           options,
	}
}

// Transform 执行转换
func (t *Transformer) Transform() {
	building := xml.NewPathOfBuilding()
	t.building = building
	t.itemIdGenerator = 1

	// 填充build
	build := building.Build
	character := &t.itemsData.Character
	build.Level = character.Level

	build.ClassName = GetCharacterName(t.passiveSkillsData.Character)
	build.AscendClassName = GetAscendancyName(
		t.passiveSkillsData.Character,
		t.passiveSkillsData.Ascendancy,
	)

	// 解析json
	t.parseItems()
	t.parseTree()
}

// GetBuilding 获取构建结果
func (t *Transformer) GetBuilding() *xml.PathOfBuilding {
	return t.building
}

func (t *Transformer) parseItems() {
	building := t.building

	itemDataArray := t.getBuildingItemDataArray()
	for _, data := range itemDataArray {
		item := xml.NewItem(t.itemIdGenerator, data)
		building.Items.ItemList = append(building.Items.ItemList, item)
		t.itemIdGenerator++

		slotSet := building.Items.ItemSet
		slotName, err := GetSlotName(data)
		if err != nil {
			panic(fmt.Sprintf("%s %d %v", *data.InventoryId, *data.X, slotName))
		}
		slot := xml.NewEquipmentSlot(slotName, item.ID)
		slotSet.Append(slot)

		if len(data.Sockets) > 0 && len(data.SocketedItems) > 0 {
			sockets := data.Sockets
			socketedItems := data.SocketedItems

			var group []*api.Item
			var prevGroupNum int
			skills := building.Skills.SkillSet.Skills
			abyssJewelCount := 0

			for i := range socketedItems {
				si := socketedItems[i]
				if si.AbyssJewel != nil && *si.AbyssJewel {
					abyssJewelCount++
					item := xml.NewItem(t.itemIdGenerator, &si.Item)
					building.Items.ItemList = append(building.Items.ItemList, item)
					t.itemIdGenerator++
					siSlotName := fmt.Sprintf("%s Abyssal Socket %d", slotName, abyssJewelCount)
					slot := xml.NewEquipmentSlot(siSlotName, item.ID)
					slotSet.Append(slot)
				} else {
					gem := &si
					groupNum := sockets[gem.Socket].Group

					if i > 0 && groupNum != prevGroupNum {
						skills = append(skills, xml.NewSkill(slotName, group))
						group = []*api.Item{}
					}

					group = append(group, &gem.Item)
					prevGroupNum = groupNum
				}
			}
			if len(group) > 0 {
				skills = append(skills, xml.NewSkill(slotName, group))
			}
			building.Skills.SkillSet.Skills = skills
		}
	}
}

// getBuildingItemDataArray 返回所有建筑物品json数据
func (t *Transformer) getBuildingItemDataArray() []*api.Item {
	itemsJson := t.itemsData.Items
	var list []*api.Item
	for _, item := range itemsJson {
		switch *item.InventoryId {
		case "Weapon2", "Offhand2":
			if t.options != nil && t.options.SkipWeapon2 {
				continue
			}
		case "MainInventory": // 位于主背包的物品
			continue
		case "ExpandedMainInventory": // 位于扩展背包的物品
			continue
		}

		if item.BaseType == "THIEFS_TRINKET" {
			continue
		}
		list = append(list, item)
	}
	return list
}

func (t *Transformer) parseTree() {
	building := t.building
	character := t.itemsData.Character

	spec := building.Tree.Spec
	itemList := building.Items.ItemList
	for i := range t.passiveSkillsData.Items {
		itemData := &t.passiveSkillsData.Items[i]
		item := xml.NewItem(t.itemIdGenerator, itemData)
		itemList = append(itemList, item)
		t.itemIdGenerator++

		socket := xml.NewSocket(
			GetNodeIdOfExpansionSlot(*itemData.X),
			item.ID,
		)
		spec.Sockets.Append(socket)
	}
	building.Items.ItemList = itemList

	spec.ClassId = t.passiveSkillsData.Character
	spec.AscendClassId = t.passiveSkillsData.Ascendancy
	spec.SecondaryAscendClassId = t.passiveSkillsData.AlternateAscendancy

	if IsPhreciaAscendancy(character.Class) {
		spec.TreeVersion = "3_28_alternate"
	} else {
		spec.TreeVersion = "3_28"
	}

	masteryEffects := t.passiveSkillsData.MasteryEffects
	if masteryEffects.IsArray {
		// 空数组
	} else if masteryEffects.IsMap {
		for node, effect := range masteryEffects.Map {
			nodeId := 0
			for _, c := range node {
				nodeId = nodeId*10 + int(c-'0')
			}
			spec.MasteryEffects = append(spec.MasteryEffects, xml.NewMasteryEffect(nodeId, effect))
		}
	}

	spec.Nodes = t.passiveSkillsData.Hashes
	spec.Nodes = append(spec.Nodes, GetEnabledNodeIdsOfJewels(t.passiveSkillsData)...)

	spec.Overrides.Parse(t.passiveSkillsData.SkillOverrides)
}
