package util

import (
	"fmt"
	"strconv"
	"unicode"
)

func ParseInt(s string) (int64, error) {
	// 找到第一个非数字字符的位置（包括负号处理）
	endIndex := len(s)
	for i, r := range s {
		if i == 0 && (r == '-' || r == '+') {
			continue // 允许第一个字符是正负号
		}
		if !unicode.IsDigit(r) {
			endIndex = i
			break
		}
	}

	// 截取有效部分
	numPart := s[:endIndex]
	if numPart == "" || numPart == "+" || numPart == "-" {
		return 0, fmt.Errorf("无效的数字格式: %s", numPart)
	}

	// 解析整数
	return strconv.ParseInt(numPart, 10, 64)
}

// ParseIntOrDefault 解析字符串为整数，失败则返回默认值
func ParseIntOrDefault(text string, def int) int {
	if text == "" {
		return def
	}
	num, err := ParseInt(text)
	if err != nil {
		return def
	}
	return int(num)
}

func MustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}
