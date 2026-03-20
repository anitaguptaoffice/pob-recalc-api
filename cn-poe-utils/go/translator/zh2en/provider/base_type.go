package provider

import (
	"github.com/cn-poe-community/cn-poe-utils/go/data/poe"
)

// BaseTypeProvider 基础类型提供者
type BaseTypeProvider struct {
	zhIdx map[string][]*poe.BaseType
}

func NewBaseTypeProvider(data *poe.Data) *BaseTypeProvider {
	// 所有基础类型
	list := make([][]poe.BaseType, 0)
	list = append(list, data.Amulets)
	list = append(list, data.Belts)
	list = append(list, data.BodyArmours)
	list = append(list, data.Boots)
	list = append(list, data.Flasks)
	list = append(list, data.Gloves)
	list = append(list, data.Helmets)
	list = append(list, data.Jewels)
	list = append(list, data.Quivers)
	list = append(list, data.Rings)
	list = append(list, data.Shields)
	list = append(list, data.Tattoos)
	list = append(list, data.Tinctures)
	list = append(list, data.Weapons)

	zhIdx := make(map[string][]*poe.BaseType)
	for _, baseTypes := range list {
		for i, baseType := range baseTypes {
			zhIdx[baseType.Zh] = append(zhIdx[baseType.Zh], &baseTypes[i])
		}
	}

	return &BaseTypeProvider{
		zhIdx: zhIdx,
	}
}

// ProvideByZh 根据中文基础类型名提供基础类型
func (p *BaseTypeProvider) ProvideByZh(zh string) []*poe.BaseType {
	if baseTypes, ok := p.zhIdx[zh]; ok {
		return baseTypes
	}
	return nil
}
