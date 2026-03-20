package util

import "github.com/cn-poe-community/cn-poe-utils/go/data/pob"

var transfiguredSkillSet = make(map[string]struct{})

func init() {
	for _, skill := range pob.DefaultData.TransfiguredSkills {
		transfiguredSkillSet[skill.En] = struct{}{}
	}
}

// IsTransfiguredSkill 判断是否为改造技能
func IsTransfiguredSkill(name string) bool {
	_, ok := transfiguredSkillSet[name]
	return ok
}
