package provider

import (
	"github.com/cn-poe-community/cn-poe-utils/go/data/poe"
)

type SkillProvider struct {
	zhIdx                 map[string]*poe.Skill
	indexableSupportZhIdx map[string]*poe.Skill
}

func NewSkillProvider(data *poe.Data) *SkillProvider {
	zhIdx := make(map[string]*poe.Skill)
	indexableSupportZhIdx := make(map[string]*poe.Skill)

	for i, item := range data.GemSkills {
		zhIdx[item.Zh] = &data.GemSkills[i]
	}
	for i, item := range data.HybridSkills {
		zhIdx[item.Zh] = &data.HybridSkills[i]
	}
	for i, item := range data.TransfiguredSkills {
		zhIdx[item.Zh] = &data.TransfiguredSkills[i]
	}

	for i, item := range data.IndexableSupports {
		indexableSupportZhIdx[item.Zh] = &data.IndexableSupports[i]
	}
	return &SkillProvider{
		zhIdx:                 zhIdx,
		indexableSupportZhIdx: indexableSupportZhIdx,
	}
}

func (p *SkillProvider) ProvideSkill(name string) *poe.Skill {
	if skill, ok := p.zhIdx[name]; ok {
		return skill
	}
	return nil
}

func (p *SkillProvider) ProvideIndexableSupport(name string) *poe.Skill {
	if skill, ok := p.indexableSupportZhIdx[name]; ok {
		return skill
	}
	return nil
}
