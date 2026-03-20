package provider

import (
	"github.com/cn-poe-community/cn-poe-utils/go/data/poe"
)

type StringProvider struct {
	idx map[string]*poe.ClientString
}

func NewStringProvider(data *poe.Data) *StringProvider {
	idx := make(map[string]*poe.ClientString)
	for i, item := range data.Strings {
		idx[item.Id] = &data.Strings[i]
	}
	return &StringProvider{
		idx: idx,
	}
}

func (p *StringProvider) MustProvide(id string) *poe.ClientString {
	if item, ok := p.idx[id]; ok {
		return item
	}
	panic("string not found of id: " + id)
}
