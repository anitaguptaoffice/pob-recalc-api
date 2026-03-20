package provider

import (
	"strings"

	"github.com/cn-poe-community/cn-poe-utils/go/data/poe"
	"github.com/cn-poe-community/cn-poe-utils/go/translator/zh2en/util"
)

const VARIABLE_PLACEHOLDER = "{0}"

type PropertyProvider struct {
	zhIdx         map[string]*poe.Property
	zhSkeletonIdx map[string][]*poe.Property
}

func NewPropertyProvider(data *poe.Data) *PropertyProvider {
	zhIdx := make(map[string]*poe.Property)
	zhSkeletonIdx := make(map[string][]*poe.Property)

	for i, item := range data.Properties {
		zhIdx[item.Zh] = &data.Properties[i]

		if strings.Contains(item.Zh, VARIABLE_PLACEHOLDER) {
			zhSkeletonIdx[util.GetTextSkeleton(item.Zh)] =
				append(zhSkeletonIdx[util.GetTextSkeleton(item.Zh)], &data.Properties[i])
		}
	}
	return &PropertyProvider{
		zhIdx:         zhIdx,
		zhSkeletonIdx: zhSkeletonIdx,
	}
}

func (p *PropertyProvider) ProvideByZh(zh string) *poe.Property {
	if prop, ok := p.zhIdx[zh]; ok {
		return prop
	}
	return nil
}

func (p *PropertyProvider) ProvideVariablePropertiesByZhSkeleton(skeleton string) []*poe.Property {
	if props, ok := p.zhSkeletonIdx[skeleton]; ok {
		return props
	}
	return nil
}
