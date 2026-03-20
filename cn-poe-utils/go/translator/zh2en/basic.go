package zh2en

import (
	"bytes"
	"strings"

	"github.com/cn-poe-community/cn-poe-utils/go/data/poe"
	"github.com/cn-poe-community/cn-poe-utils/go/translator/zh2en/provider"
	"github.com/cn-poe-community/cn-poe-utils/go/translator/zh2en/util"
)

const DEFAULT_RARITY_ITEM_NAME = "Item"

var GEM_PROPERTY_MAP = map[string]string{
	"等级": "Level",
	"品质": "Quality",
}

// BasicTranslator 基础翻译器
type BasicTranslator struct {
	attributeProvider    *provider.AttributeProvider
	baseTypeProvider     *provider.BaseTypeProvider
	passiveSkillProvider *provider.PassiveSkillProvider
	propertyProvider     *provider.PropertyProvider
	requirementProvider  *provider.RequirementProvider
	skillProvider        *provider.SkillProvider
	statProvider         *provider.StatProvider
	stringProvider       *provider.StringProvider

	qualityPrefix           *poe.ClientString
	synthesisedPrefix       *poe.ClientString
	mutatedUniqueNamePrefix *poe.ClientString
	influenceStatPrefix1    *poe.ClientString
	influenceStatPrefix2    *poe.ClientString
}

// NewBasicTranslator 创建基础翻译器
func NewBasicTranslator(data *poe.Data) *BasicTranslator {
	attributeProvider := provider.NewAttributeProvider(data)
	baseTypeProvider := provider.NewBaseTypeProvider(data)
	passiveSkillProvider := provider.NewPassiveSkillProvider(data)
	propertyProvider := provider.NewPropertyProvider(data)
	requirementProvider := provider.NewRequirementProvider(data)
	skillProvider := provider.NewSkillProvider(data)
	statProvider := provider.NewStatProvider(data)
	stringProvider := provider.NewStringProvider(data)

	return &BasicTranslator{
		attributeProvider:    attributeProvider,
		baseTypeProvider:     baseTypeProvider,
		passiveSkillProvider: passiveSkillProvider,
		propertyProvider:     propertyProvider,
		requirementProvider:  requirementProvider,
		skillProvider:        skillProvider,
		statProvider:         statProvider,
		stringProvider:       stringProvider,

		qualityPrefix:           stringProvider.MustProvide("QualityPrefix"),
		synthesisedPrefix:       stringProvider.MustProvide("SynthesisedPrefix"),
		mutatedUniqueNamePrefix: stringProvider.MustProvide("MutatedUniqueNamePrefix"),
		influenceStatPrefix1:    stringProvider.MustProvide("InfluenceStatPrefix1"),
		influenceStatPrefix2:    stringProvider.MustProvide("InfluenceStatPrefix2"),
	}
}

type TransAttrResult struct {
	Name  string
	Value *string
}

// TransAttr 翻译属性
//
// 返回的 name 为空，则 value 也为空；返回的 name 非空，value 不一定非空
//
// 比如`物品等级: 100`，翻译结果为`Item Level: nil`，因为`100`并非数据库中存在对应翻译的值
func (t *BasicTranslator) TransAttr(name string, value string) *TransAttrResult {
	attr := t.attributeProvider.ProvideByZh(name)
	if attr == nil {
		return nil
	}

	if value != "" && len(attr.Values) > 0 {
		for _, v := range attr.Values {
			if v.Zh == value {
				return &TransAttrResult{
					attr.En,
					&v.En,
				}
			}
		}
	}

	return &TransAttrResult{
		attr.En,
		nil,
	}
}

// TransAttrName 翻译属性名
func (t *BasicTranslator) TransAttrName(name string) *string {
	attr := t.TransAttr(name, "")
	if attr == nil {
		return nil
	}
	return &attr.Name
}

type TransNameAndBaseTypeResult struct {
	Name     string
	BaseType string
}

// TransNameAndBaseType 翻译名称和基础类型
//
// 如果返回的名称不为空，则基础类型也不为空
func (t *BasicTranslator) TransNameAndBaseType(name string, baseType string) *TransNameAndBaseTypeResult {
	baseTypes := t.baseTypeProvider.ProvideByZh(baseType)
	if len(baseTypes) == 0 {
		return nil
	}

	if name != "" {
		uniqueNamePrefix := ""
		// 处理秽生传奇名称前缀
		if strings.HasPrefix(name, t.mutatedUniqueNamePrefix.Zh) {
			uniqueNamePrefix = t.mutatedUniqueNamePrefix.En
			name = name[len(t.mutatedUniqueNamePrefix.Zh):]
		}

		result := t.findUnique(baseTypes, name)
		// 传奇物品
		if result != nil {
			return &TransNameAndBaseTypeResult{
				uniqueNamePrefix + result.Unique.En,
				result.BaseType.En,
			}
		}
		// 稀有物品
		return &TransNameAndBaseTypeResult{
			DEFAULT_RARITY_ITEM_NAME,
			baseTypes[0].En,
		}
	}
	// 魔法物品或普通物品
	return &TransNameAndBaseTypeResult{
		name,
		baseTypes[0].En,
	}
}

// FindUniqueResult 查找传奇物品结果
//
// 使用指针只是为了方便，两者都不为nil
type findUniqueResult struct {
	BaseType *poe.BaseType
	Unique   *poe.Unique
}

// FindUnique 查找传奇物品
func (t *BasicTranslator) findUnique(baseTypes []*poe.BaseType, name string) *findUniqueResult {
	for _, b := range baseTypes {
		if len(b.Uniques) > 0 {
			for _, u := range b.Uniques {
				if u.Zh == name {
					return &findUniqueResult{
						b,
						&u,
					}
				}
			}
		}
	}
	return nil
}

// TransBaseType 翻译基础类型
func (t *BasicTranslator) TransBaseType(baseType string) *string {
	baseTypes := t.baseTypeProvider.ProvideByZh(baseType)
	if len(baseTypes) == 0 {
		return nil
	}
	return &baseTypes[0].En
}

var andBytes = []byte("的")
var ofBytes = []byte("之")

// FindBaseTypeFromTypeLine 根据 typeLine 推断 BaseType
//
// name 用于匹配传奇，否则返回首个匹配的 BaseType
func (t *BasicTranslator) FindBaseTypeFromTypeLine(typeLine string, name string) *poe.BaseType {
	// 传奇物品、稀有物品
	if name != "" {
		// 秽生传奇
		name = strings.TrimLeft(name, t.mutatedUniqueNamePrefix.Zh)
		// 忆境物品
		typeLine = strings.TrimLeft(typeLine, t.synthesisedPrefix.Zh)

		baseTypes := t.baseTypeProvider.ProvideByZh(typeLine)
		if len(baseTypes) > 0 {
			result := t.findUnique(baseTypes, name)
			if result != nil {
				return result.BaseType
			}
			return baseTypes[0]
		}

		return nil
	}

	// 魔法物品、普通物品、未鉴定传奇物品、未鉴定稀有物品

	typeLine = strings.TrimLeft(typeLine, t.qualityPrefix.Zh)
	typeLine = strings.TrimLeft(typeLine, t.synthesisedPrefix.Zh)

	// 先检查完整匹配的情况
	baseTypes := t.baseTypeProvider.ProvideByZh(typeLine)
	if len(baseTypes) > 0 {
		return baseTypes[0]
	}

	// 处理修饰词存在的情况
	//
	// 如“显著的幼龙之大型星团珠宝”，其修饰词为：“显著的”、“幼龙之”，其zhBaseType为“大型星团珠宝”。
	//
	// 修饰词以`的`、`之`结尾，但`的`、`之`同时可能出现在 baseType 中，如`潜能之戒`。
	// 我们可以逐步去除修饰词，来检测剩余部分是否是一个 baseType
	end := len(typeLine)
	typeLineBytes := []byte(typeLine)
	lastIndex := 0
	andPos, ofPos := bytes.Index(typeLineBytes, andBytes), bytes.Index(typeLineBytes, ofBytes)
	// 未找到时，将 pos 设置为 end，这样我们可以直接使用min(andPos,ofPos)来确定最近的修饰词的位置
	// 同时，这与我们在匹配结束后，只更新最近找到的修饰词的位置的算法完美适配
	if andPos == -1 {
		andPos = end
	}
	if ofPos == -1 {
		ofPos = end
	}
	for lastIndex < end {
		pos := min(andPos, ofPos)
		if pos == end {
			break
		}

		if andPos < ofPos {
			lastIndex = pos + len(andBytes)
		} else {
			lastIndex = pos + len(ofBytes)
		}

		baseTypes := t.baseTypeProvider.ProvideByZh(typeLine[lastIndex:])
		if len(baseTypes) > 0 {
			return baseTypes[0]
		}

		// 如果最近的修饰词是“的”，则查找下一个“的”，否则查找下一个“之”
		if andPos < ofPos {
			pos := bytes.Index(typeLineBytes[lastIndex:], andBytes)
			if pos == -1 {
				andPos = end
			} else {
				andPos = lastIndex + pos
			}
		} else {
			pos := strings.Index(typeLine[lastIndex+ofPos+1:], "之")
			if pos == -1 {
				ofPos = end
			} else {
				ofPos = lastIndex + pos
			}
		}
	}

	return nil
}

type TransNameAndTypeLineResult struct {
	Name     string
	TypeLine string
}

// TransNameAndTypeLine 翻译名称和 typeLine
func (t *BasicTranslator) TransNameAndTypeLine(name string, typeLine string) *TransNameAndTypeLineResult {
	// 传奇物品、稀有物品
	if name != "" {
		uniqueNamePrefix := ""
		typeLinePrefix := ""

		if strings.HasPrefix(name, t.mutatedUniqueNamePrefix.Zh) {
			uniqueNamePrefix = t.mutatedUniqueNamePrefix.En
			name = name[len(t.mutatedUniqueNamePrefix.Zh):]
		}

		if strings.HasPrefix(typeLine, t.synthesisedPrefix.Zh) {
			typeLine = typeLine[len(t.synthesisedPrefix.Zh):]
			typeLinePrefix += t.synthesisedPrefix.En
		}
		baseType := t.FindBaseTypeFromTypeLine(typeLine, name)
		if baseType != nil {
			if len(baseType.Uniques) > 0 {
				for _, u := range baseType.Uniques {
					if u.Zh == name {
						return &TransNameAndTypeLineResult{
							Name:     uniqueNamePrefix + u.En,
							TypeLine: typeLinePrefix + baseType.En,
						}
					}
				}
			}
			return &TransNameAndTypeLineResult{
				Name:     DEFAULT_RARITY_ITEM_NAME,
				TypeLine: typeLinePrefix + baseType.En,
			}
		}

		return nil
	}
	// 魔法物品、普通物品、未鉴定稀有物品、未鉴定传奇物品
	typeLinePrefix := ""

	if strings.HasPrefix(typeLine, t.qualityPrefix.Zh) {
		typeLine = typeLine[len(t.qualityPrefix.Zh):]
		typeLinePrefix = t.qualityPrefix.En
	}

	if strings.HasPrefix(typeLine, t.synthesisedPrefix.Zh) {
		typeLine = typeLine[len(t.synthesisedPrefix.Zh):]
		typeLinePrefix += t.synthesisedPrefix.En
	}
	baseType := t.FindBaseTypeFromTypeLine(typeLine, name)
	if baseType != nil {
		return &TransNameAndTypeLineResult{
			Name:     "",
			TypeLine: typeLinePrefix + baseType.En,
		}
	}

	return nil
}

// TransSkill 翻译技能和辅助技能
func (t *BasicTranslator) TransSkill(name string) *string {
	if skill := t.skillProvider.ProvideSkill(name); skill != nil {
		return &skill.En
	}
	return nil
}

// TransSkillProp 翻译技能属性
func (t *BasicTranslator) TransSkillProp(name string) *string {
	if val, ok := GEM_PROPERTY_MAP[name]; ok {
		return &val
	}
	return nil
}

// TransIndexableSupports 翻译可索引的辅助技能
func (t *BasicTranslator) TransIndexableSupports(name string) *string {
	if skill := t.skillProvider.ProvideIndexableSupport(name); skill != nil {
		return &skill.En
	}
	return nil
}

// TransAnointed 翻译可涂油天赋
func (t *BasicTranslator) TransAnointed(name string) *string {
	if node := t.passiveSkillProvider.ProvideAnointedByZh(name); node != nil {
		return &node.En
	}
	return nil
}

// TransKeystone 翻译基石天赋
func (t *BasicTranslator) TransKeystone(name string) *string {
	if node := t.passiveSkillProvider.ProvideKeystoneByZh(name); node != nil {
		return &node.En
	}
	return nil
}

// TransAscendant 翻译升华天赋
func (t *BasicTranslator) TransAscendant(zh string) *string {
	if node := t.passiveSkillProvider.ProvideAscendantByZh(zh); node != nil {
		return &node.En
	}
	return nil
}

type TransPropertyResult struct {
	Name  string
	Value *string
}

// TransProperty 翻译属性
func (t *BasicTranslator) TransProperty(name string, value string) *TransPropertyResult {
	prop := t.propertyProvider.ProvideByZh(name)
	if prop == nil {
		return nil
	}

	enName := prop.En
	if value != "" && len(prop.Values) > 0 {
		for _, v := range prop.Values {
			if v.Zh == value {
				return &TransPropertyResult{
					Name:  enName,
					Value: &v.En,
				}
			}
		}
	}

	return &TransPropertyResult{
		Name:  enName,
		Value: nil,
	}
}

// TransPropertyName 翻译属性名
func (t *BasicTranslator) TransPropertyName(name string) *string {
	prop := t.propertyProvider.ProvideByZh(name)
	if prop != nil {
		return &prop.En
	}

	skeleton := util.GetTextSkeleton(name)
	props := t.propertyProvider.ProvideVariablePropertiesByZhSkeleton(skeleton)
	if len(props) > 0 {
		for _, prop := range props {
			zhTmpl := util.NewTemplate(prop.Zh)
			posParams, ok := zhTmpl.ParseParams(name)
			if !ok {
				continue
			}

			enTmpl := util.NewTemplate(prop.En)
			result := enTmpl.Render(posParams)
			return &result
		}
	}

	return nil
}

type TransRequirementResult struct {
	Name  string
	Value *string
}

// TransRequirement 翻译需求
func (t *BasicTranslator) TransRequirement(name string, value string) *TransRequirementResult {
	r := t.requirementProvider.ProvideByZh(name)
	if r == nil {
		return nil
	}

	enName := r.En
	if value != "" && len(r.Values) > 0 {
		for _, v := range r.Values {
			if v.Zh == value {
				return &TransRequirementResult{
					Name:  enName,
					Value: &v.En,
				}
			}
		}
	}

	return &TransRequirementResult{
		Name:  enName,
		Value: nil,
	}
}

// TransRequirementName 翻译需求名
func (t *BasicTranslator) TransRequirementName(zhName string) *string {
	r := t.requirementProvider.ProvideByZh(zhName)
	if r == nil {
		return nil
	}
	return &r.En
}

// TransRequirementSuffix 翻译需求后缀
func (t *BasicTranslator) TransRequirementSuffix(suffix string) *string {
	s := t.requirementProvider.ProvideSuffixByZh(suffix)
	if s == nil {
		return nil
	}
	return &s.En
}

// TransMod 翻译词缀
func (t *BasicTranslator) TransMod(zhMod string) *string {
	if strings.HasPrefix(zhMod, t.influenceStatPrefix1.Zh) {
		subMod := t.transModInner(zhMod[len(t.influenceStatPrefix1.Zh):])
		if subMod != nil {
			result := t.influenceStatPrefix1.En + *subMod
			return &result
		}
	} else if strings.HasPrefix(zhMod, t.influenceStatPrefix2.Zh) {
		subMod := t.transModInner(zhMod[len(t.influenceStatPrefix2.Zh):])
		if subMod != nil {
			result := t.influenceStatPrefix2.En + *subMod
			return &result
		}
	}

	return t.transModInner(zhMod)
}

// transModInner 内部翻译词缀
func (t *BasicTranslator) transModInner(zhMod string) *string {
	skeleton := util.GetTextSkeleton(zhMod)
	stats := t.statProvider.ProvideByZhSkeleton(skeleton)

	if len(stats) > 0 {
		for _, stat := range stats {
			result := t.doTransMod(stat, zhMod)
			if result != nil {
				return result
			}
		}
	} else {
		referenceStats := t.statProvider.ProvideReferenceStats()
		for _, stat := range referenceStats {
			result := t.doTransMod(stat, zhMod)
			if result != nil {
				return result
			}
		}
	}

	return nil
}

// transStatParams 翻译词缀参数
func (t *BasicTranslator) transStatParams(stat *poe.Stat, posParams map[int]string) {
	// 引用的参数值需要进行翻译
	if stat.Refs != nil {
		for key, refType := range stat.Refs {
			pos := 0
			// 简单的字符串转整数
			for _, c := range key {
				if c >= '0' && c <= '9' {
					pos = pos*10 + int(c-'0')
				} else {
					break
				}
			}

			switch refType {
			case "anointed_passive":
				if zh, ok := posParams[pos]; ok {
					en := t.TransAnointed(zh)
					if en != nil {
						posParams[pos] = *en
					}
				}
			case "keystone_passive":
				if zh, ok := posParams[pos]; ok {
					en := t.TransKeystone(zh)
					if en != nil {
						posParams[pos] = *en
					}
				}
			case "ascendant_passive":
				if zh, ok := posParams[pos]; ok {
					en := t.TransAscendant(zh)
					if en != nil {
						posParams[pos] = *en
					}
				}
			case "display_indexable_support":
				if zh, ok := posParams[pos]; ok {
					en := t.TransIndexableSupports(zh)
					if en != nil {
						posParams[pos] = *en
					}
				}
			case "display_indexable_skill":
				if zh, ok := posParams[pos]; ok {
					en := t.TransSkill(zh)
					if en != nil {
						posParams[pos] = *en
					}
				}
			}
		}
	}
}

// doTransMod 执行词缀翻译
func (t *BasicTranslator) doTransMod(stat *poe.Stat, zhMod string) *string {
	if zhMod == stat.Zh {
		return &stat.En
	}

	zhTmpl := util.NewTemplate(stat.Zh)
	posParams, ok := zhTmpl.ParseParams(zhMod)
	// 不匹配
	if !ok {
		return nil
	}

	t.transStatParams(stat, posParams)

	enTmpl := util.NewTemplate(stat.En)
	result := enTmpl.Render(posParams)
	return &result
}

type TransMultilineModResult struct {
	Mod      string
	LineSize int
}

// TransMultilineMod 翻译多行词缀
func (t *BasicTranslator) TransMultilineMod(lines []string) *TransMultilineModResult {
	if len(lines) == 0 {
		return nil
	}

	skeleton := util.GetTextSkeleton(lines[0])
	group := t.statProvider.ProvideByFirstLineZhSkeleton(skeleton)
	if group != nil {
		for _, multilineStat := range group.Stats {
			lineSize := multilineStat.LineSize
			if lineSize > len(lines) {
				continue
			}
			stat := multilineStat.Stat
			mod := strings.Join(lines[:lineSize], util.LINE_SEPARATOR)

			if util.GetTextSkeleton(stat.Zh) == util.GetTextSkeleton(mod) {
				result := t.doTransMod(stat, mod)
				if result != nil {
					return &TransMultilineModResult{
						Mod:      *result,
						LineSize: lineSize,
					}
				}
			}
		}
	} else {
		stats := t.statProvider.ProvideMultilineReferenceStats()
		for _, stat := range stats {
			if len(lines) < stat.LineSize {
				continue
			}
			result := t.doTransMod(stat.Stat, strings.Join(lines[:stat.LineSize], util.LINE_SEPARATOR))
			if result != nil {
				return &TransMultilineModResult{
					Mod:      *result,
					LineSize: stat.LineSize,
				}
			}
		}
	}

	return nil
}
