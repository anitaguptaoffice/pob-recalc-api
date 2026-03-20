package util

import (
	"testing"
)

func TestGetTextSkeleton(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
		{
			name:     "纯数字",
			input:    "1234567890",
			expected: "",
		},
		{
			name:     "纯空格",
			input:    "   ",
			expected: "",
		},
		{
			name:     "带数字和空格的文本",
			input:    "Hello 123 World",
			expected: "HelloWorld",
		},
		{
			name:     "带点的文本",
			input:    "Hello.123.World",
			expected: "HelloWorld",
		},
		{
			name:     "带加号的文本",
			input:    "Hello+123+World",
			expected: "HelloWorld",
		},
		{
			name:     "带减号的文本",
			input:    "Hello-123-World",
			expected: "HelloWorld",
		},
		{
			name:     "带大括号的文本",
			input:    "Hello{123}World",
			expected: "HelloWorld",
		},
		{
			name:     "复杂混合文本",
			input:    "Hello 123.456+789-World{0}",
			expected: "HelloWorld",
		},
		{
			name:     "纯文本",
			input:    "HelloWorld",
			expected: "HelloWorld",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetTextSkeleton(tc.input)
			if result != tc.expected {
				t.Errorf("GetTextSkeleton(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
