// Package building 提供POB构建转换功能
package building

import (
	"github.com/cn-poe-community/cn-poe-utils/go/api"
	"github.com/cn-poe-community/cn-poe-utils/go/building/transform"
	"github.com/cn-poe-community/cn-poe-utils/go/building/xml"
)

// TransformOptions 转换选项
type TransformOptions = transform.TransformOptions

// Transform 将POE API数据转换为PathOfBuilding XML
func Transform(
	items *api.GetItemsResult,
	passiveSkills *api.GetPassiveSkillsResult,
	options *TransformOptions,
) *xml.PathOfBuilding {
	t := transform.NewTransformer(items, passiveSkills, options)
	t.Transform()
	return t.GetBuilding()
}
