package api

import (
	"encoding/json"
	"fmt"
)

type Inventory struct {
	ExtraColumns int `json:"extra_columns"`
	// 金币
	Gold int `json:"gold"`
}

// Influences 影响
type Influences struct {
	Redeemer *bool `json:"redeemer,omitempty"`
	Shaper   *bool `json:"shaper,omitempty"`
	Elder    *bool `json:"elder,omitempty"`
	Crusader *bool `json:"crusader,omitempty"`
	Hunter   *bool `json:"hunter,omitempty"`
	Warlord  *bool `json:"warlord,omitempty"`
}

// SocketAttribute 插槽属性
//
// I: Int/智慧, D: Dex/敏捷, S: Str/力量, G: General/通用, A: Abyssal/深渊
type SocketAttribute string

const (
	SocketAttributeInt     SocketAttribute = "I"
	SocketAttributeDex     SocketAttribute = "D"
	SocketAttributeStr     SocketAttribute = "S"
	SocketAttributeGeneral SocketAttribute = "G"
	SocketAttributeAbyssal SocketAttribute = "A"
)

// SocketColour 插槽颜色
//
// B: Blue/蓝色, G: Green/绿色, R: Red/红色, W: White/白色, A: Abyssal/深渊
type SocketColour string

const (
	SocketColourBlue    SocketColour = "B"
	SocketColourGreen   SocketColour = "G"
	SocketColourRed     SocketColour = "R"
	SocketColourWhite   SocketColour = "W"
	SocketColourAbyssal SocketColour = "A"
)

// Rarity 稀有度
type Rarity string

const (
	RarityNormal Rarity = "Normal"
	RarityMagic  Rarity = "Magic"
	RarityRare   Rarity = "Rare"
	RarityUnique Rarity = "Unique"
)

// Socket 插槽
type Socket struct {
	Group   int             `json:"group"`
	Attr    SocketAttribute `json:"attr"`
	SColour SocketColour    `json:"sColour"`
}

// Value Requirements/Properties等类型的值
//
// 其 JSON 表示为 [string, int] 混合数组
type Value struct {
	Value string `json:"-"`
	Index int    `json:"-"`
}

func (v Value) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{v.Value, v.Index})
}

func (v *Value) UnmarshalJSON(data []byte) error {
	var arr [2]any
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	if str, ok := arr[0].(string); ok {
		v.Value = str
	} else {
		return fmt.Errorf("expected string at index 0, got %T", arr[0])
	}

	if num, ok := arr[1].(float64); ok {
		v.Index = int(num)
	} else if num, ok := arr[1].(int); ok {
		v.Index = num
	} else {
		return fmt.Errorf("expected number at index 1, got %T", arr[1])
	}

	return nil
}

// Requirement 需求
type Requirement struct {
	Name        string  `json:"name"`
	Values      []Value `json:"values"`
	DisplayMode int     `json:"displayMode"`
	Type        *int    `json:"type,omitempty"`
	Suffix      *string `json:"suffix,omitempty"`
}

// Property 属性
type Property struct {
	Name        string   `json:"name"`
	Values      []Value  `json:"values"`
	DisplayMode int      `json:"displayMode"`
	Type        *int     `json:"type,omitempty"`
	Progress    *float64 `json:"progress,omitempty"`
}

// Item 物品
type Item struct {
	// 仅限：深渊珠宝
	// 深渊珠宝必有该属性且为true
	AbyssJewel    *bool    `json:"abyssJewel,omitempty"`
	BaseType      string   `json:"baseType"`
	Corrupted     *bool    `json:"corrupted,omitempty"`
	CraftedMods   []string `json:"craftedMods,omitempty"`
	CrucibleMods  []string `json:"crucibleMods,omitempty"`
	DescrText     *string  `json:"descrText,omitempty"`
	Duplicated    *bool    `json:"duplicated,omitempty"`
	Elder         *bool    `json:"elder,omitempty"`
	EnchantMods   []string `json:"enchantMods,omitempty"`
	ExplicitMods  []string `json:"explicitMods,omitempty"`
	FlavourText   []string `json:"flavourText,omitempty"`
	FoilVariation *int     `json:"foilVariation,omitempty"`
	Fractured     *bool    `json:"fractured,omitempty"`
	FracturedMods []string `json:"fracturedMods,omitempty"`
	FrameType     int      `json:"frameType"`
	H             int      `json:"h"`
	// 仅限：宝石
	Hybrid        *Hybrid        `json:"hybrid,omitempty"`
	Icon          string         `json:"icon"`
	ID            *string        `json:"id,omitempty"`
	Identified    bool           `json:"identified"`
	Ilvl          int            `json:"ilvl"`
	ImplicitMods  []string       `json:"implicitMods,omitempty"`
	Influences    *Influences    `json:"influences,omitempty"`
	InventoryId   *string        `json:"inventoryId,omitempty"`
	IsRelic       *bool          `json:"isRelic,omitempty"`
	League        string         `json:"league"`
	Mutated       *bool          `json:"mutated,omitempty"`
	MutatedMods   []string       `json:"mutatedMods,omitempty"`
	Name          string         `json:"name"`
	Properties    []Property     `json:"properties,omitempty"`
	Rarity        *Rarity        `json:"rarity,omitempty"`
	Requirements  []Requirement  `json:"requirements,omitempty"`
	ScourgeMods   []string       `json:"scourgeMods,omitempty"`
	Searing       *bool          `json:"searing,omitempty"`
	SecDescrText  *string        `json:"secDescrText,omitempty"`
	Shaper        *bool          `json:"shaper,omitempty"`
	SocketedItems []SocketedItem `json:"socketedItems,omitempty"`
	Sockets       []Socket       `json:"sockets,omitempty"`
	Split         *bool          `json:"split,omitempty"`
	// 仅限：宝石
	// 表示宝石是辅助技能宝石
	Support     *bool    `json:"support,omitempty"`
	Synthesised *bool    `json:"synthesised,omitempty"`
	Tangled     *bool    `json:"tangled,omitempty"`
	TypeLine    string   `json:"typeLine"`
	UtilityMods []string `json:"utilityMods,omitempty"`
	Verified    bool     `json:"verified"`
	W           int      `json:"w"`
	X           *int     `json:"x,omitempty"`
	Y           *int     `json:"y,omitempty"`
}

// Hybrid 附带技能
type Hybrid struct {
	IsVaalGem    *bool      `json:"isVaalGem,omitempty"`
	BaseTypeName string     `json:"baseTypeName"`
	Properties   []Property `json:"properties,omitempty"`
	ExplicitMods []string   `json:"explicitMods"`
	SecDescrText string     `json:"secDescrText"`
}

// SocketedItem 插槽物品
type SocketedItem struct {
	Item
	// 仅限：宝石
	Colour *SocketAttribute `json:"colour,omitempty"`
	Socket int              `json:"socket"`
}

// GetItemsResult 获取物品列表的结果
type GetItemsResult struct {
	// 使用指针以获取最佳性能
	Items     []*Item    `json:"items"`
	Character Character  `json:"character"`
	Inventory *Inventory `json:"inventory,omitempty"`
}
