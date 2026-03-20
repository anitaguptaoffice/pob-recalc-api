package pob

import "encoding/json"

// Clazz 职业
type Clazz struct {
	Name         string       `json:"name"`
	Ascendancies []Ascendancy `json:"ascendancies"`
}

// Ascendancy 升华
type Ascendancy struct {
	Name string `json:"name"`
}

// Node 节点
type Node struct {
	IsProxy        *bool           `json:"isProxy,omitempty"`
	IsJewelSocket  *bool           `json:"isJewelSocket,omitempty"`
	ExpansionJewel *ExpansionJewel `json:"expansionJewel,omitempty"`
	Orbit          int             `json:"orbit"`
	OrbitIndex     int             `json:"orbitIndex"`
}

// ExpansionJewel 扩展珠宝
type ExpansionJewel struct {
	Size   int     `json:"size"`
	Index  int     `json:"index"`
	Proxy  string  `json:"proxy"`
	Parent *string `json:"parent,omitempty"`
}

// Constants 常量
type Constants struct {
	Classes              map[string]int `json:"classes"`
	CharacterAttributes  map[string]int `json:"characterAttributes"`
	PSSCentreInnerRadius int            `json:"PSSCentreInnerRadius"`
	SkillsPerOrbit       []int          `json:"skillsPerOrbit"`
	OrbitRadii           []int          `json:"orbitRadii"`
}

// Tree 天赋树
type Tree struct {
	Classes    []Clazz      `json:"classes"`
	JewelSlots []int        `json:"jewelSlots"`
	Nodes      map[int]Node `json:"nodes"`
	Constants  Constants    `json:"constants"`
}

// ClusterJewelMetadata 星团珠宝元数据
type ClusterJewelMetadata struct {
	SizeIndex       int   `json:"sizeIndex"`
	NotableIndicies []int `json:"notableIndicies"`
	SocketIndicies  []int `json:"socketIndicies"`
	SmallIndicies   []int `json:"smallIndicies"`
	TotalIndicies   int   `json:"totalIndicies"`
}

type ClusterJewels struct {
	Jewels map[string]ClusterJewelMetadata `json:"jewels"`
}

// Skill 技能
type Skill struct {
	Zh string `json:"zh"`
	En string `json:"en"`
}

// Data 完整数据
type Data struct {
	Tree                 Tree              `json:"tree"`
	PhreciaAscendancyMap map[string]string `json:"phreciaAscendancyMap"`
	RarityMap            map[int]string    `json:"rarityMap"`
	SlotMap              map[string]string `json:"slotMap"`
	ClusterJewels        ClusterJewels     `json:"clusterJewels"`
	TransfiguredSkills   []Skill           `json:"transfiguredSkills"`
}

var DefaultData Data

func init() {
	if err := json.Unmarshal([]byte(dataStr), &DefaultData); err != nil {
		panic("DefaultData 反序列化失败: " + err.Error())
	}
}
