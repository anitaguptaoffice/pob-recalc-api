package poe

// Attribute 属性
type Attribute struct {
	Zh     string           `json:"zh"`
	En     string           `json:"en"`
	Values []AttributeValue `json:"values,omitempty"`
}

// AttributeValue 属性值
type AttributeValue struct {
	Zh string `json:"zh"`
	En string `json:"en"`
}

// BaseType 基础类型
type BaseType struct {
	Zh      string   `json:"zh"`
	En      string   `json:"en"`
	Uniques []Unique `json:"uniques,omitempty"`
}

// Unique 传奇物品
type Unique struct {
	Zh string `json:"zh"`
	En string `json:"en"`
}

// Skill 技能
type Skill struct {
	Zh string `json:"zh"`
	En string `json:"en"`
}

// Node 节点
type Node struct {
	Zh string `json:"zh"`
	En string `json:"en"`
}

// Property 属性
type Property struct {
	Zh     string          `json:"zh"`
	En     string          `json:"en"`
	Values []PropertyValue `json:"values,omitempty"`
}

// PropertyValue 属性值
type PropertyValue struct {
	Zh string `json:"zh"`
	En string `json:"en"`
}

// Requirement 需求
type Requirement struct {
	Zh     string               `json:"zh"`
	En     string               `json:"en"`
	Values []RequirementValue   `json:"values,omitempty"`
}

// RequirementValue 需求值
type RequirementValue struct {
	Zh string `json:"zh"`
	En string `json:"en"`
}

// RequirementSuffix 需求后缀
type RequirementSuffix struct {
	Zh string `json:"zh"`
	En string `json:"en"`
}

// Stat 词缀
type Stat struct {
	Zh   string            `json:"zh"`
	En   string            `json:"en"`
	Refs map[string]string `json:"refs,omitempty"`
}

// ClientString 客户端字符串
type ClientString struct {
	Id   string `json:"id"`
	Zh   string `json:"zh"`
	En   string `json:"en"`
	Type string `json:"type"`
}

// Data 数据
type Data struct {
	// item types
	Amulets           []BaseType `json:"amulets"`
	Belts             []BaseType `json:"belts"`
	BodyArmours       []BaseType `json:"bodyArmours"`
	Boots             []BaseType `json:"boots"`
	Flasks            []BaseType `json:"flasks"`
	Gloves            []BaseType `json:"gloves"`
	Helmets           []BaseType `json:"helmets"`
	Jewels            []BaseType `json:"jewels"`
	Quivers           []BaseType `json:"quivers"`
	Rings             []BaseType `json:"rings"`
	Shields           []BaseType `json:"shields"`
	Tattoos           []BaseType `json:"tattoos"`
	Tinctures         []BaseType `json:"tinctures"`
	Weapons           []BaseType `json:"weapons"`
	// skills
	GemSkills         []Skill    `json:"gemSkills"`
	HybridSkills      []Skill    `json:"hybridSkills"`
	IndexableSupports []Skill    `json:"indexableSupports"`
	TransfiguredSkills []Skill   `json:"transfiguredSkills"`
	// passive skill nodes
	Anointed          []Node     `json:"anointed"`
	Ascendant         []Node     `json:"ascendant"`
	Keystones         []Node     `json:"keystones"`
	// stats
	Stats             []Stat     `json:"stats"`
	// others
	Attributes        []Attribute `json:"attributes"`
	Properties        []Property  `json:"properties"`
	Requirements      []Requirement `json:"requirements"`
	RequirementSuffixes []RequirementSuffix `json:"requirementSuffixes"`
	Strings           []ClientString `json:"strings"`
}
