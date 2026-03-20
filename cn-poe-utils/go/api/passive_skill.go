package api

import (
	"encoding/json"
	"fmt"
)

// JewelData 珠宝数据
type JewelData map[string]JewelDatum

// JewelDatum 珠宝数据项
type JewelDatum struct {
	Type         string  `json:"type"`
	Radius       *int    `json:"radius,omitempty"`
	RadiusMin    *int    `json:"radiusMin,omitempty"`
	RadiusVisual *string `json:"radiusVisual,omitempty"`
	// 星团珠宝该字段非空
	Subgraph *Subgraph `json:"subgraph,omitempty"`
}

// Subgraph 子图
type Subgraph struct {
	Groups map[string]Expansion `json:"groups"`
	Nodes  map[int]Node         `json:"nodes"`
}

// Expansion 扩展
type Expansion struct {
	Proxy  string   `json:"proxy"`
	Nodes  []string `json:"nodes"`
	X      float64  `json:"x"`
	Y      float64  `json:"y"`
	Orbits []int    `json:"orbits"`
}

// Node 节点
type Node struct {
	Skill           string          `json:"skill"`
	Name            *string         `json:"name"`
	Icon            *string         `json:"icon"`
	IsMastery       *bool           `json:"isMastery,omitempty"`
	Stats           []string        `json:"stats"`
	Group           string          `json:"group"`
	Orbit           int             `json:"orbit"`
	OrbitIndex      int             `json:"orbitIndex"`
	Out             []string        `json:"out"`
	In              []string        `json:"in"`
	IsJewelSocket   *bool           `json:"isJewelSocket,omitempty"`
	ExpansionJewel  *ExpansionJewel `json:"expansionJewel,omitempty"`
	ReminderText    []string        `json:"reminderText,omitempty"`
	IsNotable       *bool           `json:"isNotable,omitempty"`
	GrantedStrength *int            `json:"grantedStrength,omitempty"`
}

type ExpansionJewel struct {
	Size   int    `json:"size"`
	Index  int    `json:"index"`
	Proxy  string `json:"proxy"`
	Parent string `json:"parent"`
}

// SkillOverride 技能覆盖
type SkillOverride struct {
	Name              string   `json:"name"`
	Icon              string   `json:"icon"`
	ActiveEffectImage string   `json:"activeEffectImage"`
	IsKeystone        *bool    `json:"isKeystone,omitempty"`
	IsTattoo          *bool    `json:"isTattoo,omitempty"`
	IsMastery         *bool    `json:"isMastery,omitempty"`
	Stats             []string `json:"stats"`
	ReminderText      []string `json:"reminderText,omitempty"`
	FlavorText        *string  `json:"flavorText,omitempty"`
	InactiveIcon      *string  `json:"inactiveIcon,omitempty"`
	ActiveIcon        *string  `json:"activeIcon,omitempty"`
}

// MasteryEffects 大师效果
type MasteryEffects struct {
	Map     map[string]int
	Array   []any
	IsMap   bool
	IsArray bool
}

func (m *MasteryEffects) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// 检查第一个字符
	switch data[0] {
	case '{':
		m.IsMap = true
		return json.Unmarshal(data, &m.Map)
	case '[':
		m.IsArray = true
		return json.Unmarshal(data, &m.Array)
	default:
		return fmt.Errorf("unexpected data type: %s", string(data[0]))
	}
}

// SkillOverrides 技能覆盖映射
type SkillOverrides map[string]*SkillOverride

// GetPassiveSkillsResult 获取被动技能的结果
type GetPassiveSkillsResult struct {
	Character           int            `json:"character"`
	Ascendancy          int            `json:"ascendancy"`
	AlternateAscendancy int            `json:"alternate_ascendancy"`
	Hashes              []int          `json:"hashes"`
	HashesEx            []int          `json:"hashes_ex"`
	MasteryEffects      MasteryEffects `json:"mastery_effects"`
	SkillOverrides      SkillOverrides `json:"skill_overrides"`
	Items               []Item         `json:"items"`
	JewelData           JewelData      `json:"jewel_data"`
}
