package util

import (
	"regexp"
)

// LINE_SEPARATOR 行分隔符
const LINE_SEPARATOR = "\n"

// GetTextSkeleton 获取文本骨架
//
// 移除所有数字、空格、点、加号、减号
func GetTextSkeleton(text string) string {
	re := regexp.MustCompile(`[{}0-9\s.+-]`)
	return re.ReplaceAllString(text, "")
}
