package building

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/cn-poe-community/cn-poe-utils/go/api"
	"github.com/cn-poe-community/cn-poe-utils/go/data/poe"
	"github.com/cn-poe-community/cn-poe-utils/go/translator/zh2en"
)

// loadTestData 加载POE翻译数据
func loadTestData(t *testing.T) poe.Data {
	data, err := os.ReadFile("../data/poe/testdata/all.json")
	if err != nil {
		t.Fatalf("无法读取翻译数据文件: %v", err)
	}

	var result poe.Data
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("反序列化翻译数据失败: %v", err)
	}
	return result
}

// loadItems 加载物品JSON数据
func loadItems(t *testing.T) *api.GetItemsResult {
	data, err := os.ReadFile("../api/testdata/items.json")
	if err != nil {
		t.Fatalf("无法读取物品数据文件: %v", err)
	}

	var result api.GetItemsResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("反序列化物品数据失败: %v", err)
	}
	return &result
}

// loadPassiveSkills 加载被动技能JSON数据
func loadPassiveSkills(t *testing.T) *api.GetPassiveSkillsResult {
	data, err := os.ReadFile("../api/testdata/passive_skills.json")
	if err != nil {
		t.Fatalf("无法读取被动技能数据文件: %v", err)
	}

	var result api.GetPassiveSkillsResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("反序列化被动技能数据失败: %v", err)
	}
	return &result
}

// TestIntegration 集成测试：从JSON数据到XML的完整流程
func TestIntegration(t *testing.T) {
	// 1. 加载翻译数据
	poeData := loadTestData(t)

	// 2. 创建翻译器
	basicTranslator := zh2en.NewBasicTranslator(&poeData)
	jsonTranslator := zh2en.NewJsonTranslator(basicTranslator)

	// 3. 加载物品和被动技能数据
	items := loadItems(t)
	passiveSkills := loadPassiveSkills(t)

	// 4. 翻译数据
	jsonTranslator.TransItems(items)
	jsonTranslator.TransPassiveSkills(passiveSkills)

	// 5. 转换为XML
	options := &TransformOptions{}
	pob := Transform(items, passiveSkills, options)

	// 6. 写入XML文件
	xmlContent := pob.String()
	err := os.WriteFile("building.xml", []byte(xmlContent), 0644)
	if err != nil {
		t.Fatalf("写入XML文件失败: %v", err)
	}

	t.Logf("成功生成 building.xml 文件，大小: %d 字节", len(xmlContent))
}
