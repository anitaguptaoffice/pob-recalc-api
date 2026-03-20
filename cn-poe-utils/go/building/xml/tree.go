package xml

import (
	"fmt"
	"strings"

	"github.com/cn-poe-community/cn-poe-utils/go/api"
)

// Tree 天赋树
type Tree struct {
	Spec *Spec
}

func NewTree() *Tree {
	return &Tree{
		Spec: NewSpec(),
	}
}

// String 返回XML字符串
func (t *Tree) String() string {
	return fmt.Sprintf(`<Tree activeSpec="1">
%s
</Tree>`, t.Spec.String())
}

// Spec 天赋树规格
type Spec struct {
	TreeVersion            string
	AscendClassId          int
	SecondaryAscendClassId int
	ClassId                int
	MasteryEffects         []*MasteryEffect
	Nodes                  []int
	Sockets                *Sockets
	Overrides              *Overrides
}

func NewSpec() *Spec {
	return &Spec{
		MasteryEffects: []*MasteryEffect{},
		Nodes:          []int{},
		Sockets:        NewSockets(),
		Overrides:      NewOverrides(),
	}
}

// String 返回XML字符串
func (s *Spec) String() string {
	var masteryEffectsView []string
	for _, me := range s.MasteryEffects {
		masteryEffectsView = append(masteryEffectsView, me.String())
	}
	nodesView := make([]string, len(s.Nodes))
	for i, n := range s.Nodes {
		nodesView[i] = fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf(`<Spec treeVersion="%s" ascendClassId="%d" secondaryAscendClassId="%d" classId="%d" masteryEffects="%s" nodes="%s">
%s
%s
</Spec>`,
		s.TreeVersion, s.AscendClassId, s.SecondaryAscendClassId, s.ClassId,
		strings.Join(masteryEffectsView, ","),
		strings.Join(nodesView, ","),
		s.Sockets.String(),
		s.Overrides.String())
}

// MasteryEffect 大师效果
type MasteryEffect struct {
	NodeId   int
	EffectId int
}

func NewMasteryEffect(nodeId int, effectId int) *MasteryEffect {
	return &MasteryEffect{
		NodeId:   nodeId,
		EffectId: effectId,
	}
}

// String 返回XML字符串
func (m *MasteryEffect) String() string {
	return fmt.Sprintf("{%d,%d}", m.NodeId, m.EffectId)
}

// Sockets 插槽集合
type Sockets struct {
	Sockets []*Socket
}

func NewSockets() *Sockets {
	return &Sockets{
		Sockets: []*Socket{},
	}
}

// Append 添加插槽
func (s *Sockets) Append(socket *Socket) {
	s.Sockets = append(s.Sockets, socket)
}

// String 返回XML字符串
func (s *Sockets) String() string {
	var socketsView []string
	for _, socket := range s.Sockets {
		socketsView = append(socketsView, socket.String())
	}
	return fmt.Sprintf(`<Sockets>
%s
</Sockets>`, strings.Join(socketsView, "\n"))
}

// Socket 插槽
type Socket struct {
	NodeId int
	ItemId int
}

func NewSocket(nodeId int, itemId int) *Socket {
	return &Socket{
		NodeId: nodeId,
		ItemId: itemId,
	}
}

// String 返回XML字符串
func (s *Socket) String() string {
	return fmt.Sprintf(`<Socket nodeId="%d" itemId="%d"/>`, s.NodeId, s.ItemId)
}

// Overrides 覆盖集合
type Overrides struct {
	Members []*Override
}

func NewOverrides() *Overrides {
	return &Overrides{
		Members: []*Override{},
	}
}

// Parse 解析技能覆盖
func (o *Overrides) Parse(skillOverrides api.SkillOverrides) {
	o.Members = []*Override{}
	for key, value := range skillOverrides {
		o.Members = append(o.Members, NewOverride(key, value))
	}
}

// String 返回XML字符串
func (o *Overrides) String() string {
	var membersView []string
	for _, member := range o.Members {
		membersView = append(membersView, member.String())
	}
	return fmt.Sprintf(`<Overrides>
%s
</Overrides>`, strings.Join(membersView, "\n"))
}

// Override 覆盖
type Override struct {
	Dn     string
	NodeId string
}

func NewOverride(nodeId string, json *api.SkillOverride) *Override {
	return &Override{
		Dn:     json.Name,
		NodeId: nodeId,
	}
}

// String 返回XML字符串
func (o *Override) String() string {
	return fmt.Sprintf(`<Override dn="%s" nodeId="%s">
</Override>`, o.Dn, o.NodeId)
}
