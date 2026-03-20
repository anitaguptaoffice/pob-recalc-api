package xml

import (
	"fmt"
	"strings"
)

// Slot 插槽
type Slot struct {
	Name      string
	ItemPbURL string
	ItemId    *int
	NodeId    *int
	Active    bool
}

// NewEquipmentSlot 创建装备插槽
func NewEquipmentSlot(name string, itemId int) *Slot {
	slot := &Slot{
		Name:   name,
		ItemId: &itemId,
		Active: false,
	}
	if len(slot.Name) > 6 && slot.Name[:6] == "Flask " {
		slot.Active = true
	}
	return slot
}

// NewJewelSlot 创建珠宝插槽
func NewJewelSlot(name string, nodeId int) *Slot {
	return &Slot{
		Name:   name,
		NodeId: &nodeId,
		Active: false,
	}
}

// String 返回XML字符串
func (s *Slot) String() string {
	var builder []string
	builder = append(builder, fmt.Sprintf(`<Slot itemPbURL="%s"`, s.ItemPbURL))
	if s.Active {
		builder = append(builder, fmt.Sprintf(` active="%t"`, s.Active))
	}
	builder = append(builder, fmt.Sprintf(` name="%s"`, s.Name))
	if s.ItemId != nil {
		builder = append(builder, fmt.Sprintf(` itemId="%d"`, *s.ItemId))
	}
	if s.NodeId != nil {
		builder = append(builder, fmt.Sprintf(` nodeId="%d"`, *s.NodeId))
	}
	builder = append(builder, "/>")
	return strings.Join(builder, "")
}

// SocketIdURL SocketIdURL
type SocketIdURL struct {
	NodeId    int
	Name      string
	ItemPbURL string
}

// String 返回XML字符串
func (s *SocketIdURL) String() string {
	return fmt.Sprintf(`<SocketIdURL nodeId="%d" name="%s" itemPbURL="%s"/>`, s.NodeId, s.Name, s.ItemPbURL)
}

// ItemSet 物品集合
type ItemSet struct {
	UseSecondWeaponSet bool
	Id                 int
	Slots              []fmt.Stringer
}

func NewItemSet() *ItemSet {
	return &ItemSet{
		UseSecondWeaponSet: false,
		Id:                 1,
		Slots:              []fmt.Stringer{},
	}
}

// Append 添加插槽
func (is *ItemSet) Append(item fmt.Stringer) {
	is.Slots = append(is.Slots, item)
}

// String 返回XML字符串
func (is *ItemSet) String() string {
	var slotsView []string
	for _, slot := range is.Slots {
		slotsView = append(slotsView, slot.String())
	}
	return fmt.Sprintf(`<ItemSet useSecondWeaponSet="%t" id="%d">
%s
</ItemSet>`, is.UseSecondWeaponSet, is.Id, strings.Join(slotsView, "\n"))
}
