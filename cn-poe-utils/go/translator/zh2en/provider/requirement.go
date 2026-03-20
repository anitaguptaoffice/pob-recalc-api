package provider

import (
	"github.com/cn-poe-community/cn-poe-utils/go/data/poe"
)

type RequirementProvider struct {
	zhIdx       map[string]*poe.Requirement
	suffixZhIdx map[string]*poe.RequirementSuffix
}

func NewRequirementProvider(data *poe.Data) *RequirementProvider {
	zhIdx := make(map[string]*poe.Requirement)
	suffixZhIdx := make(map[string]*poe.RequirementSuffix)

	for i, item := range data.Requirements {
		zhIdx[item.Zh] = &data.Requirements[i]
	}
	for i, item := range data.RequirementSuffixes {
		suffixZhIdx[item.Zh] = &data.RequirementSuffixes[i]
	}

	return &RequirementProvider{
		zhIdx:       zhIdx,
		suffixZhIdx: suffixZhIdx,
	}
}

func (p *RequirementProvider) ProvideByZh(zh string) *poe.Requirement {
	if req, ok := p.zhIdx[zh]; ok {
		return req
	}
	return nil
}

func (p *RequirementProvider) ProvideSuffixByZh(zh string) *poe.RequirementSuffix {
	if suffix, ok := p.suffixZhIdx[zh]; ok {
		return suffix
	}
	return nil
}
