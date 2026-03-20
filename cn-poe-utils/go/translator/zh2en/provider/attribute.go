package provider

import (
	"github.com/cn-poe-community/cn-poe-utils/go/data/poe"
)

type AttributeProvider struct {
	zhIdx map[string]*poe.Attribute
}

func NewAttributeProvider(data *poe.Data) *AttributeProvider {
	zhIdx := make(map[string]*poe.Attribute)
	for i, item := range data.Attributes {
		zhIdx[item.Zh] = &data.Attributes[i]
	}
	return &AttributeProvider{
		zhIdx: zhIdx,
	}
}

func (p *AttributeProvider) ProvideByZh(zh string) *poe.Attribute {
	if attr, ok := p.zhIdx[zh]; ok {
		return attr
	}

	return nil
}
