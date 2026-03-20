package util

import (
	"regexp"
	"strconv"
	"strings"
)

// PlaceholderInfo 占位符
type PlaceholderInfo struct {
	Index int // 占位符索引，如占位符为`{0}`，则索引为0
}

// Template 模板
type Template struct {
	template     string
	placeholders []PlaceholderInfo
	staticParts  []string
}

func NewTemplate(template string) *Template {
	t := &Template{
		template: template,
	}
	t.parseTemplate()
	return t
}

var placeHolderRegex = regexp.MustCompile(`\{(\d+)\}`)

// parseTemplate 解析模板，提取占位符信息
func (t *Template) parseTemplate() {
	matches := placeHolderRegex.FindAllStringSubmatchIndex(t.template, -1)

	if len(matches) == 0 {
		t.staticParts = []string{t.template}
		return
	}

	placeholders := make([]PlaceholderInfo, 0, len(matches))
	staticParts := make([]string, 0, len(matches)+1)

	lastIndex := 0
	for _, match := range matches {
		start := match[0]
		end := match[1]
		groupStart := match[2]
		groupEnd := match[3]

		// 收集占位符前的静态文本
		before := t.template[lastIndex:start]
		staticParts = append(staticParts, before)

		// 解析占位符索引（从捕获组中提取数字）
		placeholderIndex, err := strconv.Atoi(t.template[groupStart:groupEnd])
		if err != nil {
			panic(err)
		}

		// 存储占位符信息
		placeholders = append(placeholders, PlaceholderInfo{
			Index: placeholderIndex,
		})

		lastIndex = end
	}

	// 收集最后一个占位符后的静态文本
	lastPart := t.template[lastIndex:]
	staticParts = append(staticParts, lastPart)

	t.placeholders = placeholders
	t.staticParts = staticParts
}

// ParseParams 解析渲染结果，得到位置参数与实际参数值的映射表
//
// 如果渲染结果与模板不匹配，返回 nil 和 false
func (t *Template) ParseParams(str string) (map[int]string, bool) {
	// 如果没有占位符，直接返回空 map
	if len(t.placeholders) == 0 {
		return nil, false
	}

	result := make(map[int]string)
	strIndex := 0

	// 处理第一个静态部分
	firstStaticPart := t.staticParts[0]
	if !strings.HasPrefix(str, firstStaticPart) {
		return nil, false
	}
	strIndex += len(firstStaticPart)

	// 解析每个占位符
	for i := 0; i < len(t.placeholders); i++ {
		placeholder := t.placeholders[i]
		nextStaticPart := t.staticParts[i+1]

		// 查找下一个静态部分
		var endPos int

		if len(nextStaticPart) > 0 {
			// 有后续静态文本，匹配到它的位置
			idx := strings.Index(str[strIndex:], nextStaticPart)
			if idx == -1 {
				return nil, false
			}
			endPos = strIndex + idx
		} else {
			// 没有后续静态文本，匹配到字符串结束
			endPos = len(str)
		}

		// 提取参数值
		paramValue := str[strIndex:endPos]
		result[placeholder.Index] = paramValue

		// 跳过已匹配的部分
		strIndex = endPos + len(nextStaticPart)
	}

	// 验证整个字符串是否匹配完成
	if strIndex != len(str) {
		return nil, false
	}

	return result, true
}

// Render 渲染模板
func (t *Template) Render(paramMap map[int]string) string {
	if len(t.placeholders) == 0 {
		return t.template
	}

	// 构建渲染结果
	builder := make([]string, 0, len(t.placeholders)+len(t.staticParts))

	for i := 0; i < len(t.placeholders); i++ {
		placeholder := t.placeholders[i]
		builder = append(builder, t.staticParts[i])

		paramIndex := placeholder.Index
		if value, ok := paramMap[paramIndex]; ok {
			builder = append(builder, value)
		} else {
			panic("missing parameter for placeholder index " + strconv.Itoa(paramIndex) + "with template " + t.template)
		}
	}

	// 添加最后一个静态部分
	builder = append(builder, t.staticParts[len(t.staticParts)-1])

	return strings.Join(builder, "")
}

// GetTemplate 获取原始模板字符串
func (t *Template) GetTemplate() string {
	return t.template
}
