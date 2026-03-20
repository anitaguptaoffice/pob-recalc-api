package zh2en

import (
	"log"
	"regexp"
	"strings"

	"github.com/cn-poe-community/cn-poe-utils/go/api"
)

const (
	zhThiefTrinket                       = "赏金猎人饰品"
	zhForbiddenFlesh                     = "禁断之肉"
	zhForbiddenFlame                     = "禁断之火"
	zhClassScion                         = "贵族"
	zhPassiveSkillAscendantAssassin      = "暗影"
	zhPassiveSkillAscendantAssassinFixed = "暗影（贵族）"
	zhRequirementNameClass               = "职业："
)

var enchantModRegex = regexp.MustCompile(`^元素伤害(提高|降低) \d+%$`)

// JsonTranslator JSON 翻译器
type JsonTranslator struct {
	basic *BasicTranslator
}

func NewJsonTranslator(basicTranslator *BasicTranslator) *JsonTranslator {
	return &JsonTranslator{
		basic: basicTranslator,
	}
}

// preHandleItem 翻译前预处理 Item
//
// 国服在本地化时，引入了一些错误，部分错误只能通过 hack 的方式进行解决
func (t *JsonTranslator) preHandleItem(item *api.Item) {
	if item.Name == zhForbiddenFlame || item.Name == zhForbiddenFlesh {
		if item.Requirements != nil {
			for _, requirement := range item.Requirements {
				name := requirement.Name

				if name != zhRequirementNameClass {
					continue
				}

				value := requirement.Values[0].Value
				// 禁断珠宝，其中贵族的升华大点 `暗影` 与暗影的升华大点 `暗影` 存在中文同名问题
				if value == zhClassScion {
					if item.ExplicitMods != nil {
						for i, zhStat := range item.ExplicitMods {
							if strings.HasSuffix(zhStat, zhPassiveSkillAscendantAssassin) {
								item.ExplicitMods[i] = strings.Replace(zhStat, zhPassiveSkillAscendantAssassin, zhPassiveSkillAscendantAssassinFixed, 1)
							}
						}
					}
				}

				break
			}
		}
	}

	// S26 赛季武器附魔引入的中文词缀重复问题
	if item.EnchantMods != nil {
		for i, mod := range item.EnchantMods {
			if enchantModRegex.MatchString(mod) {
				item.EnchantMods[i] = "该武器的" + mod
			}
		}
	}
}

// TransItems 翻译 items json 数据
//
// 本函数采用本地翻译，会修改原始对象
func (t *JsonTranslator) TransItems(items *api.GetItemsResult) {
	itemList := items.Items
	result := make([]*api.Item, 0, len(itemList))
	for i := range itemList {
		if t.isPobItem(itemList[i]) {
			t.TransItem(itemList[i])
			result = append(result, itemList[i])
		}
	}
	items.Items = result
}

func (t *JsonTranslator) isPobItem(item *api.Item) bool {
	if item.InventoryId != nil {
		invId := *item.InventoryId
		if invId == "MainInventory" || invId == "ExpandedMainInventory" {
			return false
		}
	}
	return item.BaseType != zhThiefTrinket
}

// TransItem 翻译 Item
//
// 本函数采用本地翻译，会修改原始对象
func (t *JsonTranslator) TransItem(item *api.Item) {
	t.preHandleItem(item)

	name := item.Name
	baseType := item.BaseType

	// 传奇物品、稀有物品的name不为""
	// 魔法物品、普通物品的name为""
	result := t.basic.TransNameAndBaseType(name, baseType)
	if result != nil {
		item.Name = result.Name
		item.BaseType = result.BaseType
	} else {
		log.Printf("untranslated: item name, %s\n", name)
		log.Printf("untranslated: item baseType, %s\n", baseType)
	}

	// 使用翻译后的baseType作为typeLine的翻译结果，性能最快
	// 但可能不满足某些需求
	item.TypeLine = item.BaseType

	if item.Requirements != nil {
		for i := range item.Requirements {
			req := &item.Requirements[i]
			name := req.Name
			result := t.basic.TransRequirementName(req.Name)
			if result != nil {
				req.Name = *result
			} else {
				log.Printf("untranslated: requirement name, %s\n", name)
			}

			if req.Values != nil {
				for j := range req.Values {
					v := &req.Values[j]
					value := v.Value
					result := t.basic.TransRequirement(name, value)
					if result != nil && result.Value != nil {
						v.Value = *result.Value
					}
				}
			}

			if req.Suffix != nil {
				suffix := *req.Suffix
				res := t.basic.TransRequirementSuffix(suffix)
				if res != nil {
					*req.Suffix = *res
				} else {
					log.Printf("untranslated: requirement suffix, %s\n", suffix)
				}
			}
		}
	}

	if item.Properties != nil {
		for i := range item.Properties {
			prop := &item.Properties[i]
			name := prop.Name
			value := t.basic.TransPropertyName(name)
			if value != nil {
				prop.Name = *value
			} else {
				log.Printf("untranslated: property name, %s\n", name)
			}

			if prop.Values != nil {
				for j := range prop.Values {
					v := &prop.Values[j]
					value := v.Value
					result := t.basic.TransProperty(name, value)
					if result != nil && result.Value != nil {
						v.Value = *result.Value
					}
				}
			}
		}
	}

	if item.SocketedItems != nil {
		for i := range item.SocketedItems {
			si := &item.SocketedItems[i]
			if si.AbyssJewel != nil && *si.AbyssJewel {
				t.TransItem(&si.Item)
			} else {
				t.transGem(&si.Item)
			}
		}
	}

	if item.EnchantMods != nil {
		for i, mod := range item.EnchantMods {
			result := t.basic.TransMod(mod)
			if result != nil {
				item.EnchantMods[i] = *result
			} else {
				log.Printf("untranslated: mod: %s\n", mod)
			}
		}
	}

	if item.ExplicitMods != nil {
		for i, mod := range item.ExplicitMods {
			result := t.basic.TransMod(mod)
			if result != nil {
				item.ExplicitMods[i] = *result
			} else {
				log.Printf("untranslated: mod: %s\n", mod)
			}
		}
	}

	if item.ImplicitMods != nil {
		for i, mod := range item.ImplicitMods {
			result := t.basic.TransMod(mod)
			if result != nil {
				item.ImplicitMods[i] = *result
			} else {
				log.Printf("untranslated: mod: %s\n", mod)
			}
		}
	}

	if item.CraftedMods != nil {
		for i, mod := range item.CraftedMods {
			result := t.basic.TransMod(mod)
			if result != nil {
				item.CraftedMods[i] = *result
			} else {
				log.Printf("untranslated: mod: %s\n", mod)
			}
		}
	}

	if item.UtilityMods != nil {
		for i, mod := range item.UtilityMods {
			result := t.basic.TransMod(mod)
			if result != nil {
				item.UtilityMods[i] = *result
			} else {
				log.Printf("untranslated: mod: %s\n", mod)
			}
		}
	}

	if item.FracturedMods != nil {
		for i, mod := range item.FracturedMods {
			result := t.basic.TransMod(mod)
			if result != nil {
				item.FracturedMods[i] = *result
			} else {
				log.Printf("untranslated: mod: %s\n", mod)
			}
		}
	}

	if item.ScourgeMods != nil {
		for i, mod := range item.ScourgeMods {
			result := t.basic.TransMod(mod)
			if result != nil {
				item.ScourgeMods[i] = *result
			} else {
				log.Printf("untranslated: mod: %s\n", mod)
			}
		}
	}

	if item.CrucibleMods != nil {
		for i, mod := range item.CrucibleMods {
			result := t.basic.TransMod(mod)
			if result != nil {
				item.CrucibleMods[i] = *result
			} else {
				log.Printf("untranslated: mod: %s\n", mod)
			}
		}
	}

	if item.MutatedMods != nil {
		for i, mod := range item.MutatedMods {
			result := t.basic.TransMod(mod)
			if result != nil {
				item.MutatedMods[i] = *result
			} else {
				log.Printf("untranslated: mod: %s\n", mod)
			}
		}
	}
}

func (t *JsonTranslator) transGem(gem *api.Item) {
	baseType := gem.BaseType
	typeLine := gem.TypeLine
	if baseType != "" {
		result := t.basic.TransSkill(baseType)
		if result != nil {
			gem.BaseType = *result
		} else {
			log.Printf("untranslated: gem baseType: %s\n", baseType)
		}
	}

	if typeLine != "" {
		result := t.basic.TransSkill(typeLine)
		if result != nil {
			gem.TypeLine = *result
		} else {
			log.Printf("untranslated: gem typeLine: %s\n", typeLine)
		}
	}

	if gem.Hybrid != nil {
		result := t.basic.TransSkill(gem.Hybrid.BaseTypeName)
		if result != nil {
			gem.Hybrid.BaseTypeName = *result
		} else {
			log.Printf("untranslated: gem hybrid baseTypeName: %s\n", gem.Hybrid.BaseTypeName)
		}
	}

	if gem.Properties != nil {
		for i := range gem.Properties {
			prop := &gem.Properties[i]
			result := t.basic.TransSkillProp(prop.Name)
			if result != nil {
				prop.Name = *result
			}
		}
	}
}

// TransPassiveSkills 翻译被动技能
func (t *JsonTranslator) TransPassiveSkills(skills *api.GetPassiveSkillsResult) {
	for i := range skills.Items {
		t.TransItem(&skills.Items[i])
	}

	for _, value := range skills.SkillOverrides {
		if value.Name != "" {
			name := value.Name
			if value.IsKeystone != nil && *value.IsKeystone {
				result := t.basic.TransKeystone(name)
				if result != nil {
					value.Name = *result
				} else {
					log.Printf("untranslated: keystone, %s\n", name)
				}
			} else {
				result := t.basic.TransBaseType(name)
				if result != nil {
					value.Name = *result
				} else {
					log.Printf("untranslated: baseType, %s\n", name)
				}
			}
		}
	}
}
