package xml

import "fmt"

// PathOfBuilding POB主结构
type PathOfBuilding struct {
	Build  *Build
	Skills *Skills
	Tree   *Tree
	Items  *Items
	Config *Config
}

func NewPathOfBuilding() *PathOfBuilding {
	return &PathOfBuilding{
		Build:  NewBuild(),
		Skills: NewSkills(),
		Tree:   NewTree(),
		Items:  NewItems(),
		Config: NewConfig(),
	}
}

// String 返回XML字符串
func (p *PathOfBuilding) String() string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<PathOfBuilding>
%s
%s
%s
%s
%s
</PathOfBuilding>`,
		p.Build.String(),
		p.Skills.String(),
		p.Tree.String(),
		p.Items.String(),
		p.Config.String())
}
